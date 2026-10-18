package awsfx

import "go.uber.org/fx"

var Module = fx.Module("aws", fx.Provide(NewConfig))
