package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/cronos/keno4min/internal/tickets/ports"
)

var ErrInvalidListParams = errors.New("invalid ticket filters")

type List struct {
	repository ports.Repository
}

func NewList(repository ports.Repository) *List {
	return &List{repository: repository}
}

func (l *List) Execute(ctx context.Context, params ports.ListParams) (ports.TicketPage, error) {
	if params.NodeID == uuid.Nil() {
		return ports.TicketPage{}, fmt.Errorf("%w: nodeId must be a non-zero UUID", ErrInvalidListParams)
	}
	if params.From.IsZero() || params.To.IsZero() || !params.From.Before(params.To) {
		return ports.TicketPage{}, fmt.Errorf("%w: from must be before to", ErrInvalidListParams)
	}
	if params.From.Nanosecond()%int(time.Millisecond) != 0 || params.To.Nanosecond()%int(time.Millisecond) != 0 {
		return ports.TicketPage{}, fmt.Errorf("%w: dates must have at most millisecond precision", ErrInvalidListParams)
	}
	if params.Limit == 0 {
		params.Limit = 50
	}
	if params.Limit < 1 || params.Limit > 100 {
		return ports.TicketPage{}, fmt.Errorf("%w: limit must be between 1 and 100", ErrInvalidListParams)
	}
	return l.repository.ListByNode(ctx, params)
}
