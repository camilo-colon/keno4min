package events

import (
	"uuid"

	"github.com/cronos/keno4min/internal/common"
)

var _ common.DomainEvent = (*TicketPaid)(nil)

const TicketPaidType = "cronos.keno4min.ticket.paid"

// TicketPaid reports a prize handed over in cash; PrizeCents is the amount credited to the node.
type TicketPaid struct {
	common.BaseEvent

	TicketID   uuid.UUID `json:"ticketId"`
	Game       string    `json:"game"`
	PaidBy     string    `json:"paidBy"`
	PrizeCents int64     `json:"prizeCents"`
}

func (t TicketPaid) Type() string {
	return TicketPaidType
}
