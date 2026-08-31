// Package draw contains the draw entity and its business rules.
package draw

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidID       = errors.New("draw ID must not be empty")
	ErrInvalidTIR      = errors.New("draw TIR must be greater than zero")
	ErrInvalidStartsAt = errors.New("draw startsAt must not be zero")
	ErrInvalidClosesAt = errors.New("draw closesAt must be after startsAt")
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
	ID       string
	TIR      uint64
	Status   Status
	StartsAt time.Time
	ClosesAt time.Time
	Balls    []uint8
	VideoID  string
}

// New creates the draw in its initial open state.
func New(id string, tir uint64, startsAt, closesAt time.Time) (*Draw, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrInvalidID
	}
	if tir == 0 {
		return nil, ErrInvalidTIR
	}
	if startsAt.IsZero() {
		return nil, ErrInvalidStartsAt
	}
	if !closesAt.After(startsAt) {
		return nil, ErrInvalidClosesAt
	}

	return &Draw{
		ID:       id,
		TIR:      tir,
		Status:   StatusOpen,
		StartsAt: startsAt,
		ClosesAt: closesAt,
	}, nil
}

// AcceptsTicketsAt reports whether at is inside the draw window.
func (d *Draw) AcceptsTicketsAt(at time.Time) bool {
	return d.Status == StatusOpen && !at.Before(d.StartsAt) && at.Before(d.ClosesAt)
}
