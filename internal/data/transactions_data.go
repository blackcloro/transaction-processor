package data

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Transaction struct {
	ID            int64      `json:"id"`
	TransactionID string     `json:"transactionId" validate:"required"`
	SourceType    string     `json:"source_type" validate:"required,oneof=game server payment"`
	State         string     `json:"state" validate:"required,oneof=win lost"`
	Amount        float64    `json:"amount,string" validate:"required,gt=0"`
	IsProcessed   bool       `json:"-"`
	IsCanceled    bool       `json:"-"`
	CreatedAt     *time.Time `json:"-"`
	ProcessedAt   *time.Time `json:"-"`
	CanceledAt    *time.Time `json:"-"`
}

var validate = validator.New()

func (t *Transaction) Validate() error {
	return validate.Struct(t)
}

type TransactionModel struct {
	DB *pgxpool.Pool
}
