package ports

import (
	"context"
	"time"
	"uuid"
)

type Runtime struct {
	Current      CurrentDraw
	LatestResult *LatestResult
}

type CurrentDraw struct {
	DrawId   uuid.UUID
	Tir      uint64
	StartsAt time.Time
	ClosesAt time.Time
}

type LatestResult struct {
	DrawId uuid.UUID
	Tir    uint64
	Balls  []int8
	Idv    string
}

type Repository interface {
	GetRuntime(context.Context) (*Runtime, error)
}
