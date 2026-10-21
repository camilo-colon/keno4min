package usersfx

import (
	"github.com/cronos/keno4min/internal/common"
	"github.com/cronos/keno4min/internal/users/adapters/dynamodb"
	"github.com/cronos/keno4min/internal/users/ports"
	"github.com/cronos/keno4min/internal/users/subscribers"
	"github.com/cronos/keno4min/internal/users/usecase"
	"go.uber.org/fx"
)

var WorkerModule = fx.Module(
	"users.worker",
	fx.Provide(
		fx.Annotate(dynamodb.NewDynamodbRepository, fx.As(new(ports.Repository))),
		usecase.NewCreate,
		fx.Annotate(
			subscribers.NewUserCreatedSubscriber,
			fx.As(new(common.DomainEventSubscriber)),
			fx.ResultTags(`group:"subscribers"`),
		),
	),
)
