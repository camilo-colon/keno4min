package usecase

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	draw "github.com/cronos/keno4min/internal/draws"
	drawports "github.com/cronos/keno4min/internal/draws/ports"
	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/users"
	userports "github.com/cronos/keno4min/internal/users/ports"
)

type createRepository struct {
	generate func(context.Context) (string, error)
	save     func(context.Context, *tickets.Ticket) error
}

func (r createRepository) GenerateCoupon(ctx context.Context) (string, error) {
	return r.generate(ctx)
}

func (r createRepository) Save(ctx context.Context, ticket *tickets.Ticket) error {
	return r.save(ctx, ticket)
}

type userRepository struct {
	userports.Repository
	find func(context.Context, uuid.UUID) (*users.User, error)
}

func (r userRepository) FindById(ctx context.Context, id uuid.UUID) (*users.User, error) {
	return r.find(ctx, id)
}

type drawRepository struct {
	drawports.Repository
	current func(context.Context) (*draw.Draw, error)
}

func (r drawRepository) GetCurrent(ctx context.Context) (*draw.Draw, error) {
	return r.current(ctx)
}

var testDrawID = uuid.MustParse("019a052b-3000-7000-8000-000000000003")

func openDraw() *draw.Draw {
	return &draw.Draw{ID: testDrawID, TIR: 3573, Status: draw.StatusOpen,
		StartsAt: time.Now().Add(-time.Minute), ClosesAt: time.Now().Add(time.Minute)}
}

func TestCreate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	user := &users.User{ID: uuid.NewV7(), NodeId: uuid.NewV7(), Username: "SellerName"}
	params := CreateParams{
		UserID: user.ID,
		Bets: []tickets.Bet{
			{Nums: []uint8{1, 80}, ValueCents: 50000},
			{Nums: []uint8{1, 80}, ValueCents: 50000},
		},
	}
	var calls []string
	var saved *tickets.Ticket
	userRepo := userRepository{find: func(gotCtx context.Context, id uuid.UUID) (*users.User, error) {
		if gotCtx != ctx || id != params.UserID {
			t.Fatal("user lookup did not receive caller context and userId")
		}
		calls = append(calls, "find")
		return user, nil
	}}
	repository := createRepository{
		generate: func(gotCtx context.Context) (string, error) {
			if gotCtx != ctx {
				t.Fatal("coupon generation did not receive caller context")
			}
			calls = append(calls, "generate")
			return "0007", nil
		},
		save: func(gotCtx context.Context, ticket *tickets.Ticket) error {
			if gotCtx != ctx {
				t.Fatal("save did not receive caller context")
			}
			calls = append(calls, "save")
			saved = ticket
			return nil
		},
	}

	drawRepo := drawRepository{current: func(gotCtx context.Context) (*draw.Draw, error) {
		if gotCtx != ctx {
			t.Fatal("current draw lookup did not receive caller context")
		}
		calls = append(calls, "current")
		return openDraw(), nil
	}}
	before := time.Now().UTC()
	ticket, err := NewCreate(repository, userRepo, drawRepo).Execute(ctx, params)
	after := time.Now().UTC()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls, []string{"find", "current", "generate", "save"}) {
		t.Fatalf("calls = %v", calls)
	}
	if ticket == nil || ticket != saved {
		t.Fatal("result must be the saved ticket")
	}
	if ticket.ID == uuid.Nil() || ticket.ID.String()[14] != '7' {
		t.Fatalf("ticket id = %s, want a generated UUID v7", ticket.ID)
	}
	if ticket.TIR != 3573 || ticket.Coupon != "0007" ||
		ticket.NodeID != user.NodeId || ticket.CreatedBy != user.Username || ticket.Status != tickets.StatusPending {
		t.Fatalf("unexpected ticket: %+v", ticket)
	}
	if !reflect.DeepEqual(ticket.Bets, params.Bets) {
		t.Fatalf("bets = %+v, want %+v", ticket.Bets, params.Bets)
	}
	if ticket.CreatedAt.Location() != time.UTC || ticket.CreatedAt.Before(before) || ticket.CreatedAt.After(after) {
		t.Fatalf("createdAt = %v, want current UTC time", ticket.CreatedAt)
	}
	next, err := NewCreate(repository, userRepo, drawRepo).Execute(ctx, params)
	if err != nil {
		t.Fatal(err)
	}
	if next.ID == ticket.ID {
		t.Fatalf("separate creations received the same ticket id: %s", ticket.ID)
	}
}

