package repository

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/blackcloro/transaction-processor/internal/data"
	"github.com/blackcloro/transaction-processor/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TransactionRepositoryTestSuite struct {
	suite.Suite
	ctx         context.Context
	pgContainer *testutil.PostgresContainer
	repo        *TransactionRepository
}

func TestTransactionRepositorySuite(t *testing.T) {
	suite.Run(t, new(TransactionRepositoryTestSuite))
}

func (s *TransactionRepositoryTestSuite) SetupSuite() {
	s.ctx = context.Background()
	var err error
	s.pgContainer, err = testutil.NewPostgresContainer(s.ctx, testutil.PostgresConfig{
		User:     "test_user",
		Password: "test_password",
		DBName:   "test_db",
	})
	require.NoError(s.T(), err)

	err = s.pgContainer.MigrateDB(s.ctx)
	require.NoError(s.T(), err)

	s.repo = NewTransactionRepository(s.pgContainer.Pool)
}

func (s *TransactionRepositoryTestSuite) TearDownSuite() {
	if s.pgContainer != nil {
		err := s.pgContainer.Terminate(s.ctx)
		assert.NoError(s.T(), err)
	}
}

func (s *TransactionRepositoryTestSuite) SetupTest() {
	_, err := s.pgContainer.Pool.Exec(s.ctx, "UPDATE account SET balance = 1000 WHERE id = 1")
	require.NoError(s.T(), err)
	_, err = s.pgContainer.Pool.Exec(s.ctx, "TRUNCATE TABLE transactions")
	require.NoError(s.T(), err)
}

func (s *TransactionRepositoryTestSuite) TestProcessTransaction() {
	testCases := []struct {
		name            string
		transaction     data.Transaction
		expectedError   error
		expectedBalance float64
		expectedOutcome string
	}{
		{
			name: "Successful win transaction",
			transaction: data.Transaction{
				TransactionID: "win-1",
				SourceType:    "game",
				State:         "win",
				Amount:        100,
			},
			expectedError:   nil,
			expectedBalance: 1100,
			expectedOutcome: "new",
		},
		{
			name: "Successful loss transaction",
			transaction: data.Transaction{
				TransactionID: "loss-1",
				SourceType:    "game",
				State:         "lost",
				Amount:        50,
			},
			expectedError:   nil,
			expectedBalance: 1050,
			expectedOutcome: "new",
		},
		{
			name: "Insufficient funds",
			transaction: data.Transaction{
				TransactionID: "loss-2",
				SourceType:    "game",
				State:         "lost",
				Amount:        2000,
			},
			expectedError:   ErrInsufficientFunds,
			expectedBalance: 1050,
			expectedOutcome: "insufficient_funds",
		},
		{
			name: "Duplicate transaction",
			transaction: data.Transaction{
				TransactionID: "win-1",
				SourceType:    "game",
				State:         "win",
				Amount:        100,
			},
			expectedError:   ErrDuplicateTransaction,
			expectedBalance: 1050,
			expectedOutcome: "duplicate",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			balance, err := s.repo.ProcessTransaction(s.ctx, &tc.transaction)

			if tc.expectedError != nil {
				s.ErrorIs(err, tc.expectedError)
			} else {
				s.NoError(err)
			}

			s.Equal(tc.expectedBalance, balance)

			s.verifyTransactionRecord(&tc.transaction, tc.expectedOutcome)
		})
	}
}

func (s *TransactionRepositoryTestSuite) TestTransactionPropertyBased() {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100 // Increased for more thorough testing
	parameters.MaxSize = 50

	properties := gopter.NewProperties(parameters)

	properties.Property("Transaction integrity and balance consistency", prop.ForAll(
		func(transactions []data.Transaction) bool {
			s.SetupTest() // Reset state for each property test run
			initialBalance := 1000.0
			expectedBalance := initialBalance
			processedTransactions := make(map[string]bool)

			for _, tx := range transactions {
				balance, err := s.repo.ProcessTransaction(s.ctx, &tx)

				if processedTransactions[tx.TransactionID] {
					if !errors.Is(err, ErrDuplicateTransaction) {
						s.T().Errorf("Expected ErrDuplicateTransaction for duplicate transaction, got: %v", err)
						return false
					}
					continue
				}

				if err != nil {
					if errors.Is(err, ErrInsufficientFunds) {
						continue
					}
					s.T().Errorf("Unexpected error: %v", err)
					return false
				}

				processedTransactions[tx.TransactionID] = true

				if tx.State == "win" {
					expectedBalance += tx.Amount
				} else if tx.State == "lost" && expectedBalance >= tx.Amount {
					expectedBalance -= tx.Amount
				}

				if math.Abs(balance-expectedBalance) > 0.01 {
					s.T().Errorf("Balance mismatch. Expected: %.2f, Got: %.2f", expectedBalance, balance)
					return false
				}
			}

			return s.verifyFinalState(expectedBalance)
		},
		gen.SliceOfN(20, genTransaction()),
	))

	properties.TestingRun(s.T())
}

