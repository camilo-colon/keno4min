package draw_test

import (
	"errors"
	"testing"
	"time"

	"github.com/cronos/keno4min/internal/draw"
)

var (
	startsAt = time.Date(2026, time.August, 30, 20, 0, 0, 0, time.UTC)
	closesAt = startsAt.Add(157 * time.Second)
)

func TestNewCreatesOpenDraw(t *testing.T) {
	d, err := draw.New("draw-3573", 3573, startsAt, closesAt)
	if err != nil {
		t.Fatalf("new draw: %v", err)
	}

	if d.ID != "draw-3573" || d.TIR != 3573 || d.Status != draw.StatusOpen ||
		!d.StartsAt.Equal(startsAt) || !d.ClosesAt.Equal(closesAt) {
		t.Fatalf("unexpected draw: %+v", d)
	}
	if len(d.Balls) != 0 || d.VideoID != "" {
		t.Fatalf("new draw has a result: %+v", d)
	}
}

func TestNewRejectsInvalidData(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		tir      uint64
		startsAt time.Time
		closesAt time.Time
		wantErr  error
	}{
		{name: "empty ID", id: " ", tir: 1, startsAt: startsAt, closesAt: closesAt, wantErr: draw.ErrInvalidID},
		{name: "zero TIR", id: "draw-1", startsAt: startsAt, closesAt: closesAt, wantErr: draw.ErrInvalidTIR},
		{name: "zero startsAt", id: "draw-1", tir: 1, closesAt: closesAt, wantErr: draw.ErrInvalidStartsAt},
		{name: "zero closesAt", id: "draw-1", tir: 1, startsAt: startsAt, wantErr: draw.ErrInvalidClosesAt},
		{name: "closesAt equals startsAt", id: "draw-1", tir: 1, startsAt: startsAt, closesAt: startsAt, wantErr: draw.ErrInvalidClosesAt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := draw.New(tt.id, tt.tir, tt.startsAt, tt.closesAt)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestDrawWindow(t *testing.T) {
	d := mustNew(t)

	if d.AcceptsTicketsAt(startsAt.Add(-time.Nanosecond)) {
		t.Fatal("accepted a ticket before startsAt")
	}
	if !d.AcceptsTicketsAt(startsAt) || !d.AcceptsTicketsAt(closesAt.Add(-time.Nanosecond)) {
		t.Fatal("rejected a ticket inside the draw window")
	}
	if d.AcceptsTicketsAt(closesAt) {
		t.Fatal("accepted a ticket at closesAt")
	}
}

func mustNew(t *testing.T) *draw.Draw {
	t.Helper()
	d, err := draw.New("draw-3573", 3573, startsAt, closesAt)
	if err != nil {
		t.Fatalf("new draw: %v", err)
	}
	return d
}
