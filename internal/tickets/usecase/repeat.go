package usecase

import (
	"context"
	"fmt"
	"slices"
	"uuid"

	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
)

// Repeat sells the bets of an existing ticket again in the current draw.
type Repeat struct {
	repository ports.Repository
	create     *Create
}

func NewRepeat(repository ports.Repository, create *Create) *Repeat {
	return &Repeat{repository: repository, create: create}
}

type RepeatParams struct {
	TicketID  uuid.UUID
	NodeID    uuid.UUID
	CreatedBy string
}

func (r *Repeat) Execute(ctx context.Context, params RepeatParams) (*CreateResult, error) {
	original, err := r.repository.GetByID(ctx, params.TicketID)
	if err != nil {
		return nil, err
	}
	// A ticket of another club is reported as missing so its existence is not revealed.
	if original.NodeID != params.NodeID {
		return nil, fmt.Errorf("%w: %s", ports.ErrTicketNotFound, params.TicketID)
	}

	return r.create.Execute(ctx, CreateParams{
		NodeID:    params.NodeID,
		CreatedBy: params.CreatedBy,
		Bets:      cloneBets(original.Bets),
	})
}

// cloneBets copies the numbers so the new ticket never shares memory with the original.
func cloneBets(bets []tickets.Bet) []tickets.Bet {
	result := make([]tickets.Bet, len(bets))
	for i, bet := range bets {
		result[i] = tickets.Bet{Nums: slices.Clone(bet.Nums), ValueCents: bet.ValueCents}
	}
	return result
}
