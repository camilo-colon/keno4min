package ports

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/cronos/keno4min/internal/tickets"
)

var (
	ErrInvalidCursor  = errors.New("invalid pagination cursor")
	ErrTicketNotFound = errors.New("ticket not found")
)

type ListParams struct {
	NodeID uuid.UUID
	From   time.Time
	To     time.Time
	Limit  int32
	Cursor string
}

type TicketPage struct {
	Items      []*tickets.Ticket
	NextCursor string
}

type Repository interface {
	GenerateCoupon(context.Context) (string, error)
	Save(context.Context, *tickets.Ticket) error
	GetByID(context.Context, uuid.UUID) (*tickets.Ticket, error)
	ListByNode(context.Context, ListParams) (TicketPage, error)
}
