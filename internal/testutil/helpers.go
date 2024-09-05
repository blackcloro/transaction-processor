package testutil

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/blackcloro/transaction-processor/internal/domain/transaction"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/require"
)

// ResetAccountBalance sets the balance of the account with the given ID to the specified amount.
func ResetAccountBalance(ctx context.Context, t require.TestingT, pool *pgxpool.Pool, accountID int, balance float64) {
	_, err := pool.Exec(ctx, "UPDATE account SET balance = $1 WHERE id = $2", balance, accountID)
	require.NoError(t, err)
}

// TruncateTransactions removes all records from the transactions table.
func TruncateTransactions(ctx context.Context, t require.TestingT, pool *pgxpool.Pool) {
	_, err := pool.Exec(ctx, "TRUNCATE TABLE transactions")
	require.NoError(t, err)
}

// GetAccountBalance retrieves the current balance of the account with the given ID.
func GetAccountBalance(ctx context.Context, t require.TestingT, pool *pgxpool.Pool, accountID int) float64 {
	var balance float64
	err := pool.QueryRow(ctx, "SELECT balance FROM account WHERE id = $1", accountID).Scan(&balance)
	require.NoError(t, err)
	return balance
}

// VerifyTransactionRecord checks if a transaction record exists and matches the expected outcome.
func VerifyTransactionRecord(ctx context.Context, t require.TestingT, pool *pgxpool.Pool, tx *transaction.Transaction, expectedOutcome string) {
	var count int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM transactions WHERE transaction_id = $1", tx.TransactionID).Scan(&count)
	require.NoError(t, err)

	switch expectedOutcome {
	case "new":
		require.Equal(t, 1, count, "Expected exactly one transaction for a new transaction")
		VerifyTransactionDetails(ctx, t, pool, tx)
	case "duplicate":
		require.Equal(t, 1, count, "Expected exactly one transaction for a duplicate transaction")
	case "insufficient_funds", "error":
		require.Equal(t, 0, count, "Expected no transactions for insufficient funds or error case")
	default:
		require.Fail(t, "Unknown expected outcome")
	}
}

// VerifyTransactionDetails checks if the stored transaction details match the expected values.
func VerifyTransactionDetails(ctx context.Context, t require.TestingT, pool *pgxpool.Pool, expectedTx *transaction.Transaction) {
	var storedTx transaction.Transaction
	err := pool.QueryRow(ctx, `
		SELECT transaction_id, source_type, state, amount
		FROM transactions WHERE transaction_id = $1
	`, expectedTx.TransactionID).Scan(
		&storedTx.TransactionID,
		&storedTx.SourceType,
		&storedTx.State,
		&storedTx.Amount,
	)
	require.NoError(t, err)
	require.Equal(t, expectedTx.TransactionID, storedTx.TransactionID)
	require.Equal(t, expectedTx.SourceType, storedTx.SourceType)
	require.Equal(t, expectedTx.State, storedTx.State)
	require.Equal(t, expectedTx.Amount, storedTx.Amount)
}

// VerifyTransactionStatuses checks if the cancellation status of transactions matches the expected pattern.
func VerifyTransactionStatuses(ctx context.Context, t require.TestingT, pool *pgxpool.Pool, txs []transaction.Transaction) {
	for i, tx := range txs {
		var isCanceled bool
		err := pool.QueryRow(ctx, "SELECT is_canceled FROM transactions WHERE transaction_id = $1", tx.TransactionID).Scan(&isCanceled)
		require.NoError(t, err)
		require.Equal(t, i%2 == 0, isCanceled, "Transaction status does not match expected")
	}
}

// VerifyFinalState checks if the final account balance and transaction sum are consistent.
func VerifyFinalState(ctx context.Context, t require.TestingT, pool *pgxpool.Pool, accountID int, expectedBalance float64) bool {
	actualBalance := GetAccountBalance(ctx, t, pool, accountID)
	balanceCorrect := math.Abs(actualBalance-expectedBalance) < 0.01
	balanceNonNegative := actualBalance >= 0

	if !balanceCorrect || !balanceNonNegative {
		return false
	}

	var totalChange float64
	err := pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN state = 'win' THEN amount ELSE -amount END), 0)
		FROM transactions
		WHERE is_canceled = false
	`).Scan(&totalChange)
	require.NoError(t, err)

	return math.Abs((actualBalance-1000)-totalChange) <= 0.01
}

func GenerateTransactions(count int) []transaction.Transaction {
	txs := make([]transaction.Transaction, count)
	for i := 0; i < count; i++ {
		txs[i] = transaction.Transaction{
			TransactionID: fmt.Sprintf("t%d", i+1),
			AccountID:     1,
			SourceType:    transaction.SourceTypeGame,
			State:         transaction.StateWin,
			Amount:        float64(i+1) * 10,
		}
	}
	return txs
}

func CompareTransactions(original, stored *transaction.Transaction) bool {
	// Compare relevant fields
	if original.TransactionID != stored.TransactionID ||
		original.SourceType != stored.SourceType ||
		original.State != stored.State ||
		math.Abs(original.Amount-stored.Amount) > 0.00001 { // Use a small epsilon for float comparison
		fmt.Printf("Mismatch in transaction details:\nOriginal: %+v\nStored: %+v\n", original, stored)
		return false
	}

	// Check if ProcessedAt is reasonably close (within 1 second)
	if !stored.ProcessedAt.IsZero() && !original.ProcessedAt.IsZero() {
		timeDiff := stored.ProcessedAt.Sub(original.ProcessedAt)
		if timeDiff < -time.Second || timeDiff > time.Second {
			fmt.Printf("ProcessedAt time mismatch:\nOriginal: %v\nStored: %v\n", original.ProcessedAt, stored.ProcessedAt)
			return false
		}
	} else if stored.ProcessedAt != original.ProcessedAt {
		fmt.Printf("ProcessedAt nil mismatch:\nOriginal: %v\nStored: %v\n", original.ProcessedAt, stored.ProcessedAt)
		return false
	}

	return true
}
