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

var ErrDrawUnavailable = errors.New("current draw is unavailable")

type Create struct {
	repository ports.Repository
	draws      drawports.Repository
	balance    ports.BalanceGateway
	now        func() time.Time
}

func NewCreate(repository ports.Repository, draws drawports.Repository, balance ports.BalanceGateway) *Create {
	return &Create{repository: repository, draws: draws, balance: balance, now: time.Now}
}

type CreateParams struct {
	NodeID    uuid.UUID
	CreatedBy string
	Bets      []tickets.Bet
}

// CreateResult carries the sold ticket and the node balance after the sale.
type CreateResult struct {
	Ticket       *tickets.Ticket
	BalanceCents int64
}

func (c *Create) Execute(ctx context.Context, params CreateParams) (*CreateResult, error) {
	draw, err := c.draws.GetCurrent(ctx)
	if err != nil || draw == nil {
		return nil, ErrDrawUnavailable
	}
	if !draw.AcceptsTicketsAt(c.now().UTC()) {
		return nil, tickets.ErrBettingClosed
	}
	coupon, err := c.repository.GenerateCoupon(ctx)
	if err != nil {
		return nil, err
	}

	ticket, err := tickets.Create(uuid.NewV7(), coupon, params.NodeID, draw.TIR, params.Bets, params.CreatedBy)
	if err != nil {
		return nil, err
	}
	debit, err := c.balance.Debit(ctx, saleDebit(ticket))
	if err != nil {
		return nil, err
	}
	if err := c.repository.Save(ctx, ticket); err != nil {
		_, refundErr := c.balance.Credit(context.WithoutCancel(ctx), saleReversal(ticket))
		return nil, errors.Join(err, refundErr)
	}
	return &CreateResult{Ticket: ticket, BalanceCents: debit.BalanceAfterCents}, nil
}

func saleDebit(ticket *tickets.Ticket) ports.BalanceMovement {
	return ports.BalanceMovement{
		NodeID:         ticket.NodeID,
		AmountCents:    ticket.TotalCents(),
		Reason:         ports.ReasonTicketSale,
		Reference:      ticket.ID.String(),
		IdempotencyKey: fmt.Sprintf("ticket:%s:debit", ticket.ID),
	}
}

func saleReversal(ticket *tickets.Ticket) ports.BalanceMovement {
	movement := saleDebit(ticket)
	movement.Reason = ports.ReasonTicketSaleReversal
	movement.IdempotencyKey = fmt.Sprintf("ticket:%s:debit-reversal", ticket.ID)
	return movement
}
