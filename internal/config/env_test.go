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
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := load(mapLookup(map[string]string{
		"APP_NAME":              "api",
		"APP_ENV":               "production",
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
