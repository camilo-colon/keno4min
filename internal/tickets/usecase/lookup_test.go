package usecase

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"uuid"

	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
)

type lookupRepository struct {
	ports.Repository
	tickets []*tickets.Ticket
	asked   []uuid.UUID
}

func (r *lookupRepository) GetByIDs(_ context.Context, ids []uuid.UUID) ([]*tickets.Ticket, error) {
	r.asked = ids
	return r.tickets, nil
}

func TestLookup(t *testing.T) {
	node, otherNode := uuid.NewV7(), uuid.NewV7()
	own := &tickets.Ticket{ID: uuid.NewV7(), NodeID: node}
	foreign := &tickets.Ticket{ID: uuid.NewV7(), NodeID: otherNode}

	t.Run("omits tickets of another club and asks each id once", func(t *testing.T) {
		repository := &lookupRepository{tickets: []*tickets.Ticket{own, foreign}}
		got, err := NewLookup(repository).Execute(context.Background(), LookupParams{
			TicketIDs: []uuid.UUID{own.ID, foreign.ID, own.ID},
			NodeID:    node,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, []*tickets.Ticket{own}) {
			t.Fatalf("got %v, want only the ticket of the club", got)
		}
		if !reflect.DeepEqual(repository.asked, []uuid.UUID{own.ID, foreign.ID}) {
			t.Fatalf("asked %v, want ids without duplicates", repository.asked)
		}
	})

	for _, tt := range []struct {
		name  string
		count int
	}{
		{name: "rejects an empty list", count: 0},
		{name: "rejects more than the limit", count: maxLookupIDs + 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ids := make([]uuid.UUID, tt.count)
			for i := range ids {
				ids[i] = uuid.NewV7()
			}
			_, err := NewLookup(&lookupRepository{}).Execute(context.Background(), LookupParams{TicketIDs: ids, NodeID: node})
			if !errors.Is(err, ErrInvalidLookupParams) {
				t.Fatalf("got %v, want %v", err, ErrInvalidLookupParams)
			}
		})
	}
}
