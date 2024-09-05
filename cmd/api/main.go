package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blackcloro/transaction-processor/internal/api"
	"github.com/blackcloro/transaction-processor/internal/api/handlers"
	"github.com/blackcloro/transaction-processor/internal/config"
	"github.com/blackcloro/transaction-processor/internal/domain/account"
	"github.com/blackcloro/transaction-processor/internal/domain/transaction"
	"github.com/blackcloro/transaction-processor/internal/infrastructure/database"
	"github.com/blackcloro/transaction-processor/internal/worker"
	"github.com/blackcloro/transaction-processor/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	logger.InitLogger()

	db, err := database.NewPostgresDB(cfg.DB.DSN)
	if err != nil {
		logger.Warn("Failed to connect to database", err)
	}
	defer db.Close()

	accountRepo := database.NewPostgresAccountRepository(db)
	transactionRepo := database.NewPostgresTransactionRepository(db)

	accountService := account.NewService(accountRepo)
	transactionService := transaction.NewService(transactionRepo)

	transactionHandler := handlers.NewTransactionHandler(accountService, transactionService, db)

	server := api.NewServer(cfg, transactionHandler)

	DBworker := worker.NewWorker(transactionService, cfg.Worker.Interval)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go DBworker.Start(ctx)

	go func() {
		if err := server.Start(); err != nil {
			logger.Fatal("Failed to start server", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", err)
	}

	logger.Info("Server exiting")
}
