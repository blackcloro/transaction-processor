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
