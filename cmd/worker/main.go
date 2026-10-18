package main

import (
	"github.com/cronos/keno4min/internal/config"
	"github.com/cronos/keno4min/internal/nodes/nodesfx"
	dynamodbfx "github.com/cronos/keno4min/internal/platform/awsfx/dynamodbfx"
	"github.com/cronos/keno4min/internal/platform/awsfx/sqsfx"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(config.Load),
		dynamodbfx.Module,
		sqsfx.Module,
		nodesfx.WorkerModule,
	)
}
