package outbox

import (
	"encoding/json"
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
		items, err := TransactItems(testTable, now, &testAggregate{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 0 {
			t.Fatalf("got %d items, want none", len(items))
		}
	})

	t.Run("writes one conditional outbox item per event", func(t *testing.T) {
		event := newTestEvent(now.Add(-time.Second))
		aggregate := &testAggregate{}
		aggregate.Record(event)

		items, err := TransactItems(testTable, now, aggregate)
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
		want := outboxItem{
			PK:            "OUTBOX#" + event.EventId.String(),
			SK:            "META",
			EntityType:    "OUTBOX",
			EventID:       event.EventId.String(),
			EventType:     testEventType,
			Source:        "cronos.keno4min",
			Detail:        got.Detail,
			OccurredOnUtc: "2026-11-02T15:04:04.000Z",
			TTL:           now.Add(7 * 24 * time.Hour).Unix(),
		}
		if got != want {
			t.Fatalf("got %+v, want %+v", got, want)
		}

		var detail testEvent
		if err := json.Unmarshal([]byte(got.Detail), &detail); err != nil {
			t.Fatalf("detail is not the event JSON: %v", err)
		}
		if detail.EventId != event.EventId || detail.AmountCents != event.AmountCents {
			t.Fatalf("detail = %+v, want %+v", detail, event)
		}
	})

	t.Run("pulls the events so they are written once", func(t *testing.T) {
		aggregate := &testAggregate{}
		aggregate.Record(newTestEvent(now))

		if _, err := TransactItems(testTable, now, aggregate); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		items, err := TransactItems(testTable, now, aggregate)
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
		_, err := TransactItems(testTable, now, aggregate)
		if !errors.Is(err, ErrTooManyEvents) {
			t.Fatalf("got %v, want ErrTooManyEvents", err)
		}
	})

	t.Run("rejects an event without id", func(t *testing.T) {
		aggregate := &testAggregate{}
		aggregate.Record(testEvent{OccurredOn: now})
		if _, err := TransactItems(testTable, now, aggregate); err == nil {
			t.Fatal("expected an error for an event without id")
		}
	})
}
