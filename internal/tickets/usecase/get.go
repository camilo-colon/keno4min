package usecase

import (
	"context"
	"fmt"
	"uuid"

	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
)

type Get struct {
	repository ports.Repository
}

func NewGet(repository ports.Repository) *Get {
	return &Get{repository: repository}
}

type GetParams struct {
	TicketID uuid.UUID
	NodeID   uuid.UUID
}

func (g *Get) Execute(ctx context.Context, params GetParams) (*tickets.Ticket, error) {
	ticket, err := g.repository.GetByID(ctx, params.TicketID)
	if err != nil {
		return nil, err
	}
	// A ticket of another club is reported as missing so its existence is not revealed.
	if ticket.NodeID != params.NodeID {
		return nil, fmt.Errorf("%w: %s", ports.ErrTicketNotFound, params.TicketID)
	}
	return ticket, nil
}
