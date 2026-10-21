package main

import (
	"testing"

	"github.com/cronos/keno4min/internal/config"
	"github.com/cronos/keno4min/internal/nodes/nodesfx"
	"github.com/cronos/keno4min/internal/platform/awsfx"
	"github.com/cronos/keno4min/internal/platform/awsfx/dynamodbfx"
	"github.com/cronos/keno4min/internal/platform/awsfx/sqsfx"
	"github.com/cronos/keno4min/internal/platform/workerfx"
	"github.com/cronos/keno4min/internal/users/usersfx"
	"go.uber.org/fx"
)

func TestDependencyGraph(t *testing.T) {
	err := fx.ValidateApp(
		fx.NopLogger,
		fx.Provide(config.Load),
		awsfx.Module,
		dynamodbfx.Module,
		sqsfx.Module,
		nodesfx.WorkerModule,
		usersfx.WorkerModule,
		workerfx.Module,
	)
	if err != nil {
		t.Fatalf("invalid worker dependency graph: %v", err)
	}
}