func TestCreateStopsOnError(t *testing.T) {
	failure := errors.New("repository unavailable")
	for _, tt := range []struct {
		name           string
		change         func(*CreateParams, *users.User)
		missing        bool
		findError      error
		currentError   error
		missingCurrent bool
		coupon         string
		couponError    error
		saveError      error
		wantError      error
		wantCalls      []string
	}{
		{name: "user lookup fails", findError: failure, wantError: failure, wantCalls: []string{"find"}},
		{name: "user lookup cancelled", findError: context.Canceled, wantError: context.Canceled, wantCalls: []string{"find"}},
		{name: "user not found", missing: true, wantError: ErrUserNotFound, wantCalls: []string{"find"}},
		{name: "coupon generation fails", couponError: failure, wantError: failure, wantCalls: []string{"find", "current", "generate"}},
		{name: "empty coupon", wantCalls: []string{"find", "current", "generate"}},
		{name: "invalid bets", coupon: "0001", change: func(p *CreateParams, _ *users.User) { p.Bets[0].Nums = []uint8{1, 1} }, wantCalls: []string{"find", "current", "generate"}},
		{name: "current draw lookup fails", currentError: failure, wantError: ErrDrawUnavailable, wantCalls: []string{"find", "current"}},
		{name: "missing current draw", missingCurrent: true, wantError: ErrDrawUnavailable, wantCalls: []string{"find", "current"}},
		{name: "missing club", coupon: "0001", change: func(_ *CreateParams, u *users.User) { u.NodeId = uuid.Nil() }, wantCalls: []string{"find", "current", "generate"}},
		{name: "blank username", coupon: "0001", change: func(_ *CreateParams, u *users.User) { u.Username = "  " }, wantCalls: []string{"find", "current", "generate"}},
		{name: "save fails", coupon: "0001", saveError: failure, wantError: failure, wantCalls: []string{"find", "current", "generate", "save"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			user := &users.User{ID: uuid.NewV7(), NodeId: uuid.NewV7(), Username: "seller"}
			params := CreateParams{
				UserID: user.ID,
				Bets:   []tickets.Bet{{Nums: []uint8{1}, ValueCents: 100}},
			}
			if tt.change != nil {
				tt.change(&params, user)
			}
			var calls []string
			userRepo := userRepository{find: func(context.Context, uuid.UUID) (*users.User, error) {
				calls = append(calls, "find")
				if tt.missing {
					return nil, nil
				}
				return user, tt.findError
			}}
			drawRepo := drawRepository{current: func(context.Context) (*draw.Draw, error) {
				calls = append(calls, "current")
				if tt.currentError != nil {
					return nil, tt.currentError
				}
				if tt.missingCurrent {
					return nil, nil
				}
				return openDraw(), nil
			}}
			repository := createRepository{
				generate: func(context.Context) (string, error) {
					calls = append(calls, "generate")
					return tt.coupon, tt.couponError
				},
				save: func(context.Context, *tickets.Ticket) error {
					calls = append(calls, "save")
					return tt.saveError
				},
			}
			ticket, err := NewCreate(repository, userRepo, drawRepo).Execute(context.Background(), params)
			if err == nil || ticket != nil {
				t.Fatalf("Execute returned (%+v, %v), want (nil, error)", ticket, err)
			}
			if tt.wantError != nil && !errors.Is(err, tt.wantError) {
				t.Fatalf("error = %v, want %v", err, tt.wantError)
			}
			if !reflect.DeepEqual(calls, tt.wantCalls) {
				t.Fatalf("calls = %v, want %v", calls, tt.wantCalls)
			}
		})
	}
}
