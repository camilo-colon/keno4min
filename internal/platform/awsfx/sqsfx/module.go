package sqsfx

import "go.uber.org/fx"

var Module = fx.Module("sqs", fx.Provide(NewSqsClient))
