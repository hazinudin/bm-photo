// Package config provides configuration loading from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port     int    `env:"PORT" env-default:"8080"`
	LogLevel string `env:"LOG_LEVEL" env-default:"info"`
}

// DatabaseConfig holds PostgreSQL database configuration.
type DatabaseConfig struct {
	Host     string `env:"DB_HOST" env-required:"true"`
	Port     int    `env:"DB_PORT" env-required:"true"`
	Name     string `env:"DB_NAME" env-required:"true"`
	Username string `env:"DB_USERNAME" env-required:"true"`
	Password string `env:"DB_PASSWORD" env-required:"true"`
}

// GCSConfig holds Google Cloud Storage configuration.
type GCSConfig struct {
	BucketName      string `env:"GCS_BUCKET_NAME" env-required:"true"`
	CredentialsPath string `env:"GOOGLE_APPLICATION_CREDENTIALS" env-required:"true"`
	TestPrefix      string `env:"GCS_TEST_PREFIX" env-default:""`
}

// CleanupConfig holds cleanup job configuration.
type CleanupConfig struct {
	Enabled         bool
	Interval        time.Duration
	RetentionPeriod time.Duration
}

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	GCS      GCSConfig
	Cleanup  CleanupConfig
}

// Load reads configuration from environment variables.
// It automatically loads .env file if present.
// It returns an error if any required field is missing.
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:     8080,
			LogLevel: "info",
		},
	}

	// Load Server config
	if port := os.Getenv("PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("invalid PORT value: %w", err)
		}
		cfg.Server.Port = p
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		cfg.Server.LogLevel = logLevel
	}

	// Load Database config
	var dbErrs []error

	if host := os.Getenv("DB_HOST"); host != "" {
		cfg.Database.Host = host
	} else {
		dbErrs = append(dbErrs, errors.New("DB_HOST is required"))
	}

	if port := os.Getenv("DB_PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			dbErrs = append(dbErrs, fmt.Errorf("invalid DB_PORT value: %w", err))
		} else {
			cfg.Database.Port = p
		}
	} else {
		dbErrs = append(dbErrs, errors.New("DB_PORT is required"))
	}

	if name := os.Getenv("DB_NAME"); name != "" {
		cfg.Database.Name = name
	} else {
		dbErrs = append(dbErrs, errors.New("DB_NAME is required"))
	}

	if username := os.Getenv("DB_USERNAME"); username != "" {
		cfg.Database.Username = username
	} else {
		dbErrs = append(dbErrs, errors.New("DB_USERNAME is required"))
	}

	if password := os.Getenv("DB_PASSWORD"); password != "" {
		cfg.Database.Password = password
	} else {
		dbErrs = append(dbErrs, errors.New("DB_PASSWORD is required"))
	}

	// Load GCS config
	if bucketName := os.Getenv("GCS_BUCKET_NAME"); bucketName != "" {
		cfg.GCS.BucketName = bucketName
	} else {
		dbErrs = append(dbErrs, errors.New("GCS_BUCKET_NAME is required"))
	}

	if credsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credsPath != "" {
		cfg.GCS.CredentialsPath = credsPath
	} else {
		dbErrs = append(dbErrs, errors.New("GOOGLE_APPLICATION_CREDENTIALS is required"))
	}

	if testPrefix := os.Getenv("GCS_TEST_PREFIX"); testPrefix != "" {
		cfg.GCS.TestPrefix = testPrefix
	}

	// Load Cleanup config
	cfg.Cleanup.Enabled = true // default
	if cleanupEnabled := os.Getenv("CLEANUP_ENABLED"); cleanupEnabled != "" {
		enabled, err := strconv.ParseBool(cleanupEnabled)
		if err == nil {
			cfg.Cleanup.Enabled = enabled
		}
	}

	cfg.Cleanup.Interval = 5 * time.Minute // default
	if cleanupInterval := os.Getenv("CLEANUP_INTERVAL"); cleanupInterval != "" {
		duration, err := time.ParseDuration(cleanupInterval)
		if err == nil {
			cfg.Cleanup.Interval = duration
		}
	}

	cfg.Cleanup.RetentionPeriod = 24 * time.Hour // default
	if cleanupRetention := os.Getenv("CLEANUP_RETENTION"); cleanupRetention != "" {
		duration, err := time.ParseDuration(cleanupRetention)
		if err == nil {
			cfg.Cleanup.RetentionPeriod = duration
		}
	}

	if len(dbErrs) > 0 {
		return nil, fmt.Errorf("configuration errors: %v", dbErrs)
	}

	return cfg, nil
}
