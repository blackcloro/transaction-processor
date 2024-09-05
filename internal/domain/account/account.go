package account

import (
	"time"

	"github.com/blackcloro/transaction-processor/internal"

	"github.com/blackcloro/transaction-processor/internal/domain/transaction"
)

type Account struct {
	ID        int64
	Balance   float64
	Version   int
	UpdatedAt time.Time
}

func (a *Account) ApplyTransaction(tx *transaction.Transaction) error {
	switch tx.State {
	case transaction.StateWin:
		a.Balance += tx.Amount
	case transaction.StateLost:
		if a.Balance < tx.Amount {
			return internal.ErrInsufficientFunds
		}
		a.Balance -= tx.Amount
	default:
		return internal.ErrInvalidTransactionState
	}
	a.Version++
	a.UpdatedAt = time.Now()
	return nil
}
