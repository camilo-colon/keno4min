package tickets

import (
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/cronos/keno4min/internal/tickets/events"
)

func TestPay(t *testing.T) {
	at := time.Date(2026, 11, 5, 15, 10, 0, 0, time.FixedZone("COT", -5*60*60))

	for _, tt := range []struct {
		name    string
		status  Status
		prize   int64
		paidBy  string
		at      time.Time
		wantErr error
		anyErr  bool
	}{
		{name: "winning ticket is paid", status: StatusWinning, prize: 175_000, paidBy: "cajero1", at: at},
		{name: "paid ticket", status: StatusPaid, prize: 175_000, paidBy: "cajero1", at: at, wantErr: ErrTicketAlreadyPaid},
		{name: "pending ticket", status: StatusPending, paidBy: "cajero1", at: at, wantErr: ErrTicketNotPayable},
		{name: "lost ticket", status: StatusLost, paidBy: "cajero1", at: at, wantErr: ErrTicketNotPayable},
		{name: "cancelled ticket", status: StatusCancelled, paidBy: "cajero1", at: at, wantErr: ErrTicketNotPayable},
		{name: "winning ticket without prize", status: StatusWinning, paidBy: "cajero1", at: at, wantErr: ErrTicketNotPayable},
		{name: "blank cashier", status: StatusWinning, prize: 175_000, paidBy: "  ", at: at, anyErr: true},
		{name: "zero time", status: StatusWinning, prize: 175_000, paidBy: "cajero1", anyErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ticket := &Ticket{ID: uuid.NewV7(), Status: tt.status, PrizeCents: tt.prize}

			err := ticket.Pay(tt.paidBy, tt.at)

			if tt.wantErr != nil || tt.anyErr {
				if err == nil || (tt.wantErr != nil && !errors.Is(err, tt.wantErr)) {
					t.Fatalf("Pay() error = %v, want %v", err, tt.wantErr)
				}
				if ticket.Status != tt.status || ticket.PaidBy != "" || !ticket.PaidAt.IsZero() {
					t.Fatalf("rejected payment changed the ticket: %+v", ticket)
				}
				if got := ticket.PullDomainEvents(); len(got) != 0 {
					t.Fatalf("rejected payment recorded events: %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Pay() unexpected error: %v", err)
			}
			if ticket.Status != StatusPaid || ticket.PaidBy != tt.paidBy || !ticket.PaidAt.Equal(at) || ticket.PaidAt.Location() != time.UTC {
				t.Fatalf("ticket not marked as paid in UTC: %+v", ticket)
			}

			got := ticket.PullDomainEvents()
			if len(got) != 1 {
				t.Fatalf("recorded %d events, want 1", len(got))
			}
			paid, ok := got[0].(events.TicketPaid)
			if !ok {
				t.Fatalf("event = %T, want events.TicketPaid", got[0])
			}
			if paid.TicketID != ticket.ID || paid.Game != Game || paid.PaidBy != tt.paidBy || paid.PrizeCents != tt.prize {
				t.Fatalf("unexpected event payload: %+v", paid)
			}
			if paid.Type() != events.TicketPaidType {
				t.Fatalf("event type = %q", paid.Type())
			}
		})
	}
}
