package transaction

import (
	"time"

	"github.com/go-playground/validator/v10"
)

type State string

const (
	StateWin  State = "win"
	StateLost State = "lost"
)

type SourceType string

const (
	SourceTypeGame    SourceType = "game"
	SourceTypeServer  SourceType = "server"
	SourceTypePayment SourceType = "payment"
)

type Transaction struct {
	ID            int64      `json:"id"`
	AccountID     int64      `json:"account_id"`
	TransactionID string     `json:"transactionId" validate:"required"`
	SourceType    SourceType `json:"source_type" validate:"required,oneof=game server payment"`
	State         State      `json:"state" validate:"required,oneof=win lost"`
	Amount        float64    `json:"amount,string" validate:"required,gt=0"`
	IsCanceled    bool       `json:"is_canceled"`
	ProcessedAt   time.Time  `json:"processed_at"`
}

func (t *Transaction) Validate() error {
	validate := validator.New()
	return validate.Struct(t)
}
