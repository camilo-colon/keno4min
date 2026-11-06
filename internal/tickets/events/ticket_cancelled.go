package events

import (
	"uuid"

	"github.com/cronos/keno4min/internal/common"
)

var _ common.DomainEvent = (*TicketCancelled)(nil)

const TicketCancelledType = "cronos.keno4min.ticket.cancelled"

// TicketCancelled reports a cancelled ticket; ValueCents is the total already refunded.
type TicketCancelled struct {
	common.BaseEvent

	TicketID    uuid.UUID `json:"ticketId"`
	Game        string    `json:"game"`
	CancelledBy string    `json:"cancelledBy"`
	ValueCents  int64     `json:"valueCents"`
}

func (t TicketCancelled) Type() string {
	return TicketCancelledType
}
