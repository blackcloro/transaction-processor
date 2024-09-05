package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/blackcloro/transaction-processor/pkg/logger"

	"github.com/blackcloro/transaction-processor/internal/data"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	ErrNoRows               = errors.New("no rows in result set")
	ErrDuplicateTransaction = errors.New("duplicate transaction")
	ErrInsufficientFunds    = errors.New("insufficient funds")
)

type TransactionRepository struct {
	db *pgxpool.Pool
}

func NewTransactionRepository(db *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) ProcessTransaction(ctx context.Context, t *data.Transaction) (float64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}

	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			logger.Warn("Rollback failed", "error", err)
		}
	}(tx, ctx)

	var currentBalance float64
	err = tx.QueryRow(ctx, "SELECT balance FROM account WHERE id = 1").Scan(&currentBalance)
	if err != nil {
		return 0, err
	}
	existingTransaction, _ := r.getTransactionByID(ctx, tx, t.TransactionID)
	if existingTransaction != nil {
		return currentBalance, ErrDuplicateTransaction
	}
	// Calculate new balance
	var newBalance float64
	switch t.State {
	case "win":
	case "lost":
		if currentBalance < t.Amount {
			return currentBalance, ErrInsufficientFunds
		}
	default:
		return 0, fmt.Errorf("invalid transaction state: %s", t.State)
	}
	// Create new transaction
	err = r.createTransaction(ctx, tx, t)
	if err != nil {
		return 0, err
	}
	// Check current balance

	newBalance, err = r.updateAccountBalance(ctx, tx, t.State, t.Amount)
	if err != nil {
		if errors.Is(err, ErrInsufficientFunds) {
			// If there are insufficient funds, cancel the transaction
			cancelErr := r.markTransactionCanceled(ctx, tx, t.TransactionID)
			if cancelErr != nil {
				logger.Error("Failed to cancel transaction", cancelErr)
				return 0, cancelErr
			}
			if err := tx.Commit(ctx); err != nil {
				return 0, err
			}
			return newBalance, ErrInsufficientFunds
		}
		return 0, err
	}

	err = r.markTransactionProcessed(ctx, tx, t.TransactionID)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	return newBalance, nil
}

func (r *TransactionRepository) getTransactionByID(ctx context.Context, tx pgx.Tx, id string) (*data.Transaction, error) {
	sql := `
		SELECT id
		FROM transactions
		WHERE transaction_id = $1
	`
	var t data.Transaction
	err := tx.QueryRow(ctx, sql, id).Scan(&t.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoRows
		}
		return nil, err
	}
	return &t, nil
}

func (r *TransactionRepository) createTransaction(ctx context.Context, tx pgx.Tx, t *data.Transaction) error {
	sql := `
		INSERT INTO transactions (transaction_id, source_type, state, amount, is_processed, is_canceled, created_at)
		VALUES ($1, $2, $3, $4, false, false, $5)
		RETURNING id, created_at
	`
	now := time.Now()
	return tx.QueryRow(ctx, sql, t.TransactionID, t.SourceType, t.State, t.Amount, now).Scan(&t.ID, &t.CreatedAt)
}

func (r *TransactionRepository) markTransactionProcessed(ctx context.Context, tx pgx.Tx, transactionID string) error {
	sql := `
		UPDATE transactions
		SET is_processed = true, processed_at = $1
		WHERE transaction_id = $2
	`
	_, err := tx.Exec(ctx, sql, time.Now(), transactionID)
	return err
}

func (r *TransactionRepository) markTransactionCanceled(ctx context.Context, tx pgx.Tx, transactionID string) error {
	sql := `
		UPDATE transactions
		SET is_canceled = true, canceled_at = $1
		WHERE transaction_id = $2
	`
	_, err := tx.Exec(ctx, sql, time.Now(), transactionID)
	return err
}

func (r *TransactionRepository) updateAccountBalance(ctx context.Context, tx pgx.Tx, state string, amount float64) (float64, error) {
	var currentBalance float64
	err := tx.QueryRow(ctx, "SELECT balance FROM account WHERE id = 1").Scan(&currentBalance)
	if err != nil {
		return 0, err
	}

	var newBalance float64
	switch {
	case state == "win":
		newBalance = currentBalance + amount
	case state == "lost":
		if currentBalance < amount {
			return currentBalance, ErrInsufficientFunds
		}
		newBalance = currentBalance - amount
	default:
		return 0, errors.New("invalid transaction state")
	}

	_, err = tx.Exec(ctx, "UPDATE account SET balance = $1 WHERE id = 1", newBalance)
	if err != nil {
		return 0, err
	}

	return newBalance, nil
}
