package dynamodbfx

import "go.uber.org/fx"

var Module = fx.Module("dynamodb", fx.Provide(NewDynamodbClient))
