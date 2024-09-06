package database

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/leanovate/gopter/prop"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/blackcloro/transaction-processor/internal"
	"github.com/blackcloro/transaction-processor/internal/domain/transaction"
	"github.com/blackcloro/transaction-processor/internal/testutil"
)

type PostgresTransactionRepositoryTestSuite struct {
	suite.Suite
	ctx         context.Context
	pgContainer *testutil.PostgresContainer
	repo        *PostgresTransactionRepository
}

func TestPostgresTransactionRepositorySuite(t *testing.T) {
	suite.Run(t, new(PostgresTransactionRepositoryTestSuite))
}

func (s *PostgresTransactionRepositoryTestSuite) SetupSuite() {
	s.ctx = context.Background()
	var err error
	s.pgContainer, err = testutil.NewPostgresContainer(s.ctx, testutil.PostgresConfig{
		User: "test_user", Password: "test_password", DBName: "test_db",
	})
	require.NoError(s.T(), err)

	err = s.pgContainer.MigrateDB(s.ctx)
	require.NoError(s.T(), err)

	s.repo = NewPostgresTransactionRepository(s.pgContainer.Pool)
}

func (s *PostgresTransactionRepositoryTestSuite) TearDownSuite() {
	if s.pgContainer != nil {
		err := s.pgContainer.Terminate(s.ctx)
		assert.NoError(s.T(), err)
	}
}

func (s *PostgresTransactionRepositoryTestSuite) SetupTest() {
	testutil.ResetAccountBalance(s.ctx, s.T(), s.pgContainer.Pool, 1, 1000)
	testutil.TruncateTransactions(s.ctx, s.T(), s.pgContainer.Pool)
}

func (s *PostgresTransactionRepositoryTestSuite) TestCreate() {
	testCases := []struct {
		name          string
		transaction   *transaction.Transaction
		expectedError error
	}{
		{
			name: "Successful transaction creation",
			transaction: &transaction.Transaction{
				TransactionID: "win-1",
				AccountID:     1,
				SourceType:    transaction.SourceTypeGame,
				State:         transaction.StateWin,
				Amount:        100,
			},
			expectedError: nil,
		},
		{
			name: "Duplicate transaction",
			transaction: &transaction.Transaction{
				TransactionID: "win-1",
				AccountID:     1,
				SourceType:    transaction.SourceTypeGame,
				State:         transaction.StateWin,
				Amount:        100,
			},
			expectedError: internal.ErrDuplicateTransaction,
		},
		{
			name: "Successful win transaction",
			transaction: &transaction.Transaction{
				TransactionID: "win-2",
				AccountID:     1,
				SourceType:    "game",
				State:         "win",
				Amount:        100,
			},
			expectedError: nil,
		},
		{
			name: "Successful loss transaction",
			transaction: &transaction.Transaction{
				TransactionID: "loss-1",
				AccountID:     1,
				SourceType:    "game",
				State:         "lost",
				Amount:        50,
			},
			expectedError: nil,
		},
		{
			name: "Duplicate transaction",
			transaction: &transaction.Transaction{
				TransactionID: "win-1",
				AccountID:     1,
				SourceType:    "game",
				State:         "win",
				Amount:        100,
			},
			expectedError: internal.ErrDuplicateTransaction,
		},
		{
			name: "Zero amount transaction",
			transaction: &transaction.Transaction{
				TransactionID: "zero-1",
				AccountID:     1,
				SourceType:    "game",
				State:         "win",
				Amount:        0,
			},
			expectedError: nil,
		},
		{
			name: "Large amount transaction within database limits",
			transaction: &transaction.Transaction{
				TransactionID: "large-1",
				AccountID:     1,
				SourceType:    "game",
				State:         "win",
				Amount:        999999.99,
			},
			expectedError: nil,
		},
		{
			name: "Transaction amount exceeding database limits",
			transaction: &transaction.Transaction{
				TransactionID: "too-large-1",
				AccountID:     1,
				SourceType:    "game",
				State:         "win",
				Amount:        1e10,
			},
			expectedError: internal.ErrNumericOverflow,
		},
	}

	s.SetupTest()
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := s.repo.Create(s.ctx, tc.transaction)
			if tc.expectedError != nil {
				s.ErrorIs(err, tc.expectedError)
			} else {
				s.NoError(err)
				// Verify the transaction was created
				storedTx, err := s.repo.GetByID(s.ctx, tc.transaction.TransactionID)
				s.NoError(err)
				s.Equal(tc.transaction.TransactionID, storedTx.TransactionID)
				s.Equal(tc.transaction.AccountID, storedTx.AccountID)
				s.Equal(tc.transaction.SourceType, storedTx.SourceType)
				s.Equal(tc.transaction.State, storedTx.State)
				s.Equal(tc.transaction.Amount, storedTx.Amount)
			}
		})
	}
}

