package test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/blackcloro/transaction-processor/internal/repository"
	"github.com/golang-migrate/migrate/v4"
	"github.com/testcontainers/testcontainers-go"

	"github.com/blackcloro/transaction-processor/internal/data"

	"github.com/blackcloro/transaction-processor/pkg/logger"

	"github.com/blackcloro/transaction-processor/internal/api"
	"github.com/blackcloro/transaction-processor/internal/config"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	DBName = "test_db"
	DBUser = "test_user"
	DBPass = "test_password"
)

type Database struct {
	DBInstance *pgxpool.Pool
	DBAddress  string
	container  testcontainers.Container
}

type IntegrationTestSuite struct {
	suite.Suite
	serverURL string
	server    *api.Server
	testDB    *Database
	repo      *repository.TransactionRepository
}

func (s *IntegrationTestSuite) SetupSuite() {
	var err error
	s.testDB, err = SetupTestDatabase()
	if err != nil {
		s.T().Fatalf("Failed to set up test database: %v", err)
	}

	// Load configuration
	cfg := &config.Config{
		Port: 8080,
		Env:  "test",
		DB: struct{ DSN string }{
			DSN: fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", DBUser, DBPass, s.testDB.DBAddress, DBName),
		},
	}

	// Initialize logger for the test server
	logger.InitLogger()

	// Initialize and start the server
	s.server = api.NewServer(cfg, s.testDB.DBInstance)
	serverErrChan := make(chan error, 1)
	go func() {
		if err := s.server.Start(); err != nil {
			serverErrChan <- err
		}
	}()

	// Wait for the server to start or encounter an error
	select {
	case err := <-serverErrChan:
		s.T().Fatalf("Server failed to start: %v", err)
	case <-time.After(5 * time.Second):
		// Assume server has started successfully after 5 seconds
	}

	s.serverURL = fmt.Sprintf("http://localhost:%d", cfg.Port)
	s.repo = repository.NewTransactionRepository(s.testDB.DBInstance)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	if s.testDB != nil {
		s.testDB.TearDown()
	}
}

func SetupTestDatabase() (*Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	container, dbInstance, dbAddr, err := createContainer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to setup test container: %w", err)
	}

	err = migrateDB(dbAddr)
	if err != nil {
		err := container.Terminate(context.Background())
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to perform db migration: %w", err)
	}

	return &Database{
		container:  container,
		DBInstance: dbInstance,
		DBAddress:  dbAddr,
	}, nil
}

func (tdb *Database) TearDown() {
	if tdb.DBInstance != nil {
		tdb.DBInstance.Close()
	}

	if tdb.container != nil {
		err := tdb.container.Terminate(context.Background())
		if err != nil {
			log.Printf("Error while tearing down test database: %v\n", err)
		}
	}
}

func createContainer(ctx context.Context) (testcontainers.Container, *pgxpool.Pool, string, error) {
	env := map[string]string{
		"POSTGRES_PASSWORD": DBPass,
		"POSTGRES_USER":     DBUser,
		"POSTGRES_DB":       DBName,
	}
	port := "5432/tcp"

	req := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16-alpine",
			ExposedPorts: []string{port},
			Env:          env,
			WaitingFor:   wait.ForLog("database system is ready to accept connections"),
		},
		Started: true,
	}
	container, err := testcontainers.GenericContainer(ctx, req)
	if err != nil {
		return container, nil, "", fmt.Errorf("failed to start container: %w", err)
	}

	p, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return container, nil, "", fmt.Errorf("failed to get container external port: %w", err)
	}

	log.Println("postgres container ready and running at port: ", p.Port())

	time.Sleep(time.Second)

	dbAddr := fmt.Sprintf("localhost:%s", p.Port())
	db, err := pgxpool.Connect(ctx, fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", DBUser, DBPass, dbAddr, DBName))
	if err != nil {
		return container, db, dbAddr, fmt.Errorf("failed to establish database connection: %w", err)
	}

	return container, db, dbAddr, nil
}

func migrateDB(dbAddr string) error {
	_, path, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("failed to get path")
	}
	pathToMigrationFiles := filepath.Dir(path) + "/../migrations"

	databaseURL := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", DBUser, DBPass, dbAddr, DBName)
	m, err := migrate.New(fmt.Sprintf("file:%s", pathToMigrationFiles), databaseURL)
	if err != nil {
		return err
	}
	defer func(m *migrate.Migrate) {
		err, _ := m.Close()
		if err != nil {
			log.Printf("Error while closing migration: %+v\n", err)
		}
	}(m)

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	log.Println("migration done")

	return nil
}

func TestIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupTest() {
	_, err := s.testDB.DBInstance.Exec(context.Background(), "TRUNCATE TABLE transactions, account RESTART IDENTITY")
	s.Require().NoError(err)
	_, err = s.testDB.DBInstance.Exec(context.Background(), "INSERT INTO account (balance) VALUES (1000.00)")
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TestTransactions() {
	testCases := []struct {
		name           string
		transaction    data.Transaction
		sourceType     string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid transaction",
			transaction: data.Transaction{
				TransactionID: "test-123",
				State:         "win",
				Amount:        100.50,
			},
			sourceType:     "server",
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Invalid state",
			transaction: data.Transaction{
				TransactionID: "test-456",
				State:         "invalid",
				Amount:        50.25,
			},
			sourceType:     "game",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Key: 'Transaction.State' Error:Field validation for 'State' failed on the 'oneof' tag",
		},
		{
			name: "Missing transaction ID",
			transaction: data.Transaction{
				State:  "lost",
				Amount: 75.00,
			},
			sourceType:     "server",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Key: 'Transaction.TransactionID' Error:Field validation for 'TransactionID' failed on the 'required' tag",
		},
		{
			name: "Invalid amount",
			transaction: data.Transaction{
				TransactionID: "test-789",
				State:         "win",
				Amount:        -10.00,
			},
			sourceType:     "payment",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Key: 'Transaction.Amount' Error:Field validation for 'Amount' failed on the 'gt' tag",
		},
		{
			name: "Invalid source type",
			transaction: data.Transaction{
				TransactionID: "test-101",
				State:         "lost",
				Amount:        200.00,
			},
			sourceType:     "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Key: 'Transaction.SourceType' Error:Field validation for 'SourceType' failed on the 'oneof' tag",
		},
		{
			name: "Missing source type",
			transaction: data.Transaction{
				TransactionID: "test-101",
				State:         "lost",
				Amount:        200.00,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Key: 'Transaction.SourceType' Error:Field validation for 'SourceType' failed on the 'required' tag",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			body, err := json.Marshal(tc.transaction)
			s.Require().NoError(err)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.serverURL+"/api/v1/transactions", bytes.NewBuffer(body))
			s.Require().NoError(err)

			req.Header.Set("Content-Type", "application/json")
			if tc.sourceType != "" {
				req.Header.Set("Source-Type", tc.sourceType)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			s.Require().NoError(err)
			defer resp.Body.Close()

			s.Equal(tc.expectedStatus, resp.StatusCode)

			if tc.expectedError != "" {
				var errorResponse map[string]string
				err = json.NewDecoder(resp.Body).Decode(&errorResponse)
				s.Require().NoError(err)
				s.Contains(errorResponse["error"], tc.expectedError)
			} else {
				var successResponse struct {
					Message     string           `json:"message"`
					Balance     float64          `json:"balance"`
					Transaction data.Transaction `json:"transaction"`
				}
				err = json.NewDecoder(resp.Body).Decode(&successResponse)
				s.Require().NoError(err)

				s.Equal("Transaction processed successfully", successResponse.Message)
				s.NotZero(successResponse.Balance)
				s.Equal(tc.transaction.TransactionID, successResponse.Transaction.TransactionID)
				s.Equal(tc.transaction.State, successResponse.Transaction.State)
				s.Equal(tc.transaction.Amount, successResponse.Transaction.Amount)
				s.Equal(tc.sourceType, successResponse.Transaction.SourceType)
				s.NotZero(successResponse.Transaction.ID)
				s.NotZero(successResponse.Transaction.CreatedAt)
			}
		})
	}
}

func (s *IntegrationTestSuite) TestTransactionPropertyBased() {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 20
	parameters.MaxSize = 50

	properties := gopter.NewProperties(parameters)

	properties.Property("Transaction integrity and balance consistency", prop.ForAll(
		func(transactions []data.Transaction) bool {
			s.SetupTest()
			initialBalance := 1000.0
			expectedBalance := initialBalance

			for _, tx := range transactions {
				_, err := s.repo.ProcessTransaction(context.Background(), &tx)
				if err != nil {
					if errors.Is(err, repository.ErrInsufficientFunds) || errors.Is(err, repository.ErrDuplicateTransaction) {
						continue
					}
					return false
				}

				// Update expected balance independently
				if tx.State == "win" {
					expectedBalance += tx.Amount
				} else if tx.State == "lost" && expectedBalance >= tx.Amount {
					expectedBalance -= tx.Amount
				}

			}

			return s.verifyFinalState(expectedBalance)
		},
		gen.SliceOfN(20, genTransaction()),
	))

	properties.TestingRun(s.T())
}

func (s *IntegrationTestSuite) verifyFinalState(expectedBalance float64) bool {
	var actualBalance float64
	err := s.testDB.DBInstance.QueryRow(context.Background(), "SELECT balance FROM account WHERE id = 1").Scan(&actualBalance)
	if err != nil {
		s.T().Errorf("Failed to fetch final balance: %v", err)
		return false
	}

	balanceCorrect := math.Abs(actualBalance-expectedBalance) < 0.01
	balanceNonNegative := actualBalance >= 0

	if !balanceCorrect || !balanceNonNegative {
		s.T().Errorf("Test failed: Balance correct: %v, Balance non-negative: %v", balanceCorrect, balanceNonNegative)
		s.T().Errorf("Expected balance: %.2f, Actual balance: %.2f", expectedBalance, actualBalance)
	}

	return balanceCorrect && balanceNonNegative
}

func genTransaction() gopter.Gen {
	return gen.Struct(reflect.TypeOf(data.Transaction{}), map[string]gopter.Gen{
		"TransactionID": gen.Identifier(),
		"SourceType":    gen.OneConstOf("game", "server", "payment"),
		"State":         gen.OneConstOf("win", "lost"),
		"Amount":        gen.Float64Range(0.01, 1000.00), // Reduced max amount
	})
}
