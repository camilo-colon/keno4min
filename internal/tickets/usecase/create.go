package usecase

import (
	"context"
	"errors"
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
	now        func() time.Time
}

func NewCreate(repository ports.Repository, draws drawports.Repository) *Create {
	return &Create{repository: repository, draws: draws, now: time.Now}
}

type CreateParams struct {
	NodeID    uuid.UUID
	CreatedBy string
	Bets      []tickets.Bet
}

func (c *Create) Execute(ctx context.Context, params CreateParams) (*tickets.Ticket, error) {
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
	if err := c.repository.Save(ctx, ticket); err != nil {
		return nil, err
	}
	return ticket, nil
}
