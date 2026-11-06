package http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/cronos/keno4min/internal/platform/auth"
	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
	"github.com/cronos/keno4min/internal/tickets/usecase"
	"github.com/gofiber/fiber/v3"
)

// Embedded interface is nil: calling a method the test does not stub panics.
type payRepository struct {
	ports.Repository
	ticket    *tickets.Ticket
	getErr    error
	updateErr error
}

func (r *payRepository) GetByID(context.Context, uuid.UUID) (*tickets.Ticket, error) {
	return r.ticket, r.getErr
}

func (r *payRepository) Update(context.Context, *tickets.Ticket) error {
	return r.updateErr
}

type payBalance struct {
	creditErr error
}

func (b payBalance) Credit(context.Context, ports.BalanceMovement) (ports.BalanceResult, error) {
	if b.creditErr != nil {
		return ports.BalanceResult{}, b.creditErr
	}
	return ports.BalanceResult{MovementID: "mv-1", BalanceAfterCents: 125_000}, nil
}

func (b payBalance) Debit(context.Context, ports.BalanceMovement) (ports.BalanceResult, error) {
	return ports.BalanceResult{}, nil
}

// testToken builds an unsigned JWT, which auth.New accepts because the gateway verifies signatures.
func testToken(t *testing.T, nodeID uuid.UUID) string {
	t.Helper()
	payload, err := json.Marshal(map[string]string{
		"sub": uuid.NewV7().String(), "unique_name": "cashier", "role": "cashier", "node_id": nodeID.String(),
	})
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	return "e30." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

func TestPayTicketHandler(t *testing.T) {
	nodeID := uuid.NewV7()
	ticketID := uuid.NewV7()

	for _, tt := range []struct {
		name       string
		path       string
		noToken    bool
		status     tickets.Status
		getErr     error
		creditErr  error
		updateErr  error
		wantStatus int
		wantBody   string
	}{
		{name: "pays the prize", status: tickets.StatusWinning, wantStatus: fiber.StatusOK, wantBody: `{"prizeCents":17500,"balanceCents":125000}`},
		{name: "missing token", noToken: true, wantStatus: fiber.StatusUnauthorized},
		{name: "invalid ticket id", path: "/tickets/not-a-uuid/pay", wantStatus: fiber.StatusBadRequest},
		{name: "missing ticket", getErr: ports.ErrTicketNotFound, wantStatus: fiber.StatusNotFound},
		{name: "already paid", status: tickets.StatusPaid, wantStatus: fiber.StatusConflict, wantBody: `{"error":"ticket is already paid"}`},
		{name: "not payable", status: tickets.StatusLost, wantStatus: fiber.StatusConflict, wantBody: `{"error":"ticket has no prize to pay"}`},
		{name: "balance unavailable", status: tickets.StatusWinning, creditErr: ports.ErrBalanceUnavailable, wantStatus: fiber.StatusServiceUnavailable},
		{name: "unexpected failure", status: tickets.StatusWinning, updateErr: errors.New("boom"), wantStatus: fiber.StatusInternalServerError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repository := &payRepository{getErr: tt.getErr, updateErr: tt.updateErr}
			if tt.getErr == nil {
				repository.ticket = &tickets.Ticket{ID: ticketID, NodeID: nodeID, Status: tt.status, PrizeCents: 17_500}
			}
			handler := NewPayTicketHandler(usecase.NewPay(repository, payBalance{creditErr: tt.creditErr}))
			app := fiber.New()
			app.Use(auth.New())
			app.Post("/tickets/:id/pay", handler.Handle)

			path := tt.path
			if path == "" {
				path = "/tickets/" + ticketID.String() + "/pay"
			}
			req := httptest.NewRequest(fiber.MethodPost, path, nil)
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
					t.Errorf("body = %s, want %s", body, tt.wantBody)
				}
			}
		})
	}
}
