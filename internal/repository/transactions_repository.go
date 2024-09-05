package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgconn"

	"github.com/blackcloro/transaction-processor/pkg/logger"

	"github.com/blackcloro/transaction-processor/internal/data"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	ErrNoRows               = errors.New("no rows in result set")
	ErrDuplicateTransaction = errors.New("duplicate transaction")
	ErrInsufficientFunds    = errors.New("insufficient funds")
	ErrNumericOverflow      = errors.New("numeric field overflow")
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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22003" {
			return currentBalance, ErrNumericOverflow
		}
		return currentBalance, err
	}
	// Update balance
	newBalance, err = r.updateAccountBalance(ctx, tx, t.State, t.Amount)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22003" {
			return currentBalance, ErrNumericOverflow
		}
		return currentBalance, err
	}

	if err := tx.Commit(ctx); err != nil {
		return currentBalance, err
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
		INSERT INTO transactions (transaction_id, source_type, state, amount, processed_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, processed_at
	`
	now := time.Now()
	return tx.QueryRow(ctx, sql, t.TransactionID, t.SourceType, t.State, t.Amount, now).Scan(&t.ID, &t.ProcessedAt)
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

func (r *TransactionRepository) PostProcess(ctx context.Context) ([]data.Transaction, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Select and update the 10 latest odd records that are not canceled
	rows, err := tx.Query(ctx, `
		WITH numbered_records AS (
			SELECT id, transaction_id, account_id, amount, state, source_type, processed_at,
				   ROW_NUMBER() OVER (ORDER BY processed_at DESC) as row_num
			FROM transactions
			WHERE is_canceled = false
		),
		updated_records AS (
			UPDATE transactions
			SET is_canceled = true
			WHERE id IN (
				SELECT id
				FROM numbered_records
				WHERE id % 2 = 1
				ORDER BY id
				LIMIT 10
			)
			RETURNING id, transaction_id, amount, state, source_type, processed_at, is_canceled
		)
		SELECT * FROM updated_records
		ORDER BY processed_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query and update transactions: %w", err)
	}
	defer rows.Close()

	var canceledTransactions []data.Transaction
	for rows.Next() {
		var t data.Transaction
		if err := rows.Scan(&t.ID, &t.TransactionID, &t.Amount, &t.State, &t.SourceType, &t.ProcessedAt, &t.IsCanceled); err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		canceledTransactions = append([]data.Transaction{t}, canceledTransactions...) // Prepend to reverse order
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	// Update account balance
	for _, t := range canceledTransactions {
		var balanceChange float64
		if t.State == "win" {
			balanceChange = -t.Amount
		} else {
			balanceChange = t.Amount
		}

		_, err := tx.Exec(ctx, `
			UPDATE account
			SET balance = balance + $1, version = version + 1, updated_at = NOW()
			WHERE id = 1
		`, balanceChange)
		if err != nil {
			return nil, fmt.Errorf("failed to update account balance: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return canceledTransactions, nil
}
