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

// Pay closes a winning ticket and credits its prize to the node, since the cashier
// handed it over in cash.
//
// Ownership is checked by node only: any cashier of the selling club can pay. The PRD
// requires the selling cashier, which stays pending until the ticket can identify them.
type Pay struct {
	repository ports.Repository
	balance    ports.BalanceGateway
	now        func() time.Time
}

func NewPay(repository ports.Repository, balance ports.BalanceGateway) *Pay {
	return &Pay{repository: repository, balance: balance, now: time.Now}
}

type PayParams struct {
	TicketID uuid.UUID
	NodeID   uuid.UUID
	PaidBy   string
}

// PayResult carries the paid prize and the node balance after the credit.
type PayResult struct {
	PrizeCents   int64
	BalanceCents int64
}

func (p *Pay) Execute(ctx context.Context, params PayParams) (*PayResult, error) {
	ticket, err := p.repository.GetByID(ctx, params.TicketID)
	if err != nil {
		return nil, err
	}
	// A ticket of another club is reported as missing so its existence is not revealed.
	if ticket.NodeID != params.NodeID {
		return nil, fmt.Errorf("%w: %s", ports.ErrTicketNotFound, params.TicketID)
	}

	if err := ticket.Pay(params.PaidBy, p.now().UTC()); err != nil {
		return nil, err
	}
	credit, err := p.balance.Credit(ctx, paymentCredit(ticket))
	if err != nil {
		return nil, err
	}
	if err := p.repository.Update(ctx, ticket); err != nil {
		_, debitErr := p.balance.Debit(context.WithoutCancel(ctx), paymentReversal(ticket))
		return nil, errors.Join(err, debitErr)
	}
	return &PayResult{PrizeCents: ticket.PrizeCents, BalanceCents: credit.BalanceAfterCents}, nil
}

func paymentCredit(ticket *tickets.Ticket) ports.BalanceMovement {
	return ports.BalanceMovement{
		NodeID:         ticket.NodeID,
		AmountCents:    ticket.PrizeCents,
		Reason:         ports.ReasonTicketPayment,
		Reference:      ticket.ID.String(),
		IdempotencyKey: fmt.Sprintf("ticket:%s:pay-credit", ticket.ID),
	}
}

func paymentReversal(ticket *tickets.Ticket) ports.BalanceMovement {
	movement := paymentCredit(ticket)
	movement.Reason = ports.ReasonTicketPaymentReversal
	movement.IdempotencyKey = fmt.Sprintf("ticket:%s:pay-credit-reversal", ticket.ID)
	return movement
}