func (s *TransactionRepositoryTestSuite) verifyTransactionRecord(tx *data.Transaction, expectedOutcome string) {
	var transactions []data.Transaction
	rows, err := s.pgContainer.Pool.Query(s.ctx, `
		SELECT id, transaction_id, source_type, state, amount, is_processed
		FROM transactions WHERE transaction_id = $1
		ORDER BY id
	`, tx.TransactionID)
	s.Require().NoError(err)
	defer rows.Close()

	for rows.Next() {
		var t data.Transaction
		err := rows.Scan(
			&t.ID,
			&t.TransactionID,
			&t.SourceType,
			&t.State,
			&t.Amount,
			&t.IsProcessed,
		)
		s.Require().NoError(err)
		transactions = append(transactions, t)
	}
	s.Require().NoError(rows.Err())

	switch expectedOutcome {
	case "new":
		s.Require().Len(transactions, 1, "Expected exactly one transaction for a new transaction")
		storedTx := transactions[0]
		s.Equal(tx.TransactionID, storedTx.TransactionID)
		s.Equal(tx.SourceType, storedTx.SourceType)
		s.Equal(tx.State, storedTx.State)
		s.Equal(tx.Amount, storedTx.Amount)
		s.True(storedTx.IsProcessed)
	case "duplicate":
		s.Require().Len(transactions, 1, "Expected exactly one transaction for a duplicate transaction")
		// For a duplicate, we don't need to check the details again,
		// as they should have been verified when it was first inserted
	case "insufficient_funds":
		s.Require().Len(transactions, 0, "Expected no transactions for insufficient funds")
	default:
		s.Fail("Unknown expected outcome")
	}
}

func (s *TransactionRepositoryTestSuite) verifyFinalState(expectedBalance float64) bool {
	var actualBalance float64
	err := s.pgContainer.Pool.QueryRow(s.ctx, "SELECT balance FROM account WHERE id = 1").Scan(&actualBalance)
	if err != nil {
		s.T().Errorf("Failed to fetch final balance: %v", err)
		return false
	}

	balanceCorrect := math.Abs(actualBalance-expectedBalance) < 0.01
	balanceNonNegative := actualBalance >= 0

	if !balanceCorrect || !balanceNonNegative {
		s.T().Errorf("Test failed: Balance correct: %v, Balance non-negative: %v", balanceCorrect, balanceNonNegative)
		s.T().Errorf("Expected balance: %.2f, Actual balance: %.2f", expectedBalance, actualBalance)
	}

	// Additional invariant: sum of all processed transactions should equal the change in balance
	var totalChange float64
	err = s.pgContainer.Pool.QueryRow(s.ctx, `
		SELECT COALESCE(SUM(CASE WHEN state = 'win' THEN amount ELSE -amount END), 0)
		FROM transactions
		WHERE is_processed = true
	`).Scan(&totalChange)
	if err != nil {
		s.T().Errorf("Failed to calculate total transaction change: %v", err)
		return false
	}

	if math.Abs((actualBalance-1000)-totalChange) > 0.01 {
		s.T().Errorf("Balance change doesn't match processed transactions. Balance change: %.2f, Total transactions: %.2f", actualBalance-1000, totalChange)
		return false
	}

	return balanceCorrect && balanceNonNegative
}

func genTransaction() gopter.Gen {
	return gen.Struct(reflect.TypeOf(data.Transaction{}), map[string]gopter.Gen{
		"TransactionID": gen.Identifier(),
		"SourceType":    gen.OneConstOf("game", "server", "payment"),
		"State":         gen.OneConstOf("win", "lost"),
		"Amount":        gen.Float64Range(0.01, 1000.00),
	})
}
