// File: internal/testutil/postgres_container.go

package testutil

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"time"

	"github.com/golang-migrate/migrate/v4"

	_ "github.com/golang-migrate/migrate/v4/database/postgres" // Import postgres driver
	_ "github.com/golang-migrate/migrate/v4/source/file"       // Import file source driver
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type PostgresContainer struct {
	Container testcontainers.Container
	Pool      *pgxpool.Pool
	Config    PostgresConfig
}

type PostgresConfig struct {
	User     string
	Password string
	DBName   string
}

func NewPostgresContainer(ctx context.Context, config PostgresConfig) (*PostgresContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     config.User,
			"POSTGRES_PASSWORD": config.Password,
			"POSTGRES_DB":       config.DBName,
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithStartupTimeout(time.Minute),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return nil, fmt.Errorf("failed to get container external port: %w", err)
	}

	hostIP, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		config.User, config.Password, hostIP, mappedPort.Port(), config.DBName)

	var pool *pgxpool.Pool
	var poolErr error
	for i := 0; i < 5; i++ { // Retry 5 times
		pool, poolErr = pgxpool.Connect(ctx, dbURL)
		if poolErr == nil {
			break
		}
		log.Printf("Failed to connect to database, retrying in 2 seconds... (Attempt %d/5)", i+1)
		time.Sleep(time.Second * 2) // Wait for 2 seconds before retrying
	}
	if poolErr != nil {
		return nil, fmt.Errorf("failed to connect to database after retries: %w", poolErr)
	}

	return &PostgresContainer{
		Container: container,
		Pool:      pool,
		Config:    config,
	}, nil
}

func (pc *PostgresContainer) MigrateDB(ctx context.Context) error {
	_, path, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("failed to get path")
	}
	pathToMigrationFiles := filepath.Dir(path) + "/../../migrations"

	hostIP, err := pc.Container.Host(ctx)
	if err != nil {
		return fmt.Errorf("failed to get container host: %w", err)
	}

	mappedPort, err := pc.Container.MappedPort(ctx, "5432")
	if err != nil {
		return fmt.Errorf("failed to get container external port: %w", err)
	}

	databaseURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		pc.Config.User, pc.Config.Password, hostIP, mappedPort.Port(), pc.Config.DBName)

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

func (pc *PostgresContainer) Terminate(ctx context.Context) error {
	pc.Pool.Close()
	return pc.Container.Terminate(ctx)
}
