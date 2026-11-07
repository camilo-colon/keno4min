package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	if err := validateDateRange(params.From, params.To); err != nil {
		return ports.TicketPage{}, fmt.Errorf("%w: %w", ErrInvalidListParams, err)
	}
	if params.Limit == 0 {
		params.Limit = 50
	}
	if params.Limit < 1 || params.Limit > 100 {
		return ports.TicketPage{}, fmt.Errorf("%w: limit must be between 1 and 100", ErrInvalidListParams)
	}
	return l.repository.ListByNode(ctx, params)
}

// validateDateRange checks a [from, to) range as the tickets index stores it.
func validateDateRange(from, to time.Time) error {
	if from.IsZero() || to.IsZero() || !from.Before(to) {
		return errors.New("from must be before to")
	}
	if from.Nanosecond()%int(time.Millisecond) != 0 || to.Nanosecond()%int(time.Millisecond) != 0 {
		return errors.New("dates must have at most millisecond precision")
	}
	return nil
}
