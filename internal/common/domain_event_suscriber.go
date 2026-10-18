package common

import "context"



type DomainEventSubscriber interface {
	On(context.Context, DomainEvent) error
	SubscribeTo() []string
	Name() string
}
