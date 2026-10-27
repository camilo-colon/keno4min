package usecase

import (
	"context"
	"errors"
	"time"
	"uuid"

	drawports "github.com/cronos/keno4min/internal/draws/ports"
	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
	userports "github.com/cronos/keno4min/internal/users/ports"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrDrawUnavailable = errors.New("current draw is unavailable")
)

type Create struct {
	repository ports.Repository
	users      userports.Repository
	draws      drawports.Repository
	now        func() time.Time
}

func NewCreate(repository ports.Repository, users userports.Repository, draws drawports.Repository) *Create {
	return &Create{repository: repository, users: users, draws: draws, now: time.Now}
}

type CreateParams struct {
	UserID uuid.UUID
	Bets   []tickets.Bet
}

func (c *Create) Execute(ctx context.Context, params CreateParams) (*tickets.Ticket, error) {
	user, err := c.users.FindById(ctx, params.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
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

	ticket, err := tickets.Create(uuid.NewV7(), coupon, user.NodeId, draw.TIR, params.Bets, user.Username)
	if err != nil {
		return nil, err
	}
	if err := c.repository.Save(ctx, ticket); err != nil {
		return nil, err
	}
	return ticket, nil
}
