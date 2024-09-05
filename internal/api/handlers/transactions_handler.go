package handlers

import (
	"errors"

	"github.com/blackcloro/transaction-processor/pkg/logger"

	"github.com/blackcloro/transaction-processor/internal"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/blackcloro/transaction-processor/internal/domain/account"
	"github.com/blackcloro/transaction-processor/internal/domain/transaction"
)

//type TransactionHandler struct {
//	accountService     *account.Service
//	transactionService *transaction.Service
//}
//
//func NewTransactionHandler(as *account.Service, ts *transaction.Service) *TransactionHandler {
//	return &TransactionHandler{
//		accountService:     as,
//		transactionService: ts,
//	}
//}

type TransactionHandler struct {
	accountService     *account.Service
	transactionService *transaction.Service
	db                 *pgxpool.Pool
}

func NewTransactionHandler(as *account.Service, ts *transaction.Service, db *pgxpool.Pool) *TransactionHandler {
	return &TransactionHandler{
		accountService:     as,
		transactionService: ts,
		db:                 db,
	}
}

//	func (h *TransactionHandler) CreateTransaction(c fiber.Ctx) error {
//		var tx transaction.Transaction
//		if err := c.Bind().JSON(&tx); err != nil {
//			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
//		}
//
//		tx.SourceType = transaction.SourceType(c.Get("Source-Type"))
//
//		if err := h.transactionService.CreateTransaction(c.Context(), &tx); err != nil {
//			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
//		}
//
//		balance, err := h.accountService.ProcessTransaction(c.Context(), 1, &tx) // Assuming single account with ID 1
//		if err != nil {
//			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
//		}
//
//		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
//			"message":     "Transaction processed successfully",
//			"balance":     balance,
//			"transaction": tx,
//		})
//	}
//
//	func (h *TransactionHandler) CreateTransaction(c fiber.Ctx) error {
//		var tx transaction.Transaction
//		if err := c.Bind().JSON(&tx); err != nil {
//			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
//		}
//
//		tx.SourceType = transaction.SourceType(c.Get("Source-Type"))
//
//		// Start a database transaction
//		dbTx, err := h.db.Begin(c.Context())
//		if err != nil {
//			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start transaction"})
//		}
//		defer dbTx.Rollback(c.Context()) // Rollback in case of error
//
//		// Check current balance
//		currentBalance, err := h.accountService.GetBalance(c.Context(), 1) // Assuming single account with ID 1
//		if err != nil {
//			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get account balance"})
//		}
//
//		// Check for sufficient funds if it's a "lost" transaction
//		if tx.State == transaction.StateLost && currentBalance < tx.Amount {
//			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Insufficient funds"})
//		}
//
//		// Create the transaction
//		if err := h.transactionService.CreateTransaction(c.Context(), &tx); err != nil {
//			if errors.Is(err, internal.ErrDuplicateTransaction) {
//				return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Duplicate transaction"})
//			}
//			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create transaction"})
//		}
//
//		// Process the transaction (update balance)
//		newBalance, err := h.accountService.ProcessTransaction(c.Context(), 1, &tx) // Assuming single account with ID 1
//		if err != nil {
//			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to process transaction"})
//		}
//
//		// Commit the database transaction
//		if err := dbTx.Commit(c.Context()); err != nil {
//			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to commit transaction"})
//		}
//
//		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
//			"message":     "Transaction processed successfully",
//			"balance":     newBalance,
//			"transaction": tx,
//		})
//	}
//
//	func (h *TransactionHandler) CreateTransaction(c fiber.Ctx) error {
//		var tx transaction.Transaction
//		if err := c.Bind().JSON(&tx); err != nil {
//			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
//		}
//
//		tx.SourceType = transaction.SourceType(c.Get("Source-Type"))
//
//		// Check current balance and process transaction atomically
//		newBalance, err := h.accountService.CheckAndProcessTransaction(c.Context(), 1, &tx) // Assuming single account with ID 1
//		if err != nil {
//			switch {
//			case errors.Is(err, internal.ErrInsufficientFunds):
//				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Insufficient funds"})
//			case errors.Is(err, internal.ErrDuplicateTransaction):
//				return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Duplicate transaction"})
//			default:
//				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to process transaction"})
//			}
//		}
//
//		// Create the transaction record
//		if err := h.transactionService.CreateTransaction(c.Context(), &tx); err != nil {
//			// If creating the transaction record fails, we should roll back the balance change
//			// This should be handled in the service layer
//			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to record transaction"})
//		}
//
//		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
//			"message":     "Transaction processed successfully",
//			"balance":     newBalance,
//			"transaction": tx,
//		})
//	}
func (h *TransactionHandler) CreateTransaction(c fiber.Ctx) error {
	var tx transaction.Transaction
	if err := c.Bind().JSON(&tx); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	tx.SourceType = transaction.SourceType(c.Get("Source-Type"))
	tx.AccountID = 1

	// Start a database transaction
	dbTx, err := h.db.Begin(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start transaction"})
	}
	defer dbTx.Rollback(c.Context()) // Rollback in case of error

	// Check current balance
	currentBalance, err := h.accountService.GetBalance(c.Context(), 1) // Assuming single account with ID 1
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get account balance"})
	}

	// Check for sufficient funds if it's a "lost" transaction
	if tx.State == transaction.StateLost && currentBalance < tx.Amount {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Insufficient funds"})
	}

	// Create the transaction
	if err := h.transactionService.CreateTransaction(c.Context(), &tx); err != nil {
		logger.Warn(err.Error())
		if errors.Is(err, internal.ErrDuplicateTransaction) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Duplicate transaction"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create transaction"})
	}

	// Process the transaction (update balance)
	newBalance, err := h.accountService.ProcessTransaction(c.Context(), 1, &tx) // Assuming single account with ID 1
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to process transaction"})
	}

	// Commit the database transaction
	if err := dbTx.Commit(c.Context()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to commit transaction"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":     "Transaction processed successfully",
		"balance":     newBalance,
		"transaction": tx,
	})
}
