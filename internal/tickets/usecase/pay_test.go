package usecase

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
)

// Pay reuses the cancelRepository and cancelBalance doubles from cancel_test.go.
func TestPay(t *testing.T) {
	now := time.Date(2026, 11, 5, 15, 10, 0, 0, time.UTC)
	nodeID := uuid.NewV7()
	failure := errors.New("boom")

	for _, tt := range []struct {
		name        string
		status      tickets.Status
		nodeID      uuid.UUID
		getErr      error
		creditErr   error
		updateErr   error
		wantErr     error
		wantCredit  bool
		wantUpdate  bool
		wantDebit   bool
		wantBalance int64
	}{
		{name: "pays and credits the prize", wantCredit: true, wantUpdate: true, wantBalance: 125_000},
		{name: "missing ticket", getErr: ports.ErrTicketNotFound, wantErr: ports.ErrTicketNotFound},
		{name: "ticket of another club", nodeID: uuid.NewV7(), wantErr: ports.ErrTicketNotFound},
		{name: "already paid", status: tickets.StatusPaid, wantErr: tickets.ErrTicketAlreadyPaid},
		{name: "lost ticket", status: tickets.StatusLost, wantErr: tickets.ErrTicketNotPayable},
		{name: "pending ticket", status: tickets.StatusPending, wantErr: tickets.ErrTicketNotPayable},
		{name: "cancelled ticket", status: tickets.StatusCancelled, wantErr: tickets.ErrTicketNotPayable},
		{name: "credit fails", creditErr: ports.ErrBalanceUnavailable, wantErr: ports.ErrBalanceUnavailable, wantCredit: true},
		{name: "update fails and credit is reversed", updateErr: failure, wantErr: failure, wantCredit: true, wantUpdate: true, wantDebit: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := tickets.Create(uuid.NewV7(), "0042", nodeID, 7, []tickets.Bet{
				{Nums: []uint{1, 2, 3}, ValueCents: 500},
			}, "seller")
			if err != nil {
				t.Fatalf("create ticket: %v", err)
			}
			ticket.PullDomainEvents()
			ticket.Status = tickets.StatusWinning
			ticket.PrizeCents = 17_500
			if tt.status != "" {
				ticket.Status = tt.status
			}
			repository := &cancelRepository{ticket: ticket, getErr: tt.getErr, updateErr: tt.updateErr}
			if tt.getErr != nil {
				repository.ticket = nil
			}
			balance := &cancelBalance{creditErr: tt.creditErr}
			useCase := NewPay(repository, balance)
			useCase.now = func() time.Time { return now }
			params := PayParams{TicketID: ticket.ID, NodeID: nodeID, PaidBy: "cashier"}
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
			} else if result.BalanceCents != tt.wantBalance || result.PrizeCents != 17_500 {
				t.Errorf("result = %+v, want balance %d and prize 17500", result, tt.wantBalance)
			}

			if got := len(balance.credits) == 1; got != tt.wantCredit {
				t.Fatalf("credits = %d, want credit %v", len(balance.credits), tt.wantCredit)
			}
			if tt.wantCredit {
				want := ports.BalanceMovement{
					NodeID: nodeID, AmountCents: 17_500, Reason: ports.ReasonTicketPayment,
					Reference: ticket.ID.String(), IdempotencyKey: "ticket:" + ticket.ID.String() + ":pay-credit",
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
				if updated.Status != tickets.StatusPaid || updated.PaidBy != "cashier" || !updated.PaidAt.Equal(now) {
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
				if debit.Reason != ports.ReasonTicketPaymentReversal || debit.AmountCents != 17_500 ||
					debit.IdempotencyKey != "ticket:"+ticket.ID.String()+":pay-credit-reversal" {
					t.Errorf("debit = %+v", debit)
				}
			}
		})
	}
}
