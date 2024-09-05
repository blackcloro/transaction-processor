package account

import (
	"context"

	"github.com/blackcloro/transaction-processor/internal/domain/transaction"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ProcessTransaction(ctx context.Context, accountID int64, tx *transaction.Transaction) (float64, error) {
	account, err := s.repo.GetByID(ctx, accountID)
	if err != nil {
		return 0, err
	}

	if err := account.ApplyTransaction(tx); err != nil {
		return account.Balance, err
	}

	if err := s.repo.Update(ctx, account); err != nil {
		return account.Balance, err
	}

	return account.Balance, nil
}

func (s *Service) GetBalance(ctx context.Context, accountID int64) (float64, error) {
	account, err := s.repo.GetByID(ctx, accountID)
	if err != nil {
		return 0, err
	}
	return account.Balance, nil
}

//
//func (s *Service) CheckAndProcessTransaction(ctx context.Context, accountID int64, tx *transaction.Transaction) (float64, error) {
//	return s.repo.WithTransaction(ctx, func(repo Repository) (float64, error) {
//		account, err := repo.GetByID(ctx, accountID)
//		if err != nil {
//			return 0, err
//		}
//
//		if tx.State == transaction.StateLost && account.Balance < tx.Amount {
//			return 0, internal.ErrInsufficientFunds
//		}
//
//		newBalance := account.Balance
//		if tx.State == transaction.StateWin {
//			newBalance += tx.Amount
//		} else {
//			newBalance -= tx.Amount
//		}
//
//		err = repo.UpdateBalance(ctx, accountID, newBalance)
//		if err != nil {
//			return 0, err
//		}
//
//		return newBalance, nil
//	})
//}
