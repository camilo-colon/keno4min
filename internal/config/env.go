// Package config loads and validates process configuration.
package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAppName         = "keno4min"
	defaultEnvironment     = "development"
	defaultHTTPAddress     = ":8080"
	defaultReadTimeout     = 10 * time.Second
	defaultWriteTimeout    = 15 * time.Second
	defaultIdleTimeout     = 60 * time.Second
	defaultShutdownTimeout = 10 * time.Second
	defaultBodyLimit       = 4 * 1024 * 1024
	defaultProxyHeader     = "X-Forwarded-For"
	defaultAWSRegion       = "us-east-1"
	defaultDynamoDBTable   = "keno4min-local"
)

var dynamoDBTableNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// Config contains all process configuration.
type Config struct {
	AppName     string
	Environment string
	HTTP        HTTPConfig
	AWS         AWSConfig
	DynamoDB    DynamoDBConfig
}

// AWSConfig contains configuration shared by AWS service clients.
type AWSConfig struct {
	Region string
}

// DynamoDBConfig identifies the table provisioned for this service.
type DynamoDBConfig struct {
	TableName   string
	EndpointURL string
}

// HTTPConfig controls the HTTP server and its browser-facing policy.
type HTTPConfig struct {
	Address         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	BodyLimit       int
	AllowedOrigins  []string
	ProxyHeader     string
	TrustedProxies  []string
}

// IsDevelopment reports whether development-only diagnostics may be enabled.
func (c Config) IsDevelopment() bool {
	return strings.EqualFold(c.Environment, "development")
}

// Load reads configuration from environment variables and validates it.
func Load() (Config, error) {
	return load(os.LookupEnv)
}

type lookupEnv func(string) (string, bool)

func load(lookup lookupEnv) (Config, error) {
	environment := stringValue(lookup, "APP_ENV", defaultEnvironment)
	localEnvironment := isLocalEnvironment(environment)

	awsRegionFallback := ""
	dynamoDBTableFallback := ""
	if localEnvironment {
		awsRegionFallback = defaultAWSRegion
		dynamoDBTableFallback = defaultDynamoDBTable
	}

	awsRegion := stringValue(lookup, "AWS_REGION", awsRegionFallback)
	if awsRegion == "" {
		return Config{}, fmt.Errorf("AWS_REGION must not be empty outside development or test")
	}

	dynamoDBTable := stringValue(lookup, "DYNAMODB_TABLE_NAME", dynamoDBTableFallback)
	if err := validateDynamoDBTableName(dynamoDBTable); err != nil {
		return Config{}, fmt.Errorf("DYNAMODB_TABLE_NAME: %w", err)
	}

	dynamoDBEndpoint := stringValue(lookup, "DYNAMODB_ENDPOINT_URL", "")
	if dynamoDBEndpoint != "" {
		if !localEnvironment {
			return Config{}, fmt.Errorf("DYNAMODB_ENDPOINT_URL is only allowed in development or test")
		}
		if err := validateDynamoDBEndpoint(dynamoDBEndpoint); err != nil {
			return Config{}, fmt.Errorf("DYNAMODB_ENDPOINT_URL: %w", err)
		}
	}

	readTimeout, err := durationValue(lookup, "HTTP_READ_TIMEOUT", defaultReadTimeout)
	if err != nil {
		return Config{}, err
	}

	writeTimeout, err := durationValue(lookup, "HTTP_WRITE_TIMEOUT", defaultWriteTimeout)
	if err != nil {
		return Config{}, err
	}

	idleTimeout, err := durationValue(lookup, "HTTP_IDLE_TIMEOUT", defaultIdleTimeout)
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := durationValue(lookup, "HTTP_SHUTDOWN_TIMEOUT", defaultShutdownTimeout)
	if err != nil {
		return Config{}, err
	}

	bodyLimit, err := intValue(lookup, "HTTP_BODY_LIMIT_BYTES", defaultBodyLimit)
	if err != nil {
		return Config{}, err
	}

	allowedOrigins := csvValue(lookup, "HTTP_ALLOWED_ORIGINS")
	if err := validateOrigins(allowedOrigins); err != nil {
		return Config{}, fmt.Errorf("HTTP_ALLOWED_ORIGINS: %w", err)
	}

	trustedProxies := csvValue(lookup, "HTTP_TRUSTED_PROXIES")
	if err := validateProxies(trustedProxies); err != nil {
		return Config{}, fmt.Errorf("HTTP_TRUSTED_PROXIES: %w", err)
	}

	return Config{
		AppName:     stringValue(lookup, "APP_NAME", defaultAppName),
		Environment: environment,
		HTTP: HTTPConfig{
			Address:         stringValue(lookup, "HTTP_ADDR", defaultHTTPAddress),
			ReadTimeout:     readTimeout,
			WriteTimeout:    writeTimeout,
			IdleTimeout:     idleTimeout,
			ShutdownTimeout: shutdownTimeout,
			BodyLimit:       bodyLimit,
			AllowedOrigins:  allowedOrigins,
			ProxyHeader:     stringValue(lookup, "HTTP_PROXY_HEADER", defaultProxyHeader),
			TrustedProxies:  trustedProxies,
		},
		AWS: AWSConfig{
			Region: awsRegion,
		},
		DynamoDB: DynamoDBConfig{
			TableName:   dynamoDBTable,
			EndpointURL: dynamoDBEndpoint,
		},
	}, nil
}

func isLocalEnvironment(environment string) bool {
	return strings.EqualFold(environment, "development") || strings.EqualFold(environment, "test")
}

func stringValue(lookup lookupEnv, key, fallback string) string {
	value, ok := lookup(key)
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		return fallback
	}
	return value
}

func durationValue(lookup lookupEnv, key string, fallback time.Duration) (time.Duration, error) {
	raw := stringValue(lookup, key, fallback.String())
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration: %q", key, raw)
	}
	return value, nil
}

func intValue(lookup lookupEnv, key string, fallback int) (int, error) {
	raw := stringValue(lookup, key, strconv.Itoa(fallback))
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer: %q", key, raw)
	}
	return value, nil
}

func csvValue(lookup lookupEnv, key string) []string {
	raw, ok := lookup(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return nil
	}

	values := make([]string, 0)
	for value := range strings.SplitSeq(raw, ",") {
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
		}
	}
	return values
}

func validateOrigins(origins []string) error {
	for _, origin := range origins {
		if origin == "*" {
			continue
		}

		parsed, err := url.Parse(origin)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
			parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return fmt.Errorf("invalid origin %q", origin)
		}
	}
	return nil
}

func validateProxies(proxies []string) error {
	for _, proxy := range proxies {
		if net.ParseIP(proxy) != nil {
			continue
		}
		if _, _, err := net.ParseCIDR(proxy); err != nil {
			return fmt.Errorf("invalid IP address or CIDR %q", proxy)
		}
	}
	return nil
}

func validateDynamoDBTableName(name string) error {
	if len(name) < 3 || len(name) > 255 {
		return fmt.Errorf("must contain between 3 and 255 characters")
	}
	if !dynamoDBTableNamePattern.MatchString(name) {
		return fmt.Errorf("must contain only letters, numbers, underscores, hyphens, or periods")
	}
	return nil
}

func validateDynamoDBEndpoint(endpoint string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("must be an absolute HTTP(S) URL without credentials, query, or fragment")
	}
	return nil
}
