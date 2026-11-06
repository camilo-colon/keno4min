package ports

import (
	"context"
	"errors"
	"uuid"
)

type BalanceReason string

const (
	ReasonTicketSale         BalanceReason = "TicketSale"
	ReasonTicketSaleReversal BalanceReason = "TicketSaleReversal"

	ReasonTicketCancellation         BalanceReason = "TicketCancellation"
	ReasonTicketCancellationReversal BalanceReason = "TicketCancellationReversal"
)

// BalanceMovement is a debit or credit over the balance of a node.
type BalanceMovement struct {
	NodeID         uuid.UUID
	AmountCents    int64
	Reason         BalanceReason
	Reference      string
	IdempotencyKey string
}

// BalanceResult is what core reports after applying a movement.
type BalanceResult struct {
	MovementID        string
	BalanceAfterCents int64
}

var (
	ErrInsufficientBalance = errors.New("insufficient node balance")
	ErrBalanceNodeNotFound = errors.New("node not found in core")
	ErrBalanceUnavailable  = errors.New("balance service unavailable")
)

// BalanceGateway moves money over the balance of a node owned by core.
// Implementations must be safe to retry with the same IdempotencyKey.
type BalanceGateway interface {
	Debit(ctx context.Context, movement BalanceMovement) (BalanceResult, error)
	Credit(ctx context.Context, movement BalanceMovement) (BalanceResult, error)
}
