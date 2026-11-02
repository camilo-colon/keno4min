package outbox

import (
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/cronos/keno4min/internal/common"
)

const (
	testTable     = "keno-test"
	testEventType = "cronos.keno4min.test.happened"
)

type testEvent struct {
	common.BaseEvent
	AmountCents int64 `json:"amountCents"`
}

func (testEvent) Type() string {
	return testEventType
}

type testAggregate struct {
	common.AggregateRoot
}

func newTestEvent(occurredOn time.Time) testEvent {
	return testEvent{EventId: uuid.NewV7(), OccurredOn: occurredOn, AmountCents: 1500}
}

func TestTransactItems(t *testing.T) {
	now := time.Date(2026, 11, 2, 15, 4, 5, 0, time.UTC)

	t.Run("returns no items when the aggregate has no events", func(t *testing.T) {
		items, err := TransactItems(testTable, &testAggregate{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 0 {
			t.Fatalf("got %d items, want none", len(items))
		}
	})

	t.Run("writes one conditional outbox item per event", func(t *testing.T) {
		event := newTestEvent(now.Add(-877 * time.Millisecond))
		aggregate := &testAggregate{}
		aggregate.Record(event)

		items, err := TransactItems(testTable, aggregate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 1 || items[0].Put == nil {
			t.Fatalf("got %v, want one put", items)
		}
		put := items[0].Put
		if aws.ToString(put.TableName) != testTable {
			t.Fatalf("table = %q, want keno-test", aws.ToString(put.TableName))
		}
		if aws.ToString(put.ConditionExpression) != "attribute_not_exists(PK)" {
			t.Fatalf("condition = %q, want attribute_not_exists(PK)", aws.ToString(put.ConditionExpression))
		}

		var got outboxItem
		if err := attributevalue.UnmarshalMap(put.Item, &got); err != nil {
			t.Fatalf("unmarshal item: %v", err)
		}
		// detail is the contract with the Pipe and consumers, so its keys are fixed here.
		want := outboxItem{
			PK:            "OUTBOX#" + event.EventId.String(),
			SK:            "META",
			EntityType:    "OUTBOX",
			EventID:       event.EventId.String(),
			EventType:     testEventType,
			Source:        "cronos.keno4min",
			Detail:        `{"eventId":"` + event.EventId.String() + `","occurredOnUtc":"2026-11-02T15:04:04.123Z","amountCents":1500}`,
			OccurredOnUtc: "2026-11-02T15:04:04.123Z",
			TTL:           time.Date(2026, 11, 9, 15, 4, 4, 123000000, time.UTC).Unix(),
		}
		if got != want {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})

	t.Run("pulls the events so they are written once", func(t *testing.T) {
		aggregate := &testAggregate{}
		aggregate.Record(newTestEvent(now))

		if _, err := TransactItems(testTable, aggregate); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		items, err := TransactItems(testTable, aggregate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 0 {
			t.Fatalf("got %d items on the second call, want none", len(items))
		}
	})

	t.Run("rejects more events than a transaction accepts", func(t *testing.T) {
		aggregate := &testAggregate{}
		for range maxTransactItems + 1 {
			aggregate.Record(newTestEvent(now))
		}
		_, err := TransactItems(testTable, aggregate)
		if !errors.Is(err, ErrTooManyEvents) {
			t.Fatalf("got %v, want ErrTooManyEvents", err)
		}
	})

	t.Run("rejects an event without occurrence date", func(t *testing.T) {
		aggregate := &testAggregate{}
		aggregate.Record(testEvent{EventId: uuid.NewV7()})
		if _, err := TransactItems(testTable, aggregate); err == nil {
			t.Fatal("expected an error for an event without occurrence date")
		}
	})

	t.Run("rejects an event without id", func(t *testing.T) {
		aggregate := &testAggregate{}
		aggregate.Record(testEvent{OccurredOn: now})
		if _, err := TransactItems(testTable, aggregate); err == nil {
			t.Fatal("expected an error for an event without id")
		}
	})
}
