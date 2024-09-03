package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

func NewPostgresDB(dsn string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Adjust connection pool settings
	config.MaxConns = 20                      // Reduced from 100
	config.MinConns = 5                       // Reduced from 10
	config.MaxConnLifetime = 30 * time.Minute // Reduced from 1 hour
	config.MaxConnIdleTime = 5 * time.Minute  // Reduced from 15 minutes

	// Use a timeout for the entire connection process
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := pgxpool.ConnectConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Ping the database to verify the connection
	if err := db.Ping(ctx); err != nil {
		db.Close() // Ensure we close the pool if ping fails
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
