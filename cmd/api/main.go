package main

import (
	"github.com/cronos/keno4min/internal/config"
	draw "github.com/cronos/keno4min/internal/draws"
	"github.com/cronos/keno4min/internal/platform/awsfx"
	dynamodbfx "github.com/cronos/keno4min/internal/platform/awsfx/dynamodbfx"
	"github.com/cronos/keno4min/internal/platform/serverfx"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(config.Load),
		awsfx.Module,
		dynamodbfx.Module,
		draw.Module,
		serverfx.Module,
	).Run()
}
