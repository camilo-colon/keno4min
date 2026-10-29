package dynamodb

import (
	"fmt"
	"strconv"
	"time"
	"uuid"

	"github.com/cronos/keno4min/internal/tickets"
)

type ticketItem struct {
	PK         string    `dynamodbav:"PK"`
	SK         string    `dynamodbav:"SK"`
	GSI1PK     string    `dynamodbav:"GSI1PK"`
	GSI1SK     string    `dynamodbav:"GSI1SK"`
	GSI2PK     string    `dynamodbav:"GSI2PK"`
	GSI2SK     string    `dynamodbav:"GSI2SK"`
	TicketID   string    `dynamodbav:"ticketId"`
	Coupon     string    `dynamodbav:"coupon"`
	NodeID     string    `dynamodbav:"nodeId"`
	TIR        uint64    `dynamodbav:"tir"`
	Bets       []betItem `dynamodbav:"bets"`
	Status     string    `dynamodbav:"status"`
	PrizeCents int64     `dynamodbav:"prizeCents"`
	CreatedBy  string    `dynamodbav:"createdBy"`
	CreatedAt  string    `dynamodbav:"createdAt"`
}

// createdAtLayout fixes the millisecond width so lexical order matches chronological order.
const createdAtLayout = "2006-01-02T15:04:05.000Z07:00"

func formatCreatedAt(t time.Time) string {
	return t.UTC().Format(createdAtLayout)
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

	createdAt := formatCreatedAt(ticket.CreatedAt)
	return ticketItem{
		PK:         "TICKET#" + ticket.ID.String(),
		SK:         "META",
		GSI1PK:     "TICKETS#TIR#" + strconv.FormatUint(ticket.TIR, 10),
		GSI1SK:     "TICKET#" + ticket.ID.String(),
		GSI2PK:     "TICKETS#NODE#" + ticket.NodeID.String(),
		GSI2SK:     createdAt,
		TicketID:   ticket.ID.String(),
		Coupon:     ticket.Coupon,
		NodeID:     ticket.NodeID.String(),
		TIR:        ticket.TIR,
		Bets:       bets,
		Status:     string(ticket.Status),
		PrizeCents: ticket.PrizeCents,
		CreatedBy:  ticket.CreatedBy,
		CreatedAt:  createdAt,
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

	createdAt, err := time.Parse(time.RFC3339Nano, item.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("decode creation date of ticket %s: %w", id, err)
	}

	return &tickets.Ticket{
		ID:         id,
		Coupon:     item.Coupon,
		NodeID:     nodeID,
		TIR:        item.TIR,
		Bets:       bets,
		Status:     tickets.Status(item.Status),
		PrizeCents: item.PrizeCents,
		CreatedBy:  item.CreatedBy,
		CreatedAt:  createdAt.UTC(),
	}, nil
}
