package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultPort           = 4000
	defaultEnv            = "development"
	defaultWorkerInterval = 15 * time.Second
	defaultDBDSN          = "postgres://transactions:pa55word@localhost:5432/transactions?sslmode=disable"
	envPrefix             = "TRANSACTION_PROCESSOR_"
	envPort               = envPrefix + "PORT"
	envEnvironment        = envPrefix + "ENV"
	envDBDSN              = envPrefix + "DB_DSN"
	envWorkerInterval     = envPrefix + "WORKER_INTERVAL"
)

type Config struct {
	Port int
	Env  string
	DB   struct {
		DSN string
	}
	Worker struct {
		Interval time.Duration
	}
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	cfg := &Config{
		Port: defaultPort,
		Env:  defaultEnv,
		DB: struct{ DSN string }{
			DSN: defaultDBDSN,
		},
		Worker: struct{ Interval time.Duration }{
			Interval: defaultWorkerInterval,
		},
	}

	if err := cfg.loadFromEnv(); err != nil {
		return nil, err
	}

	cfg.loadFromFlags()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (cfg *Config) loadFromEnv() error {
	var err error

	cfg.Port, err = parseEnvInt(envPort, cfg.Port)
	if err != nil {
		return err
	}

	cfg.Env = getEnv(envEnvironment, cfg.Env)
	cfg.DB.DSN = getEnv(envDBDSN, cfg.DB.DSN)

	cfg.Worker.Interval, err = parseEnvDuration(envWorkerInterval, cfg.Worker.Interval)
	if err != nil {
		return err
	}

	return nil
}

func (cfg *Config) loadFromFlags() {
	flag.IntVar(&cfg.Port, "port", cfg.Port, "API server port")
	flag.StringVar(&cfg.Env, "env", cfg.Env, "Environment (development|staging|production)")
	flag.StringVar(&cfg.DB.DSN, "db-dsn", cfg.DB.DSN, "PostgreSQL DSN")
	flag.DurationVar(&cfg.Worker.Interval, "worker-interval", cfg.Worker.Interval, "Override post-processor interval (e.g. '5m')")

	flag.Parse()
}

func (cfg *Config) validate() error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if cfg.DB.DSN == "" {
		return fmt.Errorf("database DSN is required")
	}
	if cfg.Port <= 0 {
		return fmt.Errorf("invalid port number: %d", cfg.Port)
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func parseEnvInt(key string, fallback int) (int, error) {
	if strValue, exists := os.LookupEnv(key); exists {
		value, err := strconv.Atoi(strValue)
		if err != nil {
			return fallback, fmt.Errorf("invalid value for %s: %w", key, err)
		}
		return value, nil
	}
	return fallback, nil
}

func parseEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	if strValue, exists := os.LookupEnv(key); exists {
		value, err := time.ParseDuration(strValue)
		if err != nil {
			return fallback, fmt.Errorf("invalid duration for %s: %w\nValid time units are ns, us, ms, s, m, h", key, err)
		}
		return value, nil
	}
	return fallback, nil
}
