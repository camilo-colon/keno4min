package nodesfx

import (
	"github.com/cronos/keno4min/internal/common"
	"github.com/cronos/keno4min/internal/nodes/adapters/dynamodb"
	"github.com/cronos/keno4min/internal/nodes/ports"
	"github.com/cronos/keno4min/internal/nodes/subscribers"
	"github.com/cronos/keno4min/internal/nodes/usecase"
	"go.uber.org/fx"
)

var RepositoryModule = fx.Module(
	"nodes.repository",
	fx.Provide(fx.Annotate(dynamodb.NewDynamodbRepository, fx.As(new(ports.Repository)))),
)

var WorkerModule = fx.Module(
	"nodes.worker",
	RepositoryModule,
	fx.Provide(
		usecase.NewCreate,
		fx.Annotate(
			subscribers.NewNodeCretedSubscriber,
			fx.As(new(common.DomainEventSubscriber)),
			fx.ResultTags(`group:"subscribers"`),
		),
	),
)
