package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName     string
	Environment string
	HTTP        HttpConfig
	Dynamodb    DynamodbConfig
}

type HttpConfig struct {
	Address string
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
		Environment: GetString("ENVIRONMENT", "development"),
		HTTP: HttpConfig{
			Address: GetString("HTTP_ADDRESS", ":8080"),
		},
		Dynamodb: DynamodbConfig{
			TableName: GetString("DYNAMODB_TABLE", ""),
			Endpoint:  GetString("DYNAMODB_ENDPOINT", ""),
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
