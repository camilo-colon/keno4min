package usecase

import (
	"context"
	"errors"
	"slices"
	"uuid"

	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
)

var ErrInvalidLookupParams = errors.New("invalid ticket lookup")

type Lookup struct {
	repository ports.Repository
}

func NewLookup(repository ports.Repository) *Lookup {
	return &Lookup{repository: repository}
}

type LookupParams struct {
	TicketIDs []uuid.UUID
	NodeID    uuid.UUID
}

func (l *Lookup) Execute(ctx context.Context, params LookupParams) ([]*tickets.Ticket, error) {
	ids := unique(params.TicketIDs)
	found, err := l.repository.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	// Tickets of another club are omitted so their existence is not revealed.
	return slices.DeleteFunc(found, func(ticket *tickets.Ticket) bool {
		return ticket.NodeID != params.NodeID
	}), nil
}

func unique(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]bool, len(ids))
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result
}
