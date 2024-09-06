// File: internal/config/config.go

package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Port   int          `mapstructure:"PORT"`
	DB     DBConfig     `mapstructure:"DB"`
	Worker WorkerConfig `mapstructure:"WORKER"`
}

type DBConfig struct {
	DSN string `mapstructure:"DSN"`
}

type WorkerConfig struct {
	Interval time.Duration `mapstructure:"INTERVAL"`
}

func Load() (*Config, error) {
	v := viper.New()

	// Set default values
	v.SetDefault("PORT", 4000)
	v.SetDefault("DB.DSN", "postgres://username:password@host:port/database_name?sslmode=disable")
	v.SetDefault("WORKER.INTERVAL", 15*time.Second)

	// Look for .env file
	v.SetConfigFile(".env")
	v.SetConfigType("env")

	// Read .env file if it exists
	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file, %s. Using defaults and environment variables.\n", err)
	}

	// Override with environment variables
	v.AutomaticEnv()
	v.SetEnvPrefix("TRANSACTION_PROCESSOR")

	// Replace dots with underscores for nested keys in environment variables
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	return &config, nil
}
