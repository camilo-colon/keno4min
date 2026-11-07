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

type PayResult struct {
	PrizeCents   int64
	BalanceCents int64
}

func (p *Pay) Execute(ctx context.Context, params PayParams) (*PayResult, error) {
	ticket, err := p.repository.GetByID(ctx, params.TicketID)
	if err != nil {
		return nil, err
	}
	if ticket.NodeID != params.NodeID {
		return nil, fmt.Errorf("%w: %s", ports.ErrTicketNotFound, params.TicketID)
	}
	return p.payTicket(ctx, ticket, params.PaidBy)
}

// payTicket pays an already loaded ticket; the caller has checked it belongs to the node.
func (p *Pay) payTicket(ctx context.Context, ticket *tickets.Ticket, paidBy string) (*PayResult, error) {
	if err := ticket.Pay(paidBy, p.now().UTC()); err != nil {
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
