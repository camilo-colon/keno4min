package workerfx

import (
	"go.uber.org/fx"
)

var Module = fx.Module(
	"worker_server",
	fx.Provide(
		fx.Annotate(
			NewWorker,
			fx.ParamTags(`group:"subscribers"`),
		),
		fx.Invoke(func(lc fx.Lifecycle, worker *Worker) {
			worker.Run(lc)
		}),
	),
)
