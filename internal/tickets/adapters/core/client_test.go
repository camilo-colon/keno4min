package core

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"uuid"

	"github.com/cronos/keno4min/internal/config"
	"github.com/cronos/keno4min/internal/tickets/ports"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClient(&config.Config{Core: config.CoreConfig{BaseURL: server.URL + "/", Timeout: time.Second}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return client
}

func TestClientSendsMovement(t *testing.T) {
	movement := ports.BalanceMovement{
		NodeID:         uuid.NewV7(),
		AmountCents:    5000,
		Reason:         ports.ReasonTicketSale,
		Reference:      "ticket-1",
		IdempotencyKey: "ticket:1:debit",
	}

	for _, tt := range []struct {
		name string
		send func(*Client) (ports.BalanceResult, error)
		path string
	}{
		{"debit", func(c *Client) (ports.BalanceResult, error) { return c.Debit(context.Background(), movement) }, "/nodes/" + movement.NodeID.String() + "/balance/debits"},
		{"credit", func(c *Client) (ports.BalanceResult, error) { return c.Credit(context.Background(), movement) }, "/nodes/" + movement.NodeID.String() + "/balance/credits"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var got movementRequest
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != tt.path {
					t.Errorf("got %s %s, want POST %s", r.Method, r.URL.Path, tt.path)
				}
				if key := r.Header.Get("Idempotency-Key"); key != movement.IdempotencyKey {
					t.Errorf("Idempotency-Key = %q, want %q", key, movement.IdempotencyKey)
				}
				if ct := r.Header.Get("Content-Type"); ct != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", ct)
				}
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Errorf("decode body: %v", err)
				}
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"id":"mv-1","balanceAfterCents":95000}`))
			})

			result, err := tt.send(client)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if want := (ports.BalanceResult{MovementID: "mv-1", BalanceAfterCents: 95000}); result != want {
				t.Fatalf("result = %+v, want %+v", result, want)
			}
			want := movementRequest{Origin: "keno4min", AmountCents: 5000, Reason: "TicketSale", Reference: "ticket-1"}
			if got != want {
				t.Fatalf("body = %+v, want %+v", got, want)
			}
		})
	}
}

func TestClientMapsStatus(t *testing.T) {
	for _, tt := range []struct {
		status int
		want   error
	}{
		{http.StatusOK, nil},
		{http.StatusNotFound, ports.ErrBalanceNodeNotFound},
		{http.StatusUnprocessableEntity, ports.ErrInsufficientBalance},
		{http.StatusBadGateway, ports.ErrBalanceUnavailable},
	} {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(`{"id":"mv-1","balanceAfterCents":0}`))
			})
			_, err := client.Debit(context.Background(), ports.BalanceMovement{NodeID: uuid.NewV7()})
			if !errors.Is(err, tt.want) || (tt.want == nil && err != nil) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}

	t.Run("unexpected status is not a domain error", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadRequest) })
		_, err := client.Debit(context.Background(), ports.BalanceMovement{NodeID: uuid.NewV7()})
		if err == nil || errors.Is(err, ports.ErrInsufficientBalance) || errors.Is(err, ports.ErrBalanceUnavailable) {
			t.Fatalf("got %v, want a generic error", err)
		}
	})
}

func TestClientReportsUnreachableCoreAsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	client, err := NewClient(&config.Config{Core: config.CoreConfig{BaseURL: server.URL, Timeout: time.Second}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := client.Debit(context.Background(), ports.BalanceMovement{NodeID: uuid.NewV7()}); !errors.Is(err, ports.ErrBalanceUnavailable) {
		t.Fatalf("got %v, want ErrBalanceUnavailable", err)
	}
}

func TestNewClientRequiresBaseURL(t *testing.T) {
	if _, err := NewClient(&config.Config{}); !errors.Is(err, ErrMissingBaseURL) {
		t.Fatalf("got %v, want ErrMissingBaseURL", err)
	}
}

func TestClientRejectsInvalidSuccessBody(t *testing.T) {
	for name, body := range map[string]string{
		"not json":            `nope`,
		"missing balance":     `{"id":"mv-1"}`,
		"missing movement id": `{"balanceAfterCents":95000}`,
	} {
		t.Run(name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) })
			_, err := client.Debit(context.Background(), ports.BalanceMovement{NodeID: uuid.NewV7()})
			if !errors.Is(err, ErrInvalidCoreResponse) {
				t.Fatalf("got %v, want ErrInvalidCoreResponse", err)
			}
		})
	}
}
