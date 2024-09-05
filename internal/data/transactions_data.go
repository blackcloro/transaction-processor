package data

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Transaction struct {
	ID            int64      `json:"id"`
	AccountID     int64      `json:"account_id"`
	TransactionID string     `json:"transactionId" validate:"required"`
	SourceType    string     `json:"source_type" validate:"required,oneof=game server payment"`
	State         string     `json:"state" validate:"required,oneof=win lost"`
	Amount        float64    `json:"amount,string" validate:"required,gt=0"`
	IsCanceled    bool       `json:"is_canceled"`
	ProcessedAt   *time.Time `json:"processed_at"`
}

var validate = validator.New()

func (t *Transaction) Validate() error {
	return validate.Struct(t)
}

type TransactionModel struct {
	DB *pgxpool.Pool
}
