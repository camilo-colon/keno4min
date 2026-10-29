package ticketsfx

import (
	"github.com/cronos/keno4min/internal/platform/serverfx"
	"github.com/cronos/keno4min/internal/tickets/adapters/dynamodb"
	"github.com/cronos/keno4min/internal/tickets/adapters/http"
	"github.com/cronos/keno4min/internal/tickets/ports"
	"github.com/cronos/keno4min/internal/tickets/usecase"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"tickets",
	fx.Provide(
		fx.Annotate(dynamodb.NewDynamodbRepository, fx.As(new(ports.Repository))),
		usecase.NewCreate,
		usecase.NewList,
		usecase.NewGet,
		http.NewCreateTicketHandler,
		http.NewListTicketsHandler,
		http.NewGetTicketHandler,
		fx.Annotate(
			http.NewRoutes,
			fx.As(new(serverfx.Route)),
			fx.ResultTags(`group:"routes"`),
		),
	),
)
