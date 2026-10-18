package draw

import (
	"github.com/cronos/keno4min/internal/draws/adapters/dynamodb"
	"github.com/cronos/keno4min/internal/draws/adapters/http"
	"github.com/cronos/keno4min/internal/draws/ports"
	"github.com/cronos/keno4min/internal/draws/usecase"
	"github.com/cronos/keno4min/internal/platform/serverfx"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"draw",

	fx.Provide(
		fx.Annotate(
			dynamodb.NewRepository,
			fx.As(new(ports.Repository)),
		),

		//casos de uso
		usecase.NewGetRuntime,

		//handlers
		http.NewGetRuntimeHandler,

		fx.Annotate(
			http.NewRoutes,
			fx.As(new(serverfx.Route)),
			fx.ResultTags(`group:"routes"`),
		),
	),
)
