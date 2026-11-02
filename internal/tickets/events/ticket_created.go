package events

import (
	"uuid"

	"github.com/cronos/keno4min/internal/common"
)

var _ common.DomainEvent = (*TicketCreated)(nil)

const TicketCreatedType = "cronos.keno4min.ticket.created"

// TicketCreated does not import tickets, so the aggregate can record it without a cycle.
type TicketCreated struct {
	common.BaseEvent

	TicketID   uuid.UUID `json:"ticketId"`
	Coupon     string    `json:"coupon"`
	NodeID     uuid.UUID `json:"nodeId"`
	TIR        uint64    `json:"tir"`
	Bets       []Bet     `json:"bets"`
	TotalCents int64     `json:"totalCents"`
	CreatedBy  string    `json:"createdBy"`
}

// Bet uses []uint so numbers are encoded as a JSON array, not as the base64 of []uint8.
type Bet struct {
	Nums       []uint `json:"nums"`
	ValueCents int64  `json:"valueCents"`
}

func (t TicketCreated) Type() string {
	return TicketCreatedType
}
