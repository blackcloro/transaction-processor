package database

import (
	"context"
	"errors"

	"github.com/jackc/pgconn"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/blackcloro/transaction-processor/internal"
	"github.com/blackcloro/transaction-processor/internal/domain/transaction"
)

type PostgresTransactionRepository struct {
	db *pgxpool.Pool
}

func NewPostgresTransactionRepository(db *pgxpool.Pool) *PostgresTransactionRepository {
	return &PostgresTransactionRepository{db: db}
}

func (r *PostgresTransactionRepository) Create(ctx context.Context, tx *transaction.Transaction) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO transactions (transaction_id, account_id, source_type, state, amount, processed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, tx.TransactionID, tx.AccountID, tx.SourceType, tx.State, tx.Amount, tx.ProcessedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return internal.ErrDuplicateTransaction
		} else if errors.As(err, &pgErr) && pgErr.Code == "22003" {
			return internal.ErrNumericOverflow
		}
		return err
	}
	return nil
}

func (r *PostgresTransactionRepository) GetByID(ctx context.Context, id string) (*transaction.Transaction, error) {
	var tx transaction.Transaction
	err := r.db.QueryRow(ctx, `
		SELECT id, transaction_id, account_id, source_type, state, amount, is_canceled, processed_at
		FROM transactions
		WHERE transaction_id = $1
	`, id).Scan(
		&tx.ID, &tx.TransactionID, &tx.AccountID, &tx.SourceType, &tx.State, &tx.Amount, &tx.IsCanceled, &tx.ProcessedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, internal.ErrTransactionNotFound
		}
		return nil, err
	}
	return &tx, nil
}

func (r *PostgresTransactionRepository) GetLatestOddRecords(ctx context.Context, limit int) ([]*transaction.Transaction, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, transaction_id, account_id, source_type, state, amount, is_canceled, processed_at
		FROM transactions
		WHERE id % 2 = 1 AND is_canceled = false
		ORDER BY processed_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*transaction.Transaction
	for rows.Next() {
		tx := &transaction.Transaction{}
		err := rows.Scan(
			&tx.ID, &tx.TransactionID, &tx.AccountID, &tx.SourceType, &tx.State, &tx.Amount, &tx.IsCanceled, &tx.ProcessedAt,
		)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return transactions, nil
}

func (r *PostgresTransactionRepository) MarkAsCanceled(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE transactions
		SET is_canceled = true
		WHERE transaction_id = ANY($1)
	`, ids)
	return err
}
