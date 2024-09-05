package transaction

import (
	"context"
)

type Repository interface {
	Create(ctx context.Context, tx *Transaction) error
	GetByID(ctx context.Context, id string) (*Transaction, error)
	GetLatestOddRecords(ctx context.Context, limit int) ([]*Transaction, error)
	MarkAsCanceled(ctx context.Context, ids []string) error
}
