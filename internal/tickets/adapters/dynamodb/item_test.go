package dynamodb

import (
	"testing"
	"time"
	"uuid"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/cronos/keno4min/internal/tickets"
)

func TestTicketItemKeepsAdminID(t *testing.T) {
	ticket := &tickets.Ticket{
		ID:        uuid.NewV7(),
		NodeID:    uuid.NewV7(),
		AdminID:   uuid.NewV7(),
		CreatedAt: time.Now().UTC(),
	}

	restored, err := fromItem(toItem(ticket))
	if err != nil {
		t.Fatal(err)
	}
	if restored.AdminID != ticket.AdminID {
		t.Fatalf("admin id = %s, want %s", restored.AdminID, ticket.AdminID)
	}
}

func TestTicketItemWithoutAdminIDOmitsTheAttribute(t *testing.T) {
	ticket := &tickets.Ticket{
		ID:        uuid.NewV7(),
		NodeID:    uuid.NewV7(),
		CreatedAt: time.Now().UTC(),
	}

	attributes, err := attributevalue.MarshalMap(toItem(ticket))
	if err != nil {
		t.Fatal(err)
	}
	if _, found := attributes["adminId"]; found {
		t.Fatal("adminId must be absent when the ticket has no admin")
	}

	restored, err := fromItem(toItem(ticket))
	if err != nil {
		t.Fatal(err)
	}
	if restored.AdminID != uuid.Nil() {
		t.Fatalf("admin id = %s, want the nil uuid", restored.AdminID)
	}
}
