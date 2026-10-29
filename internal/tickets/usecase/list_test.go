package usecase

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
)

type listRepository struct {
	ports.Repository
	list func(context.Context, ports.ListParams) (ports.TicketPage, error)
}

func (r listRepository) ListByNode(ctx context.Context, params ports.ListParams) (ports.TicketPage, error) {
	return r.list(ctx, params)
}

func validListParams() ports.ListParams {
	return ports.ListParams{
		NodeID: uuid.NewV7(),
		From:   time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC),
		To:     time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC),
		Limit:  2, Cursor: "next-page",
	}
}

func TestListReturnsOnePage(t *testing.T) {
	params := validListParams()
	want := ports.TicketPage{Items: []*tickets.Ticket{{Coupon: "0007"}}, NextCursor: "another-page"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	repo := listRepository{list: func(gotCtx context.Context, got ports.ListParams) (ports.TicketPage, error) {
		calls++
		if gotCtx != ctx || !reflect.DeepEqual(got, params) {
			t.Fatalf("unexpected query: %+v", got)
		}
		return want, nil
	}}
	got, err := NewList(repo).Execute(ctx, params)
	if err != nil || !reflect.DeepEqual(got, want) || calls != 1 {
		t.Fatalf("got (%+v, %v), calls=%d; want one page", got, err, calls)
	}
}

func TestListDefaultsToFiftyTickets(t *testing.T) {
	params := validListParams()
	params.Limit = 0
	repo := listRepository{list: func(_ context.Context, got ports.ListParams) (ports.TicketPage, error) {
		if got.Limit != 50 {
			t.Fatalf("limit = %d, want 50", got.Limit)
		}
		return ports.TicketPage{}, nil
	}}
	if _, err := NewList(repo).Execute(context.Background(), params); err != nil {
		t.Fatal(err)
	}
}

func TestListRejectsInvalidFiltersBeforeReading(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*ports.ListParams)
	}{
		{"missing node", func(p *ports.ListParams) { p.NodeID = uuid.Nil() }},
		{"missing from", func(p *ports.ListParams) { p.From = time.Time{} }},
		{"missing to", func(p *ports.ListParams) { p.To = time.Time{} }},
		{"equal dates", func(p *ports.ListParams) { p.To = p.From }},
		{"reversed dates", func(p *ports.ListParams) { p.From, p.To = p.To, p.From }},
		{"submillisecond from", func(p *ports.ListParams) { p.From = p.From.Add(time.Nanosecond) }},
		{"submillisecond to", func(p *ports.ListParams) { p.To = p.To.Add(time.Nanosecond) }},
		{"negative limit", func(p *ports.ListParams) { p.Limit = -1 }},
		{"too many tickets", func(p *ports.ListParams) { p.Limit = 101 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			params := validListParams()
			tc.change(&params)
			repo := listRepository{list: func(context.Context, ports.ListParams) (ports.TicketPage, error) {
				t.Fatal("invalid filters must not reach repository")
				return ports.TicketPage{}, nil
			}}
			if _, err := NewList(repo).Execute(context.Background(), params); !errors.Is(err, ErrInvalidListParams) {
				t.Fatalf("error = %v, want invalid filters", err)
			}
		})
	}
}

func TestListPropagatesRepositoryError(t *testing.T) {
	failure := errors.New("database unavailable")
	repo := listRepository{list: func(context.Context, ports.ListParams) (ports.TicketPage, error) {
		return ports.TicketPage{}, failure
	}}
	if _, err := NewList(repo).Execute(context.Background(), validListParams()); !errors.Is(err, failure) {
		t.Fatalf("error = %v, want original error", err)
	}
}
