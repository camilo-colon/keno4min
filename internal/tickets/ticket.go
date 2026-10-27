package tickets

import (
	"errors"
	"math"
	"strings"
	"time"
	"uuid"
)

var ErrBettingClosed = errors.New("draw is not accepting tickets")

type Bet struct {
	Nums       []uint8
	ValueCents int64
}

type Status string

const (
	StatusPending   Status = "pending"
	StatusCancelled Status = "cancelled"
	StatusLost      Status = "lost"
	StatusWinning   Status = "winning"
	StatusPaid      Status = "paid"
)

type Ticket struct {
	ID        uuid.UUID
	Coupon    string
	NodeID    uuid.UUID
	TIR       uint64
	Bets      []Bet
	Status    Status
	CreatedBy string
	CreatedAt time.Time
}

func Create(id uuid.UUID, coupon string, nodeID uuid.UUID, tir uint64, bets []Bet, createdBy string) (*Ticket, error) {
	ticket := &Ticket{
		ID:        id,
		Coupon:    coupon,
		NodeID:    nodeID,
		TIR:       tir,
		Bets:      bets,
		Status:    StatusPending,
		CreatedBy: createdBy,
		CreatedAt: time.Now().UTC(),
	}
	if err := ticket.validate(); err != nil {
		return nil, err
	}
	return ticket, nil
}

func (t Ticket) validate() error {
	if t.ID == uuid.Nil() {
		return errors.New("ticket id is required")
	}
	if t.NodeID == uuid.Nil() {
		return errors.New("node id is required")
	}
	if t.TIR == 0 {
		return errors.New("tir must be greater than zero")
	}
	if strings.TrimSpace(t.CreatedBy) == "" {
		return errors.New("created by is required")
	}
	if t.Coupon == "" {
		return errors.New("coupon is required")
	}
	return ValidateBets(t.Bets)
}

// ValidateBets checks selections, values and total overflow before a sale is processed.
func ValidateBets(bets []Bet) error {
	if len(bets) == 0 {
		return errors.New("ticket must contain at least one bet")
	}

	var total int64
	for _, bet := range bets {
		if len(bet.Nums) == 0 || len(bet.Nums) > 10 {
			return errors.New("bet must contain between 1 and 10 numbers")
		}
		if bet.ValueCents <= 0 {
			return errors.New("bet value must be greater than zero")
		}

		seen := [81]bool{}
		for _, num := range bet.Nums {
			if num < 1 || num > 80 {
				return errors.New("bet numbers must be between 1 and 80")
			}
			if seen[num] {
				return errors.New("bet numbers must not repeat")
			}
			seen[num] = true
		}

		if bet.ValueCents > math.MaxInt64-total {
			return errors.New("ticket total exceeds the supported amount")
		}
		total += bet.ValueCents
	}

	return nil
}
