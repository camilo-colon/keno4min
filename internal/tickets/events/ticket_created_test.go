package events

import (
	"encoding/json"
	"testing"
	"time"
	"uuid"
)

// The JSON is the detail that consumers of cronos.keno4min.ticket.created receive.
func TestTicketCreatedJSON(t *testing.T) {
	eventID := uuid.MustParse("0190a3b4-0000-7000-8000-000000000001")
	ticketID := uuid.MustParse("0190a3b4-0000-7000-8000-000000000002")
	nodeID := uuid.MustParse("0190a3b4-0000-7000-8000-000000000003")
	event := TicketCreated{
		EventId:    eventID,
		OccurredOn: time.Date(2026, 11, 2, 15, 4, 5, 123000000, time.UTC),
		TicketID:   ticketID,
		Coupon:     "0001",
		NodeID:     nodeID,
		TIR:        42,
		Bets:       []Bet{{Nums: []uint{1, 80}, ValueCents: 1000}},
		TotalCents: 1000,
		CreatedBy:  "seller-1",
	}

	got, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"eventId":"` + eventID.String() + `","occurredOnUtc":"2026-11-02T15:04:05.123Z",` +
		`"ticketId":"` + ticketID.String() + `","coupon":"0001","nodeId":"` + nodeID.String() + `","tir":42,` +
		`"bets":[{"nums":[1,80],"valueCents":1000}],"totalCents":1000,"createdBy":"seller-1"}`
	if string(got) != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}
