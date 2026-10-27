package draw

import (
	"errors"
	"time"
	"uuid"
)

var (
	ErrInvalidID       = errors.New("draw ID must not be empty")
	ErrInvalidTIR      = errors.New("draw TIR must be greater than zero")
	ErrInvalidStartsAt = errors.New("draw startsAt must not be zero")
	ErrInvalidClosesAt = errors.New("draw closesAt must be after startsAt")
	ErrInvalidStatus   = errors.New("unrecognized draw status")
	ErrInvalidBalls    = errors.New("balls must match the draw status and be 20 distinct integers between 1 and 80 when drawn")
)

// Status describes the lifecycle of a draw.
type Status string

const (
	StatusOpen   Status = "open"
	StatusClosed Status = "closed"
	StatusDrawn  Status = "drawn"
)

// Draw is the state shared by the API and the draw workflow.
type Draw struct {
	ID       uuid.UUID
	TIR      uint64
	Status   Status
	StartsAt time.Time
	ClosesAt time.Time
	Balls    []int
	VideoID  string
}

// New creates the draw in its initial open state.
func New(id uuid.UUID, tir uint64, startsAt, closesAt time.Time) (*Draw, error) {
	d := &Draw{ID: id, TIR: tir, Status: StatusOpen, StartsAt: startsAt, ClosesAt: closesAt}
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return d, nil
}

// Validate checks the invariants of a new or restored draw without changing it.
func (d *Draw) Validate() error {
	if d.TIR == 0 {
		return ErrInvalidTIR
	}
	if d.StartsAt.IsZero() {
		return ErrInvalidStartsAt
	}
	if !d.ClosesAt.After(d.StartsAt) {
		return ErrInvalidClosesAt
	}
	if d.Status != StatusOpen && d.Status != StatusClosed && d.Status != StatusDrawn {
		return ErrInvalidStatus
	}
	if (d.Status == StatusDrawn && len(d.Balls) != 20) ||
		(d.Status != StatusDrawn && len(d.Balls) != 0) {
		return ErrInvalidBalls
	}
	seen := [81]bool{}
	for _, ball := range d.Balls {
		if ball < 1 || ball > 80 || seen[ball] {
			return ErrInvalidBalls
		}
		seen[ball] = true
	}
	return nil
}

// AcceptsTicketsAt reports whether at is inside the draw window.
func (d *Draw) AcceptsTicketsAt(at time.Time) bool {
	return d.Status == StatusOpen && !at.Before(d.StartsAt) && at.Before(d.ClosesAt)
}
