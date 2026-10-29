package usecase

import (
	"context"
	"fmt"
	"uuid"

	drawdomain "github.com/cronos/keno4min/internal/draws"
	drawports "github.com/cronos/keno4min/internal/draws/ports"
	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
)

type Get struct {
	repository ports.Repository
	draws      drawports.Repository
}

func NewGet(repository ports.Repository, draws drawports.Repository) *Get {
	return &Get{repository: repository, draws: draws}
}

type GetParams struct {
	TicketID uuid.UUID
	NodeID   uuid.UUID
}

// TicketDetail is a ticket with the balls of its draw; Balls is empty until the draw status is drawn.
type TicketDetail struct {
	Ticket *tickets.Ticket
	Balls  []int
}

func (g *Get) Execute(ctx context.Context, params GetParams) (*TicketDetail, error) {
	ticket, err := g.repository.GetByID(ctx, params.TicketID)
	if err != nil {
		return nil, err
	}
	// A ticket of another club is reported as missing so its existence is not revealed.
	if ticket.NodeID != params.NodeID {
		return nil, fmt.Errorf("%w: %s", ports.ErrTicketNotFound, params.TicketID)
	}

	draw, err := g.draws.FindByTIR(ctx, ticket.TIR)
	if err != nil {
		return nil, err
	}
	if draw == nil {
		return nil, fmt.Errorf("draw with tir %d of ticket %s not found", ticket.TIR, ticket.ID)
	}
	// Balls may be stored before the draw is played; expose them only once it is drawn.
	balls := []int{}
	if draw.Status == drawdomain.StatusDrawn {
		balls = draw.Balls
	}
	return &TicketDetail{Ticket: ticket, Balls: balls}, nil
}
