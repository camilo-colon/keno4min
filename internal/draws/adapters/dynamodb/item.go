package dynamodb

import (
	"fmt"
	"time"
	"uuid"

	draw "github.com/cronos/keno4min/internal/draws"
	"github.com/cronos/keno4min/internal/draws/ports"
)

// runtimeItem is the materialized view read from DRAW_RUNTIME/META.
type runtimeItem struct {
	Current      currentDrawItem   `dynamodbav:"current"`
	LatestResult *latestResultItem `dynamodbav:"latestResult"`
}

type currentDrawItem struct {
	ID       string    `dynamodbav:"id"`
	Tir      uint64    `dynamodbav:"tir"`
	Status   string    `dynamodbav:"status"`
	StartsAt time.Time `dynamodbav:"startsAt"`
	ClosesAt time.Time `dynamodbav:"closesAt"`
}

type latestResultItem struct {
	DrawID  string `dynamodbav:"drawId"`
	Tir     uint64 `dynamodbav:"tir"`
	Balls   []int  `dynamodbav:"balls"`
	VideoID string `dynamodbav:"videoId"`
}

// tirLookupItem is the DRAW_TIR#<tir>/META item that points to the canonical draw.
type tirLookupItem struct {
	DrawID string `dynamodbav:"id"`
}

type drawItem struct {
	ID       string      `dynamodbav:"id"`
	Balls    []int       `dynamodbav:"balls"`
	Tir      uint64      `dynamodbav:"tir"`
	VideoID  string      `dynamodbav:"videoId"`
	Status   draw.Status `dynamodbav:"status"`
	StartsAt time.Time   `dynamodbav:"startsAt"`
	ClosesAt time.Time   `dynamodbav:"closesAt"`
}

func fromDrawItem(item drawItem) (draw.Draw, error) {
	id, err := uuid.Parse(item.ID)
	if err != nil {
		return draw.Draw{}, fmt.Errorf("decode latest draw ID: %w", err)
	}
	for _, ball := range item.Balls {
		if ball < 1 || ball > 80 {
			return draw.Draw{}, fmt.Errorf("decode latest draw balls: %w", draw.ErrInvalidBalls)
		}
	}
	return draw.Draw{
		ID:       id,
		TIR:      item.Tir,
		Status:   item.Status,
		StartsAt: item.StartsAt,
		ClosesAt: item.ClosesAt,
		Balls:    item.Balls,
		VideoID:  item.VideoID,
	}, nil
}

func fromRuntimeItem(item runtimeItem) (*ports.Runtime, error) {
	id, err := uuid.Parse(item.Current.ID)
	if err != nil {
		return nil, fmt.Errorf("decode runtime current.id: %w", err)
	}

	runtime := &ports.Runtime{
		Current: ports.CurrentDraw{
			DrawId:   id,
			Tir:      item.Current.Tir,
			StartsAt: item.Current.StartsAt,
			ClosesAt: item.Current.ClosesAt,
		},
	}
	if item.LatestResult != nil {
		resultID, err := uuid.Parse(item.LatestResult.DrawID)
		if err != nil {
			return nil, fmt.Errorf("decode runtime latestResult.drawId: %w", err)
		}
		runtime.LatestResult = &ports.LatestResult{
			DrawId: resultID,
			Tir:    item.LatestResult.Tir,
			Balls:  item.LatestResult.Balls,
			Idv:    item.LatestResult.VideoID,
		}
	}
	return runtime, nil
}
