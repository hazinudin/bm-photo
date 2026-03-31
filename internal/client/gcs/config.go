package gcs

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds GCS client configuration
type Config struct {
	BucketName          string
	CredentialsPath     string
	TestPrefix          string
	SignedURLExpiryMins int
	ConnectTimeoutSecs  int
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() (Config, error) {
	config := Config{
		TestPrefix:          getEnv("GCS_TEST_PREFIX", "test/"),
		SignedURLExpiryMins: getEnvAsInt("GCS_SIGNED_URL_EXPIRY_MINUTES", 15),
		ConnectTimeoutSecs:  getEnvAsInt("GCS_CONNECT_TIMEOUT_SECONDS", 30),
	}

	config.BucketName = os.Getenv("GCS_BUCKET_NAME")
	if config.BucketName == "" {
		return Config{}, fmt.Errorf("GCS_BUCKET_NAME environment variable is required")
	}

	config.CredentialsPath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if config.CredentialsPath == "" {
		return Config{}, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS environment variable is required")
	}

	return config, nil
}

// Validate validates the configuration
func (c Config) Validate() error {
	if c.BucketName == "" {
		return fmt.Errorf("bucket name is required")
	}
	if c.CredentialsPath == "" {
		return fmt.Errorf("credentials path is required")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