func (s *PostgresTransactionRepositoryTestSuite) TestGetLatestOddRecords() {
	// Create some test transactions
	transactions := testutil.GenerateTransactions(20)
	for _, tx := range transactions {
		err := s.repo.Create(s.ctx, &tx)
		s.Require().NoError(err)
	}

	// Test fetching latest odd records
	limit := 5
	oddRecords, err := s.repo.GetLatestOddRecords(s.ctx, limit)
	s.Require().NoError(err)
	s.Len(oddRecords, limit)

	// Verify that only odd-numbered records are returned
	for _, tx := range oddRecords {
		s.True(tx.ID%2 == 1, "Expected only odd-numbered records")
	}
}

func (s *PostgresTransactionRepositoryTestSuite) TestMarkAsCanceled() {
	// Create some test transactions
	transactions := testutil.GenerateTransactions(10)
	for _, tx := range transactions {
		err := s.repo.Create(s.ctx, &tx)
		s.Require().NoError(err)
	}

	// Mark some transactions as canceled
	idsToCancel := []string{transactions[0].TransactionID, transactions[2].TransactionID, transactions[4].TransactionID}
	err := s.repo.MarkAsCanceled(s.ctx, idsToCancel)
	s.Require().NoError(err)

	// Verify that the transactions are marked as canceled
	for _, id := range idsToCancel {
		tx, err := s.repo.GetByID(s.ctx, id)
		s.Require().NoError(err)
		s.True(tx.IsCanceled, "Expected transaction to be marked as canceled")
	}
}

func (s *PostgresTransactionRepositoryTestSuite) TestTransactionPropertyBased() {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.MaxSize = 50

	properties := gopter.NewProperties(parameters)

	properties.Property("Transaction integrity and consistency", prop.ForAll(
		func(transactions []*transaction.Transaction) bool {
			s.SetupTest()
			processedTransactions := make(map[string]bool)

			for i, tx := range transactions {
				err := s.repo.Create(s.ctx, tx)

				if processedTransactions[tx.TransactionID] {
					if !errors.Is(err, internal.ErrDuplicateTransaction) {
						fmt.Printf("Expected duplicate transaction error for transaction %d: %v\n", i, tx)
						return false
					}
					continue
				}

				if err != nil {
					fmt.Printf("Unexpected error creating transaction %d: %v\n", i, err)
					return false
				}

				processedTransactions[tx.TransactionID] = true

				// Verify the transaction was stored correctly
				storedTx, err := s.repo.GetByID(s.ctx, tx.TransactionID)
				if err != nil {
					fmt.Printf("Error retrieving transaction %d: %v\n", i, err)
					return false
				}
				testutil.CompareTransactions(tx, storedTx)
			}

			return true
		},
		gen.SliceOf(genTransaction()),
	))

	properties.TestingRun(s.T())
}

func genTransaction() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),
		gen.OneConstOf(transaction.SourceTypeGame, transaction.SourceTypeServer, transaction.SourceTypePayment),
		gen.OneConstOf(transaction.StateWin, transaction.StateLost),
		gen.Float64Range(0.01, 1000.00),
	).Map(func(v []interface{}) *transaction.Transaction {
		tx := &transaction.Transaction{
			TransactionID: v[0].(string),
			AccountID:     1,
			SourceType:    v[1].(transaction.SourceType),
			State:         v[2].(transaction.State),
			Amount:        v[3].(float64),
		}
		tx.ProcessedAt = time.Now() // Ensure ProcessedAt is set
		return tx
	})
}
