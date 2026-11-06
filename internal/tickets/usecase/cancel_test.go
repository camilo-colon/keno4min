package usecase

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	draw "github.com/cronos/keno4min/internal/draws"
	drawports "github.com/cronos/keno4min/internal/draws/ports"
	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
)

// Embedded interfaces are nil: calling a method the test does not stub panics.
type cancelRepository struct {
	ports.Repository
	ticket    *tickets.Ticket
	getErr    error
	updateErr error
	updated   []*tickets.Ticket
}

func (r *cancelRepository) GetByID(context.Context, uuid.UUID) (*tickets.Ticket, error) {
	return r.ticket, r.getErr
}

func (r *cancelRepository) Update(_ context.Context, ticket *tickets.Ticket) error {
	r.updated = append(r.updated, ticket)
	return r.updateErr
}

type cancelDraws struct {
	drawports.Repository
	current *draw.Draw
	err     error
}

func (d cancelDraws) GetCurrent(context.Context) (*draw.Draw, error) {
	return d.current, d.err
}

type cancelBalance struct {
	creditErr error
	credits   []ports.BalanceMovement
	debits    []ports.BalanceMovement
}

func (b *cancelBalance) Credit(_ context.Context, movement ports.BalanceMovement) (ports.BalanceResult, error) {
	b.credits = append(b.credits, movement)
	if b.creditErr != nil {
		return ports.BalanceResult{}, b.creditErr
	}
	return ports.BalanceResult{MovementID: "mv-1", BalanceAfterCents: 125_000}, nil
}

func (b *cancelBalance) Debit(_ context.Context, movement ports.BalanceMovement) (ports.BalanceResult, error) {
	b.debits = append(b.debits, movement)
	return ports.BalanceResult{}, nil
}

func TestCancel(t *testing.T) {
	now := time.Date(2026, 11, 5, 15, 1, 0, 0, time.UTC)
	nodeID := uuid.NewV7()
	failure := errors.New("boom")

	for _, tt := range []struct {
		name        string
		status      tickets.Status
		nodeID      uuid.UUID
		getErr      error
		drawTIR     uint64
		drawCloses  time.Time
		drawErr     error
		creditErr   error
		updateErr   error
		wantErr     error
		wantCredit  bool
		wantUpdate  bool
		wantDebit   bool
		wantBalance int64
	}{
		{name: "cancels and refunds", wantCredit: true, wantUpdate: true, wantBalance: 125_000},
		{name: "missing ticket", getErr: ports.ErrTicketNotFound, wantErr: ports.ErrTicketNotFound},
		{name: "ticket of another club", nodeID: uuid.NewV7(), wantErr: ports.ErrTicketNotFound},
		{name: "draw unavailable", drawErr: failure, wantErr: ErrDrawUnavailable},
		{name: "ticket of a previous draw", drawTIR: 6, wantErr: tickets.ErrBettingClosed},
		{name: "draw already closed", drawCloses: now, wantErr: tickets.ErrBettingClosed},
		{name: "already cancelled", status: tickets.StatusCancelled, wantErr: tickets.ErrTicketAlreadyCancelled},
		{name: "already settled", status: tickets.StatusLost, wantErr: tickets.ErrTicketNotCancellable},
		{name: "refund fails", creditErr: ports.ErrBalanceUnavailable, wantErr: ports.ErrBalanceUnavailable, wantCredit: true},
		{name: "update fails and refund is reversed", updateErr: failure, wantErr: failure, wantCredit: true, wantUpdate: true, wantDebit: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := tickets.Create(uuid.NewV7(), "0042", nodeID, 7, []tickets.Bet{
				{Nums: []uint{1, 2, 3}, ValueCents: 500},
				{Nums: []uint{80}, ValueCents: 1500},
			}, "seller")
			if err != nil {
				t.Fatalf("create ticket: %v", err)
			}
			ticket.PullDomainEvents()
			if tt.status != "" {
				ticket.Status = tt.status
			}
			repository := &cancelRepository{ticket: ticket, getErr: tt.getErr, updateErr: tt.updateErr}
			if tt.getErr != nil {
				repository.ticket = nil
			}
			current := &draw.Draw{
				TIR: 7, Status: draw.StatusOpen,
				StartsAt: now.Add(-time.Minute), ClosesAt: now.Add(time.Minute),
			}
			if tt.drawTIR != 0 {
				current.TIR = tt.drawTIR
			}
			if !tt.drawCloses.IsZero() {
				current.ClosesAt = tt.drawCloses
			}
			balance := &cancelBalance{creditErr: tt.creditErr}
			useCase := NewCancel(repository, cancelDraws{current: current, err: tt.drawErr}, balance)
			useCase.now = func() time.Time { return now }
			params := CancelParams{TicketID: ticket.ID, NodeID: nodeID, CancelledBy: "cashier"}
			if tt.nodeID != (uuid.UUID{}) {
				params.NodeID = tt.nodeID
			}

			result, err := useCase.Execute(context.Background(), params)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
				}
				if result != nil {
					t.Errorf("result = %+v, want nil", result)
				}
			} else if err != nil {
				t.Fatalf("Execute() error = %v", err)
			} else if result.BalanceCents != tt.wantBalance {
				t.Errorf("BalanceCents = %d, want %d", result.BalanceCents, tt.wantBalance)
			}

			if got := len(balance.credits) == 1; got != tt.wantCredit {
				t.Fatalf("credits = %d, want credit %v", len(balance.credits), tt.wantCredit)
			}
			if tt.wantCredit {
				want := ports.BalanceMovement{
					NodeID: nodeID, AmountCents: 2000, Reason: ports.ReasonTicketCancellation,
					Reference: ticket.ID.String(), IdempotencyKey: "ticket:" + ticket.ID.String() + ":cancel-credit",
				}
				if balance.credits[0] != want {
					t.Errorf("credit = %+v, want %+v", balance.credits[0], want)
				}
			}
			if got := len(repository.updated) == 1; got != tt.wantUpdate {
				t.Fatalf("updates = %d, want update %v", len(repository.updated), tt.wantUpdate)
			}
			if tt.wantUpdate {
				updated := repository.updated[0]
				if updated.Status != tickets.StatusCancelled || updated.PaidBy != "cashier" || !updated.PaidAt.Equal(now) {
					t.Errorf("updated ticket = status %q, paid by %q at %v", updated.Status, updated.PaidBy, updated.PaidAt)
				}
				if events := updated.PullDomainEvents(); len(events) != 1 {
					t.Errorf("updated ticket carries %d events, want 1", len(events))
				}
			}
			if got := len(balance.debits) == 1; got != tt.wantDebit {
				t.Fatalf("debits = %d, want debit %v", len(balance.debits), tt.wantDebit)
			}
			if tt.wantDebit {
				debit := balance.debits[0]
				if debit.Reason != ports.ReasonTicketCancellationReversal || debit.AmountCents != 2000 ||
					debit.IdempotencyKey != "ticket:"+ticket.ID.String()+":cancel-credit-reversal" {
					t.Errorf("debit = %+v", debit)
				}
			}
		})
	}
}
