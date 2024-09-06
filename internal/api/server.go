package api

import (
	"context"
	"fmt"
	"time"

	"github.com/blackcloro/transaction-processor/internal/api/handlers"
	"github.com/blackcloro/transaction-processor/internal/config"
	"github.com/blackcloro/transaction-processor/pkg/logger"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	fiberLogger "github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

type Server struct {
	app                *fiber.App
	config             *config.Config
	transactionHandler *handlers.TransactionHandler
}

func NewServer(cfg *config.Config, th *handlers.TransactionHandler) *Server {
	app := fiber.New()
	app.Use(fiberLogger.New())
	app.Use(recover.New())
	app.Use(limiter.New(limiter.Config{
		Max:               20,
		Expiration:        30 * time.Second,
		LimiterMiddleware: limiter.SlidingWindow{},
	}))

	server := &Server{
		app:                app,
		config:             cfg,
		transactionHandler: th,
	}

	SetupRoutes(app, th)

	return server
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Port)
	logger.Info("Starting server at address", addr)
	err := s.app.Listen(addr)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}
