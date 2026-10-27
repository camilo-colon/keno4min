package dynamodb

import (
	"time"

	"github.com/cronos/keno4min/internal/tickets"
)

type ticketItem struct {
	PK        string    `dynamodbav:"PK"`
	SK        string    `dynamodbav:"SK"`
	TicketID  string    `dynamodbav:"ticketId"`
	Coupon    string    `dynamodbav:"coupon"`
	NodeID    string    `dynamodbav:"nodeId"`
	TIR       uint64    `dynamodbav:"tir"`
	Bets      []betItem `dynamodbav:"bets"`
	Status    string    `dynamodbav:"status"`
	CreatedBy string    `dynamodbav:"createdBy"`
	CreatedAt string    `dynamodbav:"createdAt"`
}

type betItem struct {
	Nums       []uint `dynamodbav:"nums"`
	ValueCents int64  `dynamodbav:"valueCents"`
}

type couponItem struct {
	PK       string `dynamodbav:"PK"`
	SK       string `dynamodbav:"SK"`
	TicketID string `dynamodbav:"ticketId"`
}

func toItem(ticket *tickets.Ticket) ticketItem {
	bets := make([]betItem, len(ticket.Bets))
	for i, bet := range ticket.Bets {
		nums := make([]uint, len(bet.Nums))
		for j, num := range bet.Nums {
			nums[j] = uint(num)
		}
		bets[i] = betItem{Nums: nums, ValueCents: bet.ValueCents}
	}

	return ticketItem{
		PK:        "TICKET#" + ticket.ID.String(),
		SK:        "META",
		TicketID:  ticket.ID.String(),
		Coupon:    ticket.Coupon,
		NodeID:    ticket.NodeID.String(),
		TIR:       ticket.TIR,
		Bets:      bets,
		Status:    string(ticket.Status),
		CreatedBy: ticket.CreatedBy,
		CreatedAt: ticket.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}
