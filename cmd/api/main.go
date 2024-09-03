package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blackcloro/transaction-processor/internal/api"
	"github.com/blackcloro/transaction-processor/internal/config"
	"github.com/blackcloro/transaction-processor/internal/database"
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
		logger.Fatal("Failed to connect to database", err)
	}

	server := api.NewServer(cfg, db)

	go func() {
		if err := server.Start(); err != nil {
			logger.Fatal("Failed to start server", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", err)
	}

	db.Close()

	logger.Info("Server exiting")
}
