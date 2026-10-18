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

type GetRuntimeResult struct {
	T2         int    `json:"t2"`
	TimeRound  int    `json:"time_round"`
	TimeWait   int    `json:"time_wait"`
	TimeBetOff int    `json:"time_betoff"`
	Idv        string `json:"idv"`
	Balls      []uint `json:"balls"`
}

func (g *GetRuntime) Execute(ctx context.Context) (*GetRuntimeResult, error) {
	runtime, err := g.repository.GetRuntime(ctx)
	if err != nil {
		return nil, err
	}

	remaining := time.Until(runtime.Current.ClosesAt)
	t2 := int(math.Ceil(remaining.Seconds()))

	result := &GetRuntimeResult{
		Idv:        "",
		Balls:      []uint{},
		T2:         t2,
		TimeRound:  157,
		TimeWait:   -23,
		TimeBetOff: -10,
	}

	return result, nil
}
