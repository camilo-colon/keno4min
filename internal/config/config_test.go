package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadUsesDocumentedDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := load(noEnvironment)
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}

	if cfg.HTTPAddress != ":8080" {
		t.Errorf("HTTPAddress = %q, want :8080", cfg.HTTPAddress)
	}
	if cfg.HTTPBodyLimit != 4*1024*1024 {
		t.Errorf("HTTPBodyLimit = %d, want 4 MiB", cfg.HTTPBodyLimit)
	}
	if cfg.HTTPReadTimeout != 5*time.Second {
		t.Errorf("HTTPReadTimeout = %v, want 5s", cfg.HTTPReadTimeout)
	}
	if cfg.HTTPWriteTimeout != 10*time.Second {
		t.Errorf("HTTPWriteTimeout = %v, want 10s", cfg.HTTPWriteTimeout)
	}
	if cfg.HTTPIdleTimeout != 60*time.Second {
		t.Errorf("HTTPIdleTimeout = %v, want 60s", cfg.HTTPIdleTimeout)
	}
	if cfg.HTTPShutdownTimeout != 10*time.Second {
		t.Errorf("HTTPShutdownTimeout = %v, want 10s", cfg.HTTPShutdownTimeout)
	}
}

func TestLoadAppliesEnvironmentOverrides(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		envHTTPAddress:         "127.0.0.1:9090",
		envHTTPBodyLimit:       "1024",
		envHTTPReadTimeout:     "1s",
		envHTTPWriteTimeout:    "2s",
		envHTTPIdleTimeout:     "3s",
		envHTTPShutdownTimeout: "4s",
	}

	cfg, err := load(mapEnvironment(values))
	if err != nil {
		t.Fatalf("load overrides: %v", err)
	}

	if cfg.HTTPAddress != "127.0.0.1:9090" ||
		cfg.HTTPBodyLimit != 1024 ||
		cfg.HTTPReadTimeout != time.Second ||
		cfg.HTTPWriteTimeout != 2*time.Second ||
		cfg.HTTPIdleTimeout != 3*time.Second ||
		cfg.HTTPShutdownTimeout != 4*time.Second {
		t.Fatalf("unexpected overrides: %+v", cfg)
	}
}

func TestLoadRejectsInvalidEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		variable  string
		value     string
		wantError string
	}{
		{name: "empty address", variable: envHTTPAddress, value: "", wantError: envHTTPAddress},
		{name: "address without port", variable: envHTTPAddress, value: "localhost", wantError: envHTTPAddress},
		{name: "port outside range", variable: envHTTPAddress, value: ":70000", wantError: envHTTPAddress},
		{name: "body limit is not an integer", variable: envHTTPBodyLimit, value: "large", wantError: envHTTPBodyLimit},
		{name: "body limit is zero", variable: envHTTPBodyLimit, value: "0", wantError: envHTTPBodyLimit},
		{name: "read timeout is invalid", variable: envHTTPReadTimeout, value: "soon", wantError: envHTTPReadTimeout},
		{name: "write timeout is zero", variable: envHTTPWriteTimeout, value: "0s", wantError: envHTTPWriteTimeout},
		{name: "idle timeout is negative", variable: envHTTPIdleTimeout, value: "-1s", wantError: envHTTPIdleTimeout},
		{name: "shutdown timeout is invalid", variable: envHTTPShutdownTimeout, value: "10", wantError: envHTTPShutdownTimeout},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := load(mapEnvironment(map[string]string{test.variable: test.value}))
			if err == nil {
				t.Fatal("load succeeded, want error")
			}
			if !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %q, want variable %q", err, test.wantError)
			}
		})
	}
}

func noEnvironment(string) (string, bool) {
	return "", false
}

func mapEnvironment(values map[string]string) lookupEnv {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}
