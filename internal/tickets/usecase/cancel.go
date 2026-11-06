package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	drawports "github.com/cronos/keno4min/internal/draws/ports"
	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
)

// Cancel voids a pending ticket of the open draw and refunds its total to the node.
type Cancel struct {
	repository ports.Repository
	draws      drawports.Repository
	balance    ports.BalanceGateway
	now        func() time.Time
}

func NewCancel(repository ports.Repository, draws drawports.Repository, balance ports.BalanceGateway) *Cancel {
	return &Cancel{repository: repository, draws: draws, balance: balance, now: time.Now}
}

type CancelParams struct {
	TicketID    uuid.UUID
	NodeID      uuid.UUID
	CancelledBy string
}

// CancelResult carries the node balance after the refund.
type CancelResult struct {
	BalanceCents int64
}

func (c *Cancel) Execute(ctx context.Context, params CancelParams) (*CancelResult, error) {
	ticket, err := c.repository.GetByID(ctx, params.TicketID)
	if err != nil {
		return nil, err
	}
	// A ticket of another club is reported as missing so its existence is not revealed.
	if ticket.NodeID != params.NodeID {
		return nil, fmt.Errorf("%w: %s", ports.ErrTicketNotFound, params.TicketID)
	}

	now := c.now().UTC()
	draw, err := c.draws.GetCurrent(ctx)
	if err != nil || draw == nil {
		return nil, ErrDrawUnavailable
	}
	if ticket.TIR != draw.TIR || !draw.AcceptsTicketsAt(now) {
		return nil, tickets.ErrBettingClosed
	}

	if err := ticket.Cancel(params.CancelledBy, now); err != nil {
		return nil, err
	}
	credit, err := c.balance.Credit(ctx, cancellationCredit(ticket))
	if err != nil {
		return nil, err
	}
	if err := c.repository.Update(ctx, ticket); err != nil {
		_, debitErr := c.balance.Debit(context.WithoutCancel(ctx), cancellationReversal(ticket))
		return nil, errors.Join(err, debitErr)
	}
	return &CancelResult{BalanceCents: credit.BalanceAfterCents}, nil
}

func cancellationCredit(ticket *tickets.Ticket) ports.BalanceMovement {
	return ports.BalanceMovement{
		NodeID:         ticket.NodeID,
		AmountCents:    ticket.TotalCents(),
		Reason:         ports.ReasonTicketCancellation,
		Reference:      ticket.ID.String(),
		IdempotencyKey: fmt.Sprintf("ticket:%s:cancel-credit", ticket.ID),
	}
}

func cancellationReversal(ticket *tickets.Ticket) ports.BalanceMovement {
	movement := cancellationCredit(ticket)
	movement.Reason = ports.ReasonTicketCancellationReversal
	movement.IdempotencyKey = fmt.Sprintf("ticket:%s:cancel-credit-reversal", ticket.ID)
	return movement
}
