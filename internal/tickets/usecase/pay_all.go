package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
)

const payAllPageSize = 100

var ErrInvalidPayAllParams = errors.New("invalid pay all filters")

// PayAll pays every winning ticket of a node sold within [From, To). The range has no
// maximum length: the caller decides how far back to pay.
//
// Tickets are paid one by one and a failure does not stop the rest: each payment keeps
// its own idempotency key in core, so retrying the whole batch never credits twice.
type PayAll struct {
	repository ports.Repository
	pay        *Pay
}

func NewPayAll(repository ports.Repository, pay *Pay) *PayAll {
	return &PayAll{repository: repository, pay: pay}
}

type PayAllParams struct {
	NodeID uuid.UUID
	PaidBy string
	From   time.Time
	To     time.Time
}

type PaidTicket struct {
	TicketID   uuid.UUID
	Coupon     string
	PrizeCents int64
}

type FailedTicket struct {
	TicketID uuid.UUID
	Coupon   string
	Err      error
}

type PayAllResult struct {
	Paid   []PaidTicket
	Failed []FailedTicket
	// BalanceCents is the node balance after the last successful credit; nil when nothing was paid.
	BalanceCents *int64
}

func (r *PayAllResult) TotalPrizeCents() int64 {
	var total int64
	for _, paid := range r.Paid {
		total += paid.PrizeCents
	}
	return total
}

func (p *PayAll) Execute(ctx context.Context, params PayAllParams) (*PayAllResult, error) {
	if err := validateDateRange(params.From, params.To); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidPayAllParams, err)
	}
	// All winners are collected before paying, so a listing failure aborts without side effects.
	winners, err := p.winners(ctx, ports.ListParams{NodeID: params.NodeID, From: params.From, To: params.To, Limit: payAllPageSize})
	if err != nil {
		return nil, err
	}

	result := &PayAllResult{Paid: []PaidTicket{}, Failed: []FailedTicket{}}
	for _, ticket := range winners {
		paid, err := p.pay.payTicket(ctx, ticket, params.PaidBy)
		if err != nil {
			result.Failed = append(result.Failed, FailedTicket{TicketID: ticket.ID, Coupon: ticket.Coupon, Err: err})
			continue
		}
		result.Paid = append(result.Paid, PaidTicket{TicketID: ticket.ID, Coupon: ticket.Coupon, PrizeCents: paid.PrizeCents})
		result.BalanceCents = &paid.BalanceCents
	}
	return result, nil
}

func (p *PayAll) winners(ctx context.Context, params ports.ListParams) ([]*tickets.Ticket, error) {
	var winners []*tickets.Ticket
	for {
		page, err := p.repository.ListByNode(ctx, params)
		if err != nil {
			return nil, err
		}
		for _, ticket := range page.Items {
			if ticket.Status == tickets.StatusWinning {
				winners = append(winners, ticket)
			}
		}
		if page.NextCursor == "" {
			return winners, nil
		}
		params.Cursor = page.NextCursor
	}
}
