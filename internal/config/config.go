// Package config loads and validates process configuration.
package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

const (
	defaultHTTPAddress         = ":8080"
	defaultHTTPBodyLimit       = 4 * 1024 * 1024
	defaultHTTPReadTimeout     = 5 * time.Second
	defaultHTTPWriteTimeout    = 10 * time.Second
	defaultHTTPIdleTimeout     = 60 * time.Second
	defaultHTTPShutdownTimeout = 10 * time.Second
)

const (
	envHTTPAddress         = "KENO4MIN_HTTP_ADDRESS"
	envHTTPBodyLimit       = "KENO4MIN_HTTP_BODY_LIMIT_BYTES"
	envHTTPReadTimeout     = "KENO4MIN_HTTP_READ_TIMEOUT"
	envHTTPWriteTimeout    = "KENO4MIN_HTTP_WRITE_TIMEOUT"
	envHTTPIdleTimeout     = "KENO4MIN_HTTP_IDLE_TIMEOUT"
	envHTTPShutdownTimeout = "KENO4MIN_HTTP_SHUTDOWN_TIMEOUT"
)

// Config contains the validated settings used by the API process.
type Config struct {
	HTTPAddress         string
	HTTPBodyLimit       int
	HTTPReadTimeout     time.Duration
	HTTPWriteTimeout    time.Duration
	HTTPIdleTimeout     time.Duration
	HTTPShutdownTimeout time.Duration
}

// Load reads configuration from the process environment.
func Load() (Config, error) {
	return load(os.LookupEnv)
}

type lookupEnv func(string) (string, bool)

func load(lookup lookupEnv) (Config, error) {
	cfg := Config{
		HTTPAddress:         defaultHTTPAddress,
		HTTPBodyLimit:       defaultHTTPBodyLimit,
		HTTPReadTimeout:     defaultHTTPReadTimeout,
		HTTPWriteTimeout:    defaultHTTPWriteTimeout,
		HTTPIdleTimeout:     defaultHTTPIdleTimeout,
		HTTPShutdownTimeout: defaultHTTPShutdownTimeout,
	}

	if value, ok := lookup(envHTTPAddress); ok {
		cfg.HTTPAddress = value
	}

	var err error
	if cfg.HTTPBodyLimit, err = intFromEnvironment(lookup, envHTTPBodyLimit, cfg.HTTPBodyLimit); err != nil {
		return Config{}, err
	}
	if cfg.HTTPReadTimeout, err = durationFromEnvironment(lookup, envHTTPReadTimeout, cfg.HTTPReadTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTPWriteTimeout, err = durationFromEnvironment(lookup, envHTTPWriteTimeout, cfg.HTTPWriteTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTPIdleTimeout, err = durationFromEnvironment(lookup, envHTTPIdleTimeout, cfg.HTTPIdleTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTPShutdownTimeout, err = durationFromEnvironment(lookup, envHTTPShutdownTimeout, cfg.HTTPShutdownTimeout); err != nil {
		return Config{}, err
	}

	if err := validateAddress(cfg.HTTPAddress); err != nil {
		return Config{}, fmt.Errorf("%s: %w", envHTTPAddress, err)
	}

	return cfg, nil
}

func intFromEnvironment(lookup lookupEnv, name string, fallback int) (int, error) {
	value, ok := lookup(name)
	if !ok {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}
	return parsed, nil
}

func durationFromEnvironment(lookup lookupEnv, name string, fallback time.Duration) (time.Duration, error) {
	value, ok := lookup(name)
	if !ok {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a Go duration: %w", name, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}
	return parsed, nil
}

func validateAddress(address string) error {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("must be a host:port listen address: %w", err)
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}
