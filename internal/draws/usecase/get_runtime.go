package usecase

import (
	"context"
	"math"
	"time"

	"github.com/cronos/keno4min/internal/draws/ports"
)

type GetRuntime struct {
	repository ports.Repository
}

func NewGetRuntime(repository ports.Repository) *GetRuntime {
	return &GetRuntime{repository: repository}
}

type HistoryResult struct {
	Tir   uint64 `json:"tir"`
	Balls []int  `json:"balls"`
}

type GetRuntimeResult struct {
	T2         int             `json:"t2"`
	Tir        uint64          `json:"tir"`
	TimeRound  int             `json:"time_round"`
	TimeWait   int             `json:"time_wait"`
	TimeBetOff int             `json:"time_betoff"`
	Idv        string          `json:"idv"`
	Balls      []int           `json:"balls"`
	Jp1        int             `json:"jp1"`
	History    []HistoryResult `json:"history"`
}

func (g *GetRuntime) Execute(ctx context.Context) (*GetRuntimeResult, error) {
	runtime, err := g.repository.GetRuntime(ctx)
	if err != nil {
		return nil, err
	}

	draws, err := g.repository.GetLatestDraws(ctx)
	if err != nil {
		return nil, err
	}

	remaining := time.Until(runtime.Current.ClosesAt)
	t2 := int(math.Ceil(remaining.Seconds()))

	history := make([]HistoryResult, 0, len(draws))
	for _, draw := range draws {
		history = append(history, HistoryResult{
			Tir:   draw.TIR,
			Balls: draw.Balls,
		})
	}

	result := &GetRuntimeResult{
		T2:         t2,
		Tir:        runtime.Current.Tir,
		Balls:      []int{},
		TimeRound:  157,
		TimeWait:   -23,
		TimeBetOff: -10,
		Jp1:        10000_00,
		History:    history,
	}

	if t2 > 0 && runtime.LatestResult != nil {
		result.Tir = runtime.LatestResult.Tir
	}

	if last := runtime.LatestResult; last != nil {
		result.Balls = last.Balls
		result.Idv = last.Idv
	}

	return result, nil
}
