// Command api runs the Keno4min HTTP API.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/cronos/keno4min/internal/config"
	"github.com/cronos/keno4min/internal/platform/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	httpServer := server.New(cfg, logger)
	if err := httpServer.Run(ctx); err != nil {
		logger.Error("HTTP server failed", "error", err)
		os.Exit(1)
	}
}
