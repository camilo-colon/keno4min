package common

import (
	"testing"
	"time"
	"uuid"
)

func TestNewBaseEvent(t *testing.T) {
	bogota := time.FixedZone("COT", -5*60*60)
	occurredOn := time.Date(2026, 11, 2, 10, 4, 5, 123456789, bogota)

	event := NewBaseEvent(occurredOn)

	if event.EventId == uuid.Nil() {
		t.Fatal("event id is nil")
	}
	want := time.Date(2026, 11, 2, 15, 4, 5, 123000000, time.UTC)
	if !event.OccurredOn.Equal(want) || event.OccurredOn.Location() != time.UTC || event.OccurredOn.Nanosecond() != want.Nanosecond() {
		t.Fatalf("occurred on = %v, want %v in UTC", event.OccurredOn, want)
	}
}
