package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cronos/keno4min/internal/config"
	"github.com/cronos/keno4min/internal/tickets/ports"
)

const (
	origin          = "keno4min"
	maxDrainedBytes = 64 << 10
)

var (
	ErrMissingBaseURL      = errors.New("core base url is not configured")
	ErrInvalidCoreResponse = errors.New("core returned an invalid balance response")
)

// Client implements ports.BalanceGateway against the core REST API.
// Calls between microservices are trusted, so no authentication is sent.
type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(cfg *config.Config) (*Client, error) {
	baseURL := strings.TrimRight(cfg.Core.BaseURL, "/")
	if baseURL == "" {
		return nil, ErrMissingBaseURL
	}
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: cfg.Core.Timeout}}, nil
}

type movementRequest struct {
	Origin      string `json:"origin"`
	AmountCents int64  `json:"amountCents"`
	Reason      string `json:"reason"`
	Reference   string `json:"reference"`
}

type movementResponse struct {
	ID                string `json:"id"`
	BalanceAfterCents *int64 `json:"balanceAfterCents"`
}

func (c *Client) Debit(ctx context.Context, movement ports.BalanceMovement) (ports.BalanceResult, error) {
	return c.send(ctx, "debits", movement)
}

func (c *Client) Credit(ctx context.Context, movement ports.BalanceMovement) (ports.BalanceResult, error) {
	return c.send(ctx, "credits", movement)
}

func (c *Client) send(ctx context.Context, kind string, movement ports.BalanceMovement) (ports.BalanceResult, error) {
	body, err := json.Marshal(movementRequest{
		Origin:      origin,
		AmountCents: movement.AmountCents,
		Reason:      string(movement.Reason),
		Reference:   movement.Reference,
	})
	if err != nil {
		return ports.BalanceResult{}, fmt.Errorf("core %s: encode request: %w", kind, err)
	}

	url := fmt.Sprintf("%s/nodes/%s/balance/%s", c.baseURL, movement.NodeID, kind)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return ports.BalanceResult{}, fmt.Errorf("core %s: build request: %w", kind, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", movement.IdempotencyKey)

	res, err := c.http.Do(req)
	if err != nil {
		return ports.BalanceResult{}, fmt.Errorf("%w: %s: %w", ports.ErrBalanceUnavailable, kind, err)
	}
	defer res.Body.Close()
	response := io.LimitReader(res.Body, maxDrainedBytes)
	// Draining the body lets the transport reuse the connection.
	defer func() { _, _ = io.Copy(io.Discard, response) }()

	if err := mapStatus(kind, res.StatusCode); err != nil {
		return ports.BalanceResult{}, err
	}
	return decodeResult(kind, response)
}

func decodeResult(kind string, body io.Reader) (ports.BalanceResult, error) {
	var response movementResponse
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return ports.BalanceResult{}, fmt.Errorf("%w: %s: %w", ErrInvalidCoreResponse, kind, err)
	}
	if response.ID == "" || response.BalanceAfterCents == nil {
		return ports.BalanceResult{}, fmt.Errorf("%w: %s: missing id or balanceAfterCents", ErrInvalidCoreResponse, kind)
	}
	return ports.BalanceResult{MovementID: response.ID, BalanceAfterCents: *response.BalanceAfterCents}, nil
}

// mapStatus translates core responses into domain errors.
// The 404 and 422 codes are pending confirmation with the core team.
func mapStatus(kind string, status int) error {
	switch {
	case status >= 200 && status < 300:
		return nil
	case status == http.StatusNotFound:
		return ports.ErrBalanceNodeNotFound
	case status == http.StatusUnprocessableEntity:
		return ports.ErrInsufficientBalance
	case status >= 500:
		return fmt.Errorf("%w: %s: status %d", ports.ErrBalanceUnavailable, kind, status)
	default:
		return fmt.Errorf("core %s: unexpected status %d", kind, status)
	}
}
