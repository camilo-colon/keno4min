package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := load(mapLookup(nil))
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}

	if cfg.AppName != defaultAppName || cfg.HTTP.Address != defaultHTTPAddress {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.HTTP.ReadTimeout != defaultReadTimeout || cfg.HTTP.ShutdownTimeout != defaultShutdownTimeout {
		t.Fatalf("unexpected timeout defaults: %+v", cfg.HTTP)
	}
	if cfg.HTTP.BodyLimit != defaultBodyLimit {
		t.Fatalf("unexpected body limit: %d", cfg.HTTP.BodyLimit)
	}
	if cfg.AWS.Region != defaultAWSRegion {
		t.Fatalf("unexpected AWS region: %q", cfg.AWS.Region)
	}
	if cfg.DynamoDB.TableName != defaultDynamoDBTable || cfg.DynamoDB.EndpointURL != "" {
		t.Fatalf("unexpected DynamoDB defaults: %+v", cfg.DynamoDB)
	}
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := load(mapLookup(map[string]string{
		"APP_NAME":              "api",
		"APP_ENV":               "production",
		"AWS_REGION":            "us-west-2",
		"DYNAMODB_TABLE_NAME":   "keno4min-production",
		"HTTP_ADDR":             "127.0.0.1:9090",
		"HTTP_READ_TIMEOUT":     "3s",
		"HTTP_WRITE_TIMEOUT":    "4s",
		"HTTP_IDLE_TIMEOUT":     "30s",
		"HTTP_SHUTDOWN_TIMEOUT": "8s",
		"HTTP_BODY_LIMIT_BYTES": "1024",
		"HTTP_ALLOWED_ORIGINS":  "https://example.com, https://*.example.org",
		"HTTP_PROXY_HEADER":     "X-Real-IP",
		"HTTP_TRUSTED_PROXIES":  "127.0.0.1,10.0.0.0/8",
	}))
	if err != nil {
		t.Fatalf("load overrides: %v", err)
	}

	if cfg.AppName != "api" || cfg.Environment != "production" || cfg.IsDevelopment() {
		t.Fatalf("unexpected application config: %+v", cfg)
	}
	if cfg.HTTP.Address != "127.0.0.1:9090" || cfg.HTTP.ReadTimeout != 3*time.Second || cfg.HTTP.BodyLimit != 1024 {
		t.Fatalf("unexpected HTTP config: %+v", cfg.HTTP)
	}
	if len(cfg.HTTP.AllowedOrigins) != 2 || len(cfg.HTTP.TrustedProxies) != 2 {
		t.Fatalf("unexpected list config: %+v", cfg.HTTP)
	}
	if cfg.AWS.Region != "us-west-2" || cfg.DynamoDB.TableName != "keno4min-production" {
		t.Fatalf("unexpected AWS configuration: AWS=%+v DynamoDB=%+v", cfg.AWS, cfg.DynamoDB)
	}
}

func TestLoadAllowsDynamoDBEndpointInLocalEnvironments(t *testing.T) {
	for _, environment := range []string{"development", "test"} {
		t.Run(environment, func(t *testing.T) {
			cfg, err := load(mapLookup(map[string]string{
				"APP_ENV":               environment,
				"DYNAMODB_ENDPOINT_URL": "http://localhost:8000",
			}))
			if err != nil {
				t.Fatalf("load local DynamoDB endpoint: %v", err)
			}
			if cfg.DynamoDB.EndpointURL != "http://localhost:8000" {
				t.Fatalf("unexpected DynamoDB endpoint: %q", cfg.DynamoDB.EndpointURL)
			}
		})
	}
}

func TestLoadRequiresAWSConfigurationOutsideLocalEnvironments(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]string
		key    string
	}{
		{
			name: "region",
			values: map[string]string{
				"APP_ENV":             "production",
				"DYNAMODB_TABLE_NAME": "keno4min-production",
			},
			key: "AWS_REGION",
		},
		{
			name: "table",
			values: map[string]string{
				"APP_ENV":    "staging",
				"AWS_REGION": "us-east-1",
			},
			key: "DYNAMODB_TABLE_NAME",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := load(mapLookup(tt.values))
			if err == nil || !strings.Contains(err.Error(), tt.key) {
				t.Fatalf("expected error for %s, got %v", tt.key, err)
			}
		})
	}
}

func TestLoadRejectsDynamoDBEndpointOutsideLocalEnvironments(t *testing.T) {
	_, err := load(mapLookup(map[string]string{
		"APP_ENV":               "production",
		"AWS_REGION":            "us-east-1",
		"DYNAMODB_TABLE_NAME":   "keno4min-production",
		"DYNAMODB_ENDPOINT_URL": "https://example.com",
	}))
	if err == nil || !strings.Contains(err.Error(), "DYNAMODB_ENDPOINT_URL") {
		t.Fatalf("expected production endpoint error, got %v", err)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "duration", key: "HTTP_READ_TIMEOUT", value: "soon"},
		{name: "non-positive duration", key: "HTTP_IDLE_TIMEOUT", value: "0s"},
		{name: "body limit", key: "HTTP_BODY_LIMIT_BYTES", value: "0"},
		{name: "origin", key: "HTTP_ALLOWED_ORIGINS", value: "example.com"},
		{name: "proxy", key: "HTTP_TRUSTED_PROXIES", value: "not-an-ip"},
		{name: "short DynamoDB table", key: "DYNAMODB_TABLE_NAME", value: "ab"},
		{name: "invalid DynamoDB table", key: "DYNAMODB_TABLE_NAME", value: "invalid/table"},
		{name: "DynamoDB endpoint", key: "DYNAMODB_ENDPOINT_URL", value: "localhost:8000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := load(mapLookup(map[string]string{tt.key: tt.value}))
			if err == nil || !strings.Contains(err.Error(), tt.key) {
				t.Fatalf("expected error for %s, got %v", tt.key, err)
			}
		})
	}
}

func mapLookup(values map[string]string) lookupEnv {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
