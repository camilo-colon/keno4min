package usecase

import (
	"context"
	"fmt"
	"uuid"

	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
)

type GetByCoupon struct {
	repository ports.Repository
}

func NewGetByCoupon(repository ports.Repository) *GetByCoupon {
	return &GetByCoupon{repository: repository}
}

type GetByCouponParams struct {
	Coupon string
	NodeID uuid.UUID
}

func (g *GetByCoupon) Execute(ctx context.Context, params GetByCouponParams) (*tickets.Ticket, error) {
	ticket, err := g.repository.GetByCoupon(ctx, params.Coupon)
	if err != nil {
		return nil, err
	}
	// A ticket of another club is reported as missing so its existence is not revealed.
	if ticket.NodeID != params.NodeID {
		return nil, fmt.Errorf("%w: coupon %s", ports.ErrTicketNotFound, params.Coupon)
	}
	return ticket, nil
}
