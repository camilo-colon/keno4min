package outbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"uuid"

	"github.com/cronos/keno4min/internal/common"
)

const entityType = "OUTBOX"

// occurredOnLayout fixes the millisecond width, as the other items of the table do.
const occurredOnLayout = "2006-01-02T15:04:05.000Z07:00"

type outboxItem struct {
	PK            string `dynamodbav:"PK"`
	SK            string `dynamodbav:"SK"`
	EntityType    string `dynamodbav:"entityType"`
	EventID       string `dynamodbav:"eventId"`
	EventType     string `dynamodbav:"eventType"`
	Source        string `dynamodbav:"source"`
	Detail        string `dynamodbav:"detail"`
	OccurredOnUtc string `dynamodbav:"occurredOnUtc"`
	TTL           int64  `dynamodbav:"ttl"`
}

func toItem(event common.DomainEvent) (outboxItem, error) {
	if event.Id() == uuid.Nil() {
		return outboxItem{}, errors.New("event id is required")
	}
	// A zero date would set a ttl in 1970 and DynamoDB could delete the item right away.
	if event.OccurredOnUtc().IsZero() {
		return outboxItem{}, fmt.Errorf("event %s must have an occurrence date", event.Id())
	}
	detail, err := json.Marshal(event)
	if err != nil {
		return outboxItem{}, fmt.Errorf("marshal event %s: %w", event.Id(), err)
	}
	return outboxItem{
		PK:            "OUTBOX#" + event.Id().String(),
		SK:            "META",
		EntityType:    entityType,
		EventID:       event.Id().String(),
		EventType:     event.Type(),
		Source:        source,
		Detail:        string(detail),
		OccurredOnUtc: event.OccurredOnUtc().UTC().Format(occurredOnLayout),
		TTL:           event.OccurredOnUtc().Add(retention).Unix(),
	}, nil
}
