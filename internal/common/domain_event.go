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

func (b BaseEvent) Id() uuid.UUID {
	return b.EventId
}

func (b BaseEvent) OccurredOnUtc() time.Time {
	return b.OccurredOn
}
