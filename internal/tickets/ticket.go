package tickets

import (
	"errors"
	"math"
	"slices"
	"strings"
	"time"
	"uuid"

	"github.com/cronos/keno4min/internal/common"
	"github.com/cronos/keno4min/internal/tickets/events"
)

// Game identifies this product in the events consumed by other systems.
const Game = "keno4min"

var (
	ErrBettingClosed          = errors.New("draw is not accepting tickets")
	ErrTicketAlreadyCancelled = errors.New("ticket is already cancelled")
	ErrTicketNotCancellable   = errors.New("ticket can no longer be cancelled")
	ErrTicketAlreadyPaid      = errors.New("ticket is already paid")
	ErrTicketNotPayable       = errors.New("ticket has no prize to pay")
)

type Bet struct {
	Nums       []uint
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
	common.AggregateRoot

	ID         uuid.UUID
	Coupon     string
	NodeID     uuid.UUID
	TIR        uint64
	Bets       []Bet
	Status     Status
	PrizeCents int64
	Balls      []uint8
	CreatedBy  string
	CreatedAt  time.Time
	// PaidBy and PaidAt record the cashier who closed the ticket and when, for both
	// cancellation and payment. Only Status tells which one happened: a cancelled
	// ticket also fills them. Both are empty while the ticket is open.
	PaidBy string
	PaidAt time.Time
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
	ticket.recordCreated()
	return ticket, nil
}

// TotalCents assumes validated bets, whose sum cannot overflow.
func (t *Ticket) TotalCents() int64 {
	var total int64
	for _, bet := range t.Bets {
		total += bet.ValueCents
	}
	return total
}

// Cancel moves a pending ticket to cancelled and records who did it.
// The draw window is checked by the caller, since the ticket only knows its TIR.
func (t *Ticket) Cancel(cancelledBy string, at time.Time) error {
	if strings.TrimSpace(cancelledBy) == "" {
		return errors.New("cancelled by is required")
	}
	if at.IsZero() {
		return errors.New("cancellation time is required")
	}
	switch t.Status {
	case StatusPending:
	case StatusCancelled:
		return ErrTicketAlreadyCancelled
	default:
		return ErrTicketNotCancellable
	}

	t.Status = StatusCancelled
	t.PaidBy = cancelledBy
	t.PaidAt = at.UTC()
	t.Record(events.TicketCancelled{
		BaseEvent:   common.NewBaseEvent(t.PaidAt),
		TicketID:    t.ID,
		Game:        Game,
		CancelledBy: cancelledBy,
		ValueCents:  t.TotalCents(),
	})
	return nil
}

// Pay moves a winning ticket to paid and records who handed over the prize.
// Ownership is checked by the caller; the jackpot is not part of the prize yet.
func (t *Ticket) Pay(paidBy string, at time.Time) error {
	if strings.TrimSpace(paidBy) == "" {
		return errors.New("paid by is required")
	}
	if at.IsZero() {
		return errors.New("payment time is required")
	}
	switch t.Status {
	case StatusWinning:
	case StatusPaid:
		return ErrTicketAlreadyPaid
	default:
		return ErrTicketNotPayable
	}
	// A winning ticket always carries a prize; a zero one means corrupt settlement data.
	if t.PrizeCents <= 0 {
		return ErrTicketNotPayable
	}

	t.Status = StatusPaid
	t.PaidBy = paidBy
	t.PaidAt = at.UTC()
	t.Record(events.TicketPaid{
		BaseEvent:  common.NewBaseEvent(t.PaidAt),
		TicketID:   t.ID,
		Game:       Game,
		PaidBy:     paidBy,
		PrizeCents: t.PrizeCents,
	})
	return nil
}

func (t *Ticket) recordCreated() {
	bets := make([]events.Bet, len(t.Bets))
	for i, bet := range t.Bets {
		bets[i] = events.Bet{Numbers: slices.Clone(bet.Nums), ValueCents: bet.ValueCents}
	}
	t.Record(events.TicketCreated{
		BaseEvent:  common.NewBaseEvent(t.CreatedAt),
		TicketID:   t.ID,
		Game:       Game,
		Coupon:     t.Coupon,
		NodeID:     t.NodeID,
		CreatedBy:  t.CreatedBy,
		ValueCents: t.TotalCents(),
		Details:    events.TicketCreatedDetails{TIR: t.TIR, Bets: bets},
	})
}

func (t *Ticket) validate() error {
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
	if t.PrizeCents < 0 {
		return errors.New("prize must not be negative")
	}
	return validateBets(t.Bets)
}

// validateBets checks selections, values and total overflow before a sale is processed.
func validateBets(bets []Bet) error {
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
