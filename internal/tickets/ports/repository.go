package ports

import (
	"context"

	"github.com/cronos/keno4min/internal/tickets"
)

type Repository interface {
	GenerateCoupon(context.Context) (string, error)
	Save(context.Context, *tickets.Ticket) error
}
