package subscribers

import (
	"context"
	"errors"

	"github.com/cronos/keno4min/internal/common"
	"github.com/cronos/keno4min/internal/nodes/events"
	"github.com/cronos/keno4min/internal/nodes/usecase"
)

var _ common.DomainEventSubscriber = (*NodeCreatedSubscriber)(nil)

type NodeCreatedSubscriber struct {
	create *usecase.Create
}

func NewNodeCretedSubscriber(create *usecase.Create) *NodeCreatedSubscriber {
	return &NodeCreatedSubscriber{
		create: create,
	}
}

func (h NodeCreatedSubscriber) On(ctx context.Context, domainEvent common.DomainEvent) error {
	event, ok := domainEvent.(events.NodeCreatedDomainEvent)
	if !ok {
		return errors.New("domain event isn't NodeCreatedDomainEvent")
	}
	return h.create.Execute(ctx, usecase.CreateParams{
		ID:           event.NodeId,
		ParentId:     event.ParentId,
		Type:         event.NodeType,
		Name:         event.Name,
		Address:      event.Address,
		TimeZone:     event.TimeZone,
		Currency:     event.Currency,
		BalanceCents: event.BalanceCents,
		Active:       event.Active,
	})
}

func (n NodeCreatedSubscriber) SubscribeTo() []common.DomainEvent {
	return []common.DomainEvent{events.NodeCreatedDomainEvent{}}
}

func (n NodeCreatedSubscriber) Name() string {
	return "cronos.keno4min.node.create_on_node_created"
}
