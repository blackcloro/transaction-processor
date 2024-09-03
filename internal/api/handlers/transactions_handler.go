package handlers

import (
	"errors"

	"github.com/blackcloro/transaction-processor/internal/data"
	"github.com/blackcloro/transaction-processor/internal/repository"
	"github.com/blackcloro/transaction-processor/pkg/logger"
	"github.com/gofiber/fiber/v3"
)

type TransactionHandler struct {
	repo *repository.TransactionRepository
}

func NewTransactionHandler(repo *repository.TransactionRepository) *TransactionHandler {
	return &TransactionHandler{
		repo: repo,
	}
}

func (h *TransactionHandler) CreateTransaction(c fiber.Ctx) error {
	var transaction data.Transaction
	if err := c.Bind().JSON(&transaction); err != nil {
		logger.Error("Invalid request body", err,
			"path", c.Path(),
			"method", c.Method(),
			"ip", c.IP())

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	sourceType := c.Get("Source-Type")

	transaction.SourceType = sourceType

	if err := transaction.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	logger.Info("Processing transaction", "transactionID", transaction.TransactionID)

	balance, err := h.repo.ProcessTransaction(c.Context(), &transaction)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateTransaction) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Duplicate transaction",
			})
		}
		logger.Warn("Failed to process transaction", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process transaction",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":     "Transaction processed successfully",
		"balance":     balance,
		"transaction": transaction,
	})
}
