package ports

import (
	"context"
	"time"
	"uuid"

	"github.com/cronos/keno4min/internal/draws"
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
	Balls  []int
	Idv    string
}

type Repository interface {
	GetCurrent(context.Context) (*draw.Draw, error)
	FindByID(context.Context, uuid.UUID) (*draw.Draw, error)
	FindByTIR(context.Context, uint64) (*draw.Draw, error)
	GetRuntime(context.Context) (*Runtime, error)
	GetLatestDraws(context.Context) ([]draw.Draw, error)
}
