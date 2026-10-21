package common

import "context"

type DomainEventSubscriber interface {
	On(context.Context, DomainEvent) error
	// SubscribeTo returns struct values used as type prototypes, not occurred events.
	// Type returns an exact name or a pattern with a single trailing '*'.
	SubscribeTo() []DomainEvent
	Name() string
}
