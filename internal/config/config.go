package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port int
	Env  string
	DB   struct {
		DSN string
	}
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	cfg := &Config{}

	// Set default values
	cfg.Port = 4000
	cfg.Env = "development"
	cfg.DB.DSN = os.Getenv("TRANSACTIONS_DB_DSN")

	// Override with environment variables if they exist
	if envPort := os.Getenv("PORT"); envPort != "" {
		if port, err := strconv.Atoi(envPort); err == nil {
			cfg.Port = port
		} else {
			return nil, fmt.Errorf("invalid PORT environment variable: %w", err)
		}
	}

	if envEnv := os.Getenv("ENV"); envEnv != "" {
		cfg.Env = envEnv
	}

	// Define command-line flags
	flag.IntVar(&cfg.Port, "port", cfg.Port, "API server port")
	flag.StringVar(&cfg.Env, "env", cfg.Env, "Environment (development|staging|production)")
	flag.StringVar(&cfg.DB.DSN, "db-dsn", cfg.DB.DSN, "PostgreSQL DSN")

	// Parse command-line flags (these will override env vars)
	flag.Parse()

	return cfg, nil
}
