package worker

import (
	"context"
	"time"

	"github.com/blackcloro/transaction-processor/internal/domain/transaction"
	"github.com/blackcloro/transaction-processor/pkg/logger"
)

type Worker struct {
	transactionService *transaction.Service
	interval           time.Duration
	stopChan           chan struct{}
	processingDone     chan struct{}
}

func NewWorker(ts *transaction.Service, interval time.Duration) *Worker {
	return &Worker{
		transactionService: ts,
		interval:           interval,
		stopChan:           make(chan struct{}),
		processingDone:     make(chan struct{}),
	}
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(w.processingDone)
			return
		case <-w.stopChan:
			close(w.processingDone)
			return
		case <-ticker.C:
			w.runPostProcessing(ctx)
		}
	}
}

func (w *Worker) runPostProcessing(ctx context.Context) {
	err := w.transactionService.PostProcess(ctx)
	if err != nil {
		logger.Error("Failed to run post-processing", err)
		return
	}

	logger.Info("Post-processing completed")
}

func (w *Worker) Stop() {
	close(w.stopChan)
	<-w.processingDone
}
