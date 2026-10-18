package subscribers

import (
	"context"
	"errors"

	"github.com/cronos/keno4min/internal/common"
	"github.com/cronos/keno4min/internal/nodes"
	"github.com/cronos/keno4min/internal/nodes/events"
	"github.com/cronos/keno4min/internal/nodes/ports"
)

var _ common.DomainEventSubscriber = (*NodeCreatedSubscriber)(nil)

type NodeCreatedSubscriber struct {
	repositoy ports.Repository
}

func NewNodeCretedSubscriber(repository ports.Repository) *NodeCreatedSubscriber {
	return &NodeCreatedSubscriber{
		repositoy: repository,
	}
}

func (h NodeCreatedSubscriber) On(context context.Context, domainEvent common.DomainEvent) error {
	event, ok := domainEvent.(events.NodeCreatedDomainEvent)
	if !ok {
		return errors.New("domain event isn't NodeCreatedDomainEvent")
	}
	node := nodes.New(event.EventId, event.ParentId, event.NodeType, event.Name, event.Address, event.Currency, event.BalanceCents, event.Active)
	if err := h.repositoy.Save(context, node); err != nil {
		return err
	}
	return nil
}

func (n NodeCreatedSubscriber) SubscribeTo() []string {
	return []string{events.NodeCreatedDomainEventType}
}

func (n NodeCreatedSubscriber) Name() string {
	return "cronos.keno4min.node.created"
}
