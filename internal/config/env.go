package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName     string
	Environment string
	HTTP        HttpConfig
	Dynamodb    DynamodbConfig
	Core        CoreConfig
}

type HttpConfig struct {
	Address string
}

// CoreConfig points to the core service that owns the node balance.
type CoreConfig struct {
	BaseURL string
	Timeout time.Duration
}

type DynamodbConfig struct {
	TableName string
	Endpoint  string
}

func Load() (*Config, error) {
	// The .env file is optional: containers and deployed environments
	// receive their configuration as real environment variables.
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("load .env file: %w", err)
	}

	return &Config{
		AppName:     GetString("APP_NAME", "Cronos Keno4min API"),
		Environment: GetString("ENVIRONMENT", "dev"),
		HTTP: HttpConfig{
			Address: GetString("HTTP_ADDRESS", ":8080"),
		},
		Dynamodb: DynamodbConfig{
			TableName: GetString("DYNAMODB_TABLE", ""),
			Endpoint:  GetString("DYNAMODB_ENDPOINT", ""),
		},
		Core: CoreConfig{
			BaseURL: GetString("CORE_BASE_URL", ""),
			Timeout: GetDuration("CORE_TIMEOUT", 5*time.Second),
		},
	}, nil
}

func (c *Config) IsDevelopment() bool {
	return strings.EqualFold(c.Environment, "dev")
}

func GetString(key string, fallback string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return val
}

// GetDuration parses values such as "5s" or "500ms" and falls back when absent or invalid.
func GetDuration(key string, fallback time.Duration) time.Duration {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	parsed, err := time.ParseDuration(val)
	if err != nil {
		log.Printf("invalid duration for %s=%q, using %s", key, val, fallback)
		return fallback
	}
	return parsed
}
