package common

type Aggregate interface {
	PullDomainEvents() []DomainEvent
}

var _ Aggregate = (*AggregateRoot)(nil)

type AggregateRoot struct {
	domainEvents []DomainEvent
}

func (a *AggregateRoot) Record(event DomainEvent) {
	a.domainEvents = append(a.domainEvents, event)
}

func (a *AggregateRoot) PullDomainEvents() []DomainEvent {
	events := a.domainEvents
	a.domainEvents = nil
	return events
}
