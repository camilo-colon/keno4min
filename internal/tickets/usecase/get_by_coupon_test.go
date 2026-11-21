package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"uuid"

	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
)

// couponRepository implements only the reads used by GetByCoupon.
type couponRepository struct {
	ports.Repository
	byCoupon map[string]*tickets.Ticket
}

func (r couponRepository) GetByCoupon(_ context.Context, coupon string) (*tickets.Ticket, error) {
	ticket, ok := r.byCoupon[coupon]
	if !ok {
		return nil, fmt.Errorf("%w: coupon %s", ports.ErrTicketNotFound, coupon)
	}
	return ticket, nil
}

func TestGetByCouponReturnsTicketOfTheSameClub(t *testing.T) {
	ticket := newCouponTicket(t, "0001")
	useCase := NewGetByCoupon(couponRepository{byCoupon: map[string]*tickets.Ticket{"0001": ticket}})

	got, err := useCase.Execute(t.Context(), GetByCouponParams{Coupon: "0001", NodeID: ticket.NodeID})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != ticket.ID {
		t.Fatalf("ticket id = %s, want %s", got.ID, ticket.ID)
	}
}

func TestGetByCouponOfMissingCouponIsNotFound(t *testing.T) {
	useCase := NewGetByCoupon(couponRepository{})

	_, err := useCase.Execute(t.Context(), GetByCouponParams{Coupon: "9999", NodeID: uuid.NewV7()})
	if !errors.Is(err, ports.ErrTicketNotFound) {
		t.Fatalf("error = %v, want ErrTicketNotFound", err)
	}
}

func TestGetByCouponOfAnotherClubIsNotFound(t *testing.T) {
	ticket := newCouponTicket(t, "0001")
	useCase := NewGetByCoupon(couponRepository{byCoupon: map[string]*tickets.Ticket{"0001": ticket}})

	_, err := useCase.Execute(t.Context(), GetByCouponParams{Coupon: "0001", NodeID: uuid.NewV7()})
	if !errors.Is(err, ports.ErrTicketNotFound) {
		t.Fatalf("error = %v, want ErrTicketNotFound", err)
	}
}

func newCouponTicket(t *testing.T, coupon string) *tickets.Ticket {
	bets := []tickets.Bet{{Nums: []uint{1, 2, 3}, ValueCents: 500}}
	ticket, err := tickets.Create(uuid.NewV7(), coupon, uuid.NewV7(), uuid.NewV7(), 4321, bets, "cashier")
	if err != nil {
		t.Fatal(err)
	}
	return ticket
}
