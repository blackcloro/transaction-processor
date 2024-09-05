package database

import (
	"context"

	"github.com/blackcloro/transaction-processor/internal/domain/account"
	"github.com/jackc/pgx/v4/pgxpool"
)

type PostgresAccountRepository struct {
	db *pgxpool.Pool
}

func NewPostgresAccountRepository(db *pgxpool.Pool) *PostgresAccountRepository {
	return &PostgresAccountRepository{db: db}
}

func (r *PostgresAccountRepository) GetByID(ctx context.Context, id int64) (*account.Account, error) {
	var a account.Account
	err := r.db.QueryRow(ctx, "SELECT id, balance, version, updated_at FROM account WHERE id = $1", id).
		Scan(&a.ID, &a.Balance, &a.Version, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *PostgresAccountRepository) Update(ctx context.Context, a *account.Account) error {
	_, err := r.db.Exec(ctx, "UPDATE account SET balance = $1, version = $2, updated_at = $3 WHERE id = $4",
		a.Balance, a.Version, a.UpdatedAt, a.ID)
	return err
}

//func (r *PostgresAccountRepository) UpdateBalance(ctx context.Context, id int, amount float64) error {
//	_, err := r.db.Exec(ctx, "UPDATE account SET balance = $1, updated_at = $3 WHERE id = $4",
//		amount, time.Now(), id)
//	return err
//}

//
//func (r *PostgresAccountRepository) WithTransaction(ctx context.Context, fn func(account.Repository) (float64, error)) (float64, error) {
//	tx, err := r.db.Begin(ctx)
//	if err != nil {
//		return 0, err
//	}
//	defer tx.Rollback(ctx)
//
//	txRepo := &PostgresAccountRepository{db: &pgxpool.Pool{Conn: tx}}
//	result, err := fn(txRepo)
//	if err != nil {
//		return 0, err
//	}
//
//	if err := tx.Commit(ctx); err != nil {
//		return 0, err
//	}
//
//	return result, nil
//}
