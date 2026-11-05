package usecase

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
	"uuid"

	draw "github.com/cronos/keno4min/internal/draws"
	drawports "github.com/cronos/keno4min/internal/draws/ports"
	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
)

type repeatRepository struct {
	ports.Repository
	stored *tickets.Ticket
	saved  *tickets.Ticket
}

func (r *repeatRepository) GetByID(_ context.Context, id uuid.UUID) (*tickets.Ticket, error) {
	if r.stored == nil || r.stored.ID != id {
		return nil, fmt.Errorf("%w: %s", ports.ErrTicketNotFound, id)
	}
	return r.stored, nil
}

func (r *repeatRepository) GenerateCoupon(context.Context) (string, error) {
	return "NEWCOUPON0000001", nil
}

func (r *repeatRepository) Save(_ context.Context, ticket *tickets.Ticket) error {
	r.saved = ticket
	return nil
}

type repeatDraws struct {
	drawports.Repository
	current *draw.Draw
}

func (d *repeatDraws) GetCurrent(context.Context) (*draw.Draw, error) {
	return d.current, nil
}

func TestRepeat(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	node := uuid.NewV7()
	original := &tickets.Ticket{
		ID:        uuid.NewV7(),
		Coupon:    "OLDCOUPON0000001",
		NodeID:    node,
		TIR:       41,
		Status:    tickets.StatusCancelled,
		CreatedBy: "seller",
		Bets: []tickets.Bet{
			{Nums: []uint{1, 2, 3}, ValueCents: 50000},
			{Nums: []uint{10, 20, 30}, ValueCents: 25000},
		},
	}
	openDraw := &draw.Draw{ID: uuid.NewV7(), TIR: 42, Status: draw.StatusOpen, StartsAt: now.Add(-time.Minute), ClosesAt: now.Add(time.Minute)}
	closedDraw := &draw.Draw{ID: uuid.NewV7(), TIR: 42, Status: draw.StatusClosed, StartsAt: now.Add(-time.Minute), ClosesAt: now.Add(time.Minute)}

	newRepeat := func(repository *repeatRepository, current *draw.Draw) *Repeat {
		create := NewCreate(repository, &repeatDraws{current: current})
		create.now = func() time.Time { return now }
		return NewRepeat(repository, create)
	}

	t.Run("creates a new ticket in the current draw with the same bets", func(t *testing.T) {
		repository := &repeatRepository{stored: original}
		got, err := newRepeat(repository, openDraw).Execute(context.Background(), RepeatParams{
			TicketID: original.ID, NodeID: node, CreatedBy: "cashier",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != repository.saved {
			t.Fatal("the repeated ticket was not saved")
		}
		if got.ID == original.ID || got.Coupon == original.Coupon {
			t.Fatalf("got id %s and coupon %s, want new ones", got.ID, got.Coupon)
		}
		if got.TIR != openDraw.TIR {
			t.Fatalf("got TIR %d, want current draw %d", got.TIR, openDraw.TIR)
		}
		if got.CreatedBy != "cashier" || got.NodeID != node || got.Status != tickets.StatusPending {
			t.Fatalf("got createdBy %q, node %s, status %s", got.CreatedBy, got.NodeID, got.Status)
		}
		if !reflect.DeepEqual(got.Bets, original.Bets) {
			t.Fatalf("got bets %v, want %v", got.Bets, original.Bets)
		}
		got.Bets[0].Nums[0] = 80
		if original.Bets[0].Nums[0] != 1 {
			t.Fatal("the repeated ticket shares bet numbers with the original")
		}
	})

	for _, tt := range []struct {
		name    string
		stored  *tickets.Ticket
		current *draw.Draw
		want    error
	}{
		{name: "reports a missing ticket", stored: nil, current: openDraw, want: ports.ErrTicketNotFound},
		{name: "hides a ticket of another club", stored: &tickets.Ticket{ID: original.ID, NodeID: uuid.NewV7(), Bets: original.Bets}, current: openDraw, want: ports.ErrTicketNotFound},
		{name: "rejects when the current draw is closed", stored: original, current: closedDraw, want: tickets.ErrBettingClosed},
		{name: "rejects when there is no current draw", stored: original, current: nil, want: ErrDrawUnavailable},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repository := &repeatRepository{stored: tt.stored}
			_, err := newRepeat(repository, tt.current).Execute(context.Background(), RepeatParams{
				TicketID: original.ID, NodeID: node, CreatedBy: "cashier",
			})
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
			if repository.saved != nil {
				t.Fatal("no ticket should be saved")
			}
		})
	}
}
