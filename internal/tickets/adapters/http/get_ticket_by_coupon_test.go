package http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

// ticketRepository implements only the reads used by the get handlers.
type ticketRepository struct {
	ports.Repository
	ticket *tickets.Ticket
}

func (r ticketRepository) GetByID(_ context.Context, id uuid.UUID) (*tickets.Ticket, error) {
	if r.ticket == nil || r.ticket.ID != id {
		return nil, fmt.Errorf("%w: %s", ports.ErrTicketNotFound, id)
	}
	return r.ticket, nil
}

func (r ticketRepository) GetByCoupon(_ context.Context, coupon string) (*tickets.Ticket, error) {
	if r.ticket == nil || r.ticket.Coupon != coupon {
		return nil, fmt.Errorf("%w: coupon %s", ports.ErrTicketNotFound, coupon)
	}
	return r.ticket, nil
}

func TestGetTicketByCouponAnswersLikeGetTicket(t *testing.T) {
	ticket := newTestTicket(t)
	app := newTicketsApp(ticketRepository{ticket: ticket})
	token := testToken(t, ticket.NodeID)

	status, byCoupon := get(t, app, "/tickets/coupon/0001", token)
	if status != fiber.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", status, byCoupon)
	}
	_, byID := get(t, app, "/tickets/"+ticket.ID.String(), token)
	if byCoupon != byID {
		t.Fatalf("body by coupon = %s, want the body by id %s", byCoupon, byID)
	}
}

func TestGetTicketByCouponOfMissingCouponIsNotFound(t *testing.T) {
	ticket := newTestTicket(t)
	app := newTicketsApp(ticketRepository{ticket: ticket})

	status, body := get(t, app, "/tickets/coupon/9999", testToken(t, ticket.NodeID))
	if status != fiber.StatusNotFound {
		t.Fatalf("status = %d, want 404; body %s", status, body)
	}
}

func TestGetTicketByCouponOfAnotherClubIsNotFound(t *testing.T) {
	ticket := newTestTicket(t)
	app := newTicketsApp(ticketRepository{ticket: ticket})

	status, body := get(t, app, "/tickets/coupon/0001", testToken(t, uuid.NewV7()))
	if status != fiber.StatusNotFound {
		t.Fatalf("status = %d, want 404; body %s", status, body)
	}
}

// newTicketsApp registers the real routes; only the get handlers are wired.
func newTicketsApp(repository ports.Repository) *fiber.App {
	app := fiber.New()
	app.Use(auth.New())
	getTicket := NewGetTicketHandler(usecase.NewGet(repository))
	getByCoupon := NewGetTicketByCouponHandler(usecase.NewGetByCoupon(repository))
	NewRoutes(nil, nil, getTicket, getByCoupon, nil, nil, nil, nil, nil).Register(app)
	return app
}

func get(t *testing.T, app *fiber.App, path, token string) (int, string) {
	t.Helper()
	request := httptest.NewRequest(fiber.MethodGet, path, nil)
	request.Header.Set(fiber.HeaderAuthorization, "Bearer "+token)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(body)
}

// testToken builds an unsigned JWT; the API trusts the gateway to verify signatures.
func testToken(t *testing.T, nodeID uuid.UUID) string {
	t.Helper()
	payload, err := json.Marshal(map[string]string{
		"sub": uuid.NewV7().String(), "unique_name": "cashier", "role": "cashier", "node_id": nodeID.String(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return "e30." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

func newTestTicket(t *testing.T) *tickets.Ticket {
	bets := []tickets.Bet{{Nums: []uint{1, 2, 3}, ValueCents: 500}}
	ticket, err := tickets.Create(uuid.NewV7(), "0001", uuid.NewV7(), uuid.NewV7(), 4321, bets, "cashier")
	if err != nil {
		t.Fatal(err)
	}
	return ticket
}
