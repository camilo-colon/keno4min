package main

import (
	"testing"

	"github.com/cronos/keno4min/internal/config"
	draw "github.com/cronos/keno4min/internal/draws"
	"github.com/cronos/keno4min/internal/platform/awsfx"
	dynamodbfx "github.com/cronos/keno4min/internal/platform/awsfx/dynamodbfx"
	"github.com/cronos/keno4min/internal/platform/serverfx"
	"go.uber.org/fx"
)

func TestDependencyGraph(t *testing.T) {
	err := fx.ValidateApp(
		fx.NopLogger,
		fx.Provide(config.Load),
		awsfx.Module,
		dynamodbfx.Module,
		draw.Module,
		serverfx.Module,
	)
	if err != nil {
		t.Fatalf("invalid API dependency graph: %v", err)
	}
}
