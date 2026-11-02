package tickets

import (
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/cronos/keno4min/internal/tickets/events"
)

func TestCreateRecordsTicketCreated(t *testing.T) {
	id, nodeID := uuid.NewV7(), uuid.NewV7()
	bets := []Bet{
		{Nums: []uint{1, 80}, ValueCents: 1000},
		{Nums: []uint{7}, ValueCents: 500},
	}

	ticket, err := Create(id, "0001", nodeID, 42, bets, "seller-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recorded := ticket.PullDomainEvents()
	if len(recorded) != 1 {
		t.Fatalf("got %d events, want one", len(recorded))
	}
	event, ok := recorded[0].(events.TicketCreated)
	if !ok {
		t.Fatalf("got %T, want TicketCreated", recorded[0])
	}
	if event.EventId == uuid.Nil() {
		t.Fatal("event id is nil")
	}
	if !event.OccurredOn.Equal(ticket.CreatedAt.Truncate(time.Millisecond)) || event.OccurredOn.Location() != time.UTC {
		t.Fatalf("occurred on = %v, want the creation date %v in UTC", event.OccurredOn, ticket.CreatedAt)
	}

	want := events.TicketCreated{
		BaseEvent:  event.BaseEvent,
		TicketID:   id,
		Coupon:     "0001",
		NodeID:     nodeID,
		TIR:        42,
		Bets:       []events.Bet{{Nums: []uint{1, 80}, ValueCents: 1000}, {Nums: []uint{7}, ValueCents: 500}},
		TotalCents: 1500,
		CreatedBy:  "seller-1",
	}
	if !reflect.DeepEqual(event, want) {
		t.Fatalf("got %+v, want %+v", event, want)
	}
}
