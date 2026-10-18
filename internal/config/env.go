// Package config loads and validates process configuration.
package config

import (
	"log"
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
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
		return nil, err
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
	return strings.EqualFold(c.Environment, "development")
}

func GetString(key string, fallback string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return val
}
