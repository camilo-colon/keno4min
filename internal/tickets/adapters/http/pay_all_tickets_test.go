package http

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"

	"github.com/cronos/keno4min/internal/platform/auth"
	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
	"github.com/cronos/keno4min/internal/tickets/usecase"
	"github.com/gofiber/fiber/v3"
)

// Embedded interface is nil: calling a method the test does not stub panics.
type payAllRepository struct {
	ports.Repository
	items   []*tickets.Ticket
	listErr error
}

func (r *payAllRepository) ListByNode(context.Context, ports.ListParams) (ports.TicketPage, error) {
	return ports.TicketPage{Items: r.items}, r.listErr
}

func (r *payAllRepository) Update(context.Context, *tickets.Ticket) error {
	return nil
}

// payAllBalance fails the credit of the tickets in failFor.
type payAllBalance struct {
	failFor map[string]error
}

func (b payAllBalance) Credit(_ context.Context, movement ports.BalanceMovement) (ports.BalanceResult, error) {
	if err := b.failFor[movement.Reference]; err != nil {
		return ports.BalanceResult{}, err
	}
	return ports.BalanceResult{MovementID: "mv-1", BalanceAfterCents: 125_000}, nil
}

func (b payAllBalance) Debit(context.Context, ports.BalanceMovement) (ports.BalanceResult, error) {
	return ports.BalanceResult{}, nil
}

func TestPayAllTicketsHandler(t *testing.T) {
	nodeID := uuid.NewV7()
	first := uuid.MustParse("019a5b6e-0000-7000-8000-000000000001")
	second := uuid.MustParse("019a5b6e-0000-7000-8000-000000000002")
	winners := func() []*tickets.Ticket {
		return []*tickets.Ticket{
			{ID: first, Coupon: "0001", NodeID: nodeID, Status: tickets.StatusWinning, PrizeCents: 10_000},
			{ID: second, Coupon: "0002", NodeID: nodeID, Status: tickets.StatusWinning, PrizeCents: 5_000},
		}
	}

	for _, tt := range []struct {
		name       string
		noToken    bool
		body       string
		items      []*tickets.Ticket
		listErr    error
		failFor    map[string]error
		wantStatus int
		wantBody   string
	}{
		{
			name: "pays every winner", items: winners(), wantStatus: fiber.StatusOK,
			wantBody: `{"paidCount":2,"totalPrizeCents":15000,"balanceCents":125000,"paid":[` +
				`{"id":"` + first.String() + `","coupon":"0001","prizeCents":10000},` +
				`{"id":"` + second.String() + `","coupon":"0002","prizeCents":5000}],"failed":[]}`,
		},
		{
			name: "partial payment", items: winners(), failFor: map[string]error{first.String(): ports.ErrBalanceUnavailable},
			wantStatus: fiber.StatusOK,
			wantBody: `{"paidCount":1,"totalPrizeCents":5000,"balanceCents":125000,"paid":[` +
				`{"id":"` + second.String() + `","coupon":"0002","prizeCents":5000}],"failed":[` +
				`{"id":"` + first.String() + `","coupon":"0001","reason":"balance_unavailable"}]}`,
		},
		{
			name: "nothing paid because core is down", items: winners(),
			failFor:    map[string]error{first.String(): ports.ErrBalanceUnavailable, second.String(): errors.New("boom")},
			wantStatus: fiber.StatusServiceUnavailable,
			wantBody: `{"paidCount":0,"totalPrizeCents":0,"paid":[],"failed":[` +
				`{"id":"` + first.String() + `","coupon":"0001","reason":"balance_unavailable"},` +
				`{"id":"` + second.String() + `","coupon":"0002","reason":"internal_error"}]}`,
		},
		{
			name: "nothing paid for other reasons", items: winners()[:1],
			failFor:    map[string]error{first.String(): errors.New("boom")},
			wantStatus: fiber.StatusOK,
		},
		{
			name: "nothing to pay", wantStatus: fiber.StatusOK,
			wantBody: `{"paidCount":0,"totalPrizeCents":0,"paid":[],"failed":[]}`,
		},
		{name: "listing fails", listErr: errors.New("boom"), wantStatus: fiber.StatusInternalServerError},
		{name: "missing token", noToken: true, wantStatus: fiber.StatusUnauthorized},
		{name: "malformed body", body: `{"from":"yesterday"}`, wantStatus: fiber.StatusBadRequest},
		{
			name: "from after to", body: `{"from":"2026-11-06T00:00:00Z","to":"2026-11-01T00:00:00Z"}`,
			wantStatus: fiber.StatusBadRequest, wantBody: `{"error":"invalid pay all filters: from must be before to"}`,
		},
		{name: "missing dates", body: `{}`, wantStatus: fiber.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repository := &payAllRepository{items: tt.items, listErr: tt.listErr}
			pay := usecase.NewPay(repository, payAllBalance{failFor: tt.failFor})
			handler := NewPayAllTicketsHandler(usecase.NewPayAll(repository, pay))
			app := fiber.New()
			app.Use(auth.New())
			app.Post("/tickets/pay", handler.Handle)

			body := tt.body
			if body == "" {
				body = `{"from":"2026-11-01T00:00:00Z","to":"2026-11-06T00:00:00Z"}`
			}
			req := httptest.NewRequest(fiber.MethodPost, "/tickets/pay", strings.NewReader(body))
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			if !tt.noToken {
				req.Header.Set(fiber.HeaderAuthorization, "Bearer "+testToken(t, nodeID))
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			if tt.wantBody != "" {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("read body: %v", err)
				}
				if string(body) != tt.wantBody {
					t.Errorf("body = %s\nwant   %s", body, tt.wantBody)
				}
			}
		})
	}
}
