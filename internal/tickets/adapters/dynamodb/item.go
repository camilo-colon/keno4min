package dynamodb

import (
	"fmt"
	"strconv"
	"time"
	"uuid"

	"github.com/cronos/keno4min/internal/tickets"
)

type ticketItem struct {
	PK        string    `dynamodbav:"PK"`
	SK        string    `dynamodbav:"SK"`
	GSI1PK    string    `dynamodbav:"GSI1PK"`
	GSI1SK    string    `dynamodbav:"GSI1SK"`
	GSI2PK    string    `dynamodbav:"GSI2PK"`
	TicketID  string    `dynamodbav:"ticketId"`
	Coupon    string    `dynamodbav:"coupon"`
	NodeID    string    `dynamodbav:"nodeId"`
	TIR       uint64    `dynamodbav:"tir"`
	Bets      []betItem `dynamodbav:"bets"`
	Status    string    `dynamodbav:"status"`
	CreatedBy string    `dynamodbav:"createdBy"`
	CreatedAt int64     `dynamodbav:"createdAt"`
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
		GSI1PK:    "TICKETS#TIR#" + strconv.FormatUint(ticket.TIR, 10),
		GSI1SK:    "TICKET#" + ticket.ID.String(),
		GSI2PK:    "TICKETS#NODE#" + ticket.NodeID.String(),
		TicketID:  ticket.ID.String(),
		Coupon:    ticket.Coupon,
		NodeID:    ticket.NodeID.String(),
		TIR:       ticket.TIR,
		Bets:      bets,
		Status:    string(ticket.Status),
		CreatedBy: ticket.CreatedBy,
		CreatedAt: ticket.CreatedAt.UnixMilli(),
	}
}

func fromItem(item ticketItem) (*tickets.Ticket, error) {
	id, err := uuid.Parse(item.TicketID)
	if err != nil {
		return nil, fmt.Errorf("decode ticket id %q: %w", item.TicketID, err)
	}
	nodeID, err := uuid.Parse(item.NodeID)
	if err != nil {
		return nil, fmt.Errorf("decode node id of ticket %s: %w", id, err)
	}

	bets := make([]tickets.Bet, len(item.Bets))
	for i, bet := range item.Bets {
		nums := make([]uint8, len(bet.Nums))
		for j, num := range bet.Nums {
			nums[j] = uint8(num)
		}
		bets[i] = tickets.Bet{Nums: nums, ValueCents: bet.ValueCents}
	}

	return &tickets.Ticket{
		ID:        id,
		Coupon:    item.Coupon,
		NodeID:    nodeID,
		TIR:       item.TIR,
		Bets:      bets,
		Status:    tickets.Status(item.Status),
		CreatedBy: item.CreatedBy,
		CreatedAt: time.UnixMilli(item.CreatedAt).UTC(),
	}, nil
}
