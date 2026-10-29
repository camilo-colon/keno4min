package main

import (
	"github.com/cronos/keno4min/internal/config"
	"github.com/cronos/keno4min/internal/draws/drawsfx"
	"github.com/cronos/keno4min/internal/platform/awsfx"
	dynamodbfx "github.com/cronos/keno4min/internal/platform/awsfx/dynamodbfx"
	"github.com/cronos/keno4min/internal/platform/serverfx"
	"github.com/cronos/keno4min/internal/tickets/ticketsfx"
	"go.uber.org/fx"
)

func main() {
	fx.New(apiModule()).Run()
}

func apiModule() fx.Option {
	return fx.Options(
		fx.Provide(config.Load),
		awsfx.Module,
		dynamodbfx.Module,
		drawsfx.Module,
		ticketsfx.Module,
		serverfx.Module,
	)
}
