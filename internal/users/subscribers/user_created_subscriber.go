package subscribers

import (
	"context"
	"errors"

	"github.com/cronos/keno4min/internal/common"
	"github.com/cronos/keno4min/internal/users/events"
	"github.com/cronos/keno4min/internal/users/usecase"
)

var _ common.DomainEventSubscriber = (*UserCreatedSubscriber)(nil)

type UserCreatedSubscriber struct {
	create *usecase.Create
}

func NewUserCreatedSubscriber(create *usecase.Create) *UserCreatedSubscriber {
	return &UserCreatedSubscriber{create: create}
}

func (s UserCreatedSubscriber) On(ctx context.Context, domainEvent common.DomainEvent) error {
	event, ok := domainEvent.(events.UserCreatedDomainEvent)
	if !ok {
		return errors.New("domain event isn't UserCreatedDomainEvent")
	}
	return s.create.Execute(ctx, usecase.CreateParams{
		ID:       event.UserID,
		Username: event.Username,
		RoleId:   event.RoleId,
		NodeId:   event.NodeId,
	})
}

func (s UserCreatedSubscriber) SubscribeTo() []common.DomainEvent {
	return []common.DomainEvent{events.UserCreatedDomainEvent{}}
}

func (s UserCreatedSubscriber) Name() string {
	return "cronos.keno4min.user.create_on_user_created"
}
