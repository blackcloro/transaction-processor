package api

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/blackcloro/transaction-processor/internal/api/handlers"
	"github.com/blackcloro/transaction-processor/internal/config"
	"github.com/blackcloro/transaction-processor/internal/repository"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"github.com/jackc/pgx/v4/pgxpool"
)

type Server struct {
	app    *fiber.App
	config *config.Config
}

func NewServer(cfg *config.Config, db *pgxpool.Pool) *Server {
	app := fiber.New()
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(limiter.New(limiter.Config{
		Max:               20,
		Expiration:        30 * time.Second,
		LimiterMiddleware: limiter.SlidingWindow{},
	}))
	transactionRepo := repository.NewTransactionRepository(db)
	transactionHandler := handlers.NewTransactionHandler(transactionRepo)

	SetupRoutes(app, transactionHandler)

	return &Server{
		app:    app,
		config: cfg,
	}
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Port)
	slog.Info("Starting server", "env", s.config.Env, "address", addr)
	return s.app.Listen(addr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	shutdownComplete := make(chan struct{})

	var shutdownErr error
	go func() {
		defer close(shutdownComplete)
		shutdownErr = s.app.ShutdownWithContext(ctx)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-shutdownComplete:
		if shutdownErr != nil {
			slog.Error("Error during shutdown", "error", shutdownErr)
		}
		return shutdownErr
	}
}
