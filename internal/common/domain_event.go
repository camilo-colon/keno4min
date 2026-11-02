package common

import (
	"time"
	"uuid"
)

type DomainEvent interface {
	Id() uuid.UUID
	Type() string
	OccurredOnUtc() time.Time
}

type BaseEvent struct {
	EventId    uuid.UUID `json:"eventId"`
	OccurredOn time.Time `json:"occurredOnUtc"`
}

// NewBaseEvent keeps occurredOnUtc in UTC with millisecond precision, as the outbox persists it.
func NewBaseEvent(occurredOn time.Time) BaseEvent {
	return BaseEvent{EventId: uuid.NewV7(), OccurredOn: occurredOn.UTC().Truncate(time.Millisecond)}
}

func (b BaseEvent) Id() uuid.UUID {
	return b.EventId
}

func (b BaseEvent) OccurredOnUtc() time.Time {
	return b.OccurredOn
}
