package account

import (
	"context"
)

type Repository interface {
	GetByID(ctx context.Context, id int64) (*Account, error)
	Update(ctx context.Context, account *Account) error
	// UpdateBalance(ctx context.Context, a *Account) error
}
