package api

import (
	"github.com/blackcloro/transaction-processor/internal/api/handlers"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
)

func SetupRoutes(app *fiber.App, th *handlers.TransactionHandler) {
	api := app.Group("/api/v1")

	api.Post("/transactions", th.CreateTransaction)
	// Check if the server is up and running.
	api.Get(healthcheck.DefaultLivenessEndpoint, healthcheck.NewHealthChecker())
}
