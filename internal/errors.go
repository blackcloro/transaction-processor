package internal

import "errors"

var (
	ErrInsufficientFunds       = errors.New("insufficient funds")
	ErrDuplicateTransaction    = errors.New("duplicate transaction")
	ErrInvalidTransactionState = errors.New("invalid transaction state")
	ErrNumericOverflow         = errors.New("numeric field overflow")
	ErrTransactionNotFound     = errors.New("transaction not found")
)
