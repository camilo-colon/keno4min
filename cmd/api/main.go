package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"cronos.bet/keno4min/internal/config"
	"cronos.bet/keno4min/internal/server/httpapi"
)

func main() {
	if err := execute(); err != nil {
		log.Printf("run API server: %v", err)
		os.Exit(1)
	}
}

func execute() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return run(ctx, cfg)
}

func run(ctx context.Context, cfg config.Config) error {
	if err := ctx.Err(); err != nil {
		return nil
	}

	listener, err := net.Listen("tcp", cfg.HTTPAddress)
	if err != nil {
		return fmt.Errorf("listen on configured HTTP address: %w", err)
	}
	defer func() {
		if closeErr := listener.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
			log.Printf("close HTTP listener: %v", closeErr)
		}
	}()

	server := httpapi.New(httpapi.Config{
		BodyLimit:       cfg.HTTPBodyLimit,
		ReadTimeout:     cfg.HTTPReadTimeout,
		WriteTimeout:    cfg.HTTPWriteTimeout,
		IdleTimeout:     cfg.HTTPIdleTimeout,
		ShutdownTimeout: cfg.HTTPShutdownTimeout,
	})
	if err := server.Serve(ctx, listener); err != nil {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	return nil
}
