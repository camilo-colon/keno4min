package serverfx

import "go.uber.org/fx"

var Module = fx.Module(
	"api_server",
	fx.Provide(
		NewServer,
	),
	fx.Invoke(
		RegisterRoutes,
		Run,
	),
)
