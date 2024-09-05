package transaction

import (
	"context"
	"time"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateTransaction(ctx context.Context, tx *Transaction) error {
	if err := tx.Validate(); err != nil {
		return err
	}
	tx.ProcessedAt = time.Now()
	return s.repo.Create(ctx, tx)
}

func (s *Service) PostProcess(ctx context.Context) error {
	transactions, err := s.repo.GetLatestOddRecords(ctx, 10)
	if err != nil {
		return err
	}

	ids := make([]string, len(transactions))
	for i, tx := range transactions {
		ids[i] = tx.TransactionID
	}

	return s.repo.MarkAsCanceled(ctx, ids)
}
