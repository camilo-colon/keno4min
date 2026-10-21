package usecase

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"uuid"

	"github.com/cronos/keno4min/internal/nodes"
)

type createRepository struct {
	find func(context.Context, uuid.UUID) (*nodes.Node, error)
	save func(context.Context, *nodes.Node) error
}

func (r createRepository) FindById(ctx context.Context, id uuid.UUID) (*nodes.Node, error) {
	return r.find(ctx, id)
}

func (r createRepository) Save(ctx context.Context, node *nodes.Node) error {
	return r.save(ctx, node)
}

func TestCreate(t *testing.T) {
	id, parentID := uuid.NewV7(), uuid.NewV7()
	failure := errors.New("repository unavailable")
	for _, tt := range []struct {
		name      string
		parentID  *uuid.UUID
		existing  bool
		parent    bool
		readError int
		saveError error
		wantError error
		wantReads []uuid.UUID
		wantSave  bool
	}{
		{name: "root", wantReads: []uuid.UUID{id}, wantSave: true},
		{name: "existing parent", parentID: &parentID, parent: true, wantReads: []uuid.UUID{id, parentID}, wantSave: true},
		{name: "missing parent", parentID: &parentID, wantReads: []uuid.UUID{id, parentID}, wantError: ErrParentNotFound},
		{name: "redelivery skips parent validation", parentID: &parentID, existing: true, wantReads: []uuid.UUID{id}},
		{name: "node lookup fails", readError: 1, wantReads: []uuid.UUID{id}, wantError: failure},
		{name: "parent lookup fails", parentID: &parentID, readError: 2, wantReads: []uuid.UUID{id, parentID}, wantError: failure},
		{name: "save fails", saveError: failure, wantReads: []uuid.UUID{id}, wantSave: true, wantError: failure},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			params := CreateParams{
				ID: id, ParentId: tt.parentID, Type: nodes.Cashier, Name: "cashier",
				Address: "address", TimeZone: "America/Bogota", Currency: "COP", BalanceCents: 125, Active: true,
			}
			var reads []uuid.UUID
			var saved *nodes.Node
			repository := createRepository{
				find: func(gotCtx context.Context, gotID uuid.UUID) (*nodes.Node, error) {
					if gotCtx != ctx {
						t.Fatal("lookup did not receive caller context")
					}
					reads = append(reads, gotID)
					if tt.readError == len(reads) {
						return nil, failure
					}
					if (gotID == id && tt.existing) || (gotID == parentID && tt.parent) {
						return &nodes.Node{ID: gotID}, nil
					}
					return nil, nil
				},
				save: func(gotCtx context.Context, node *nodes.Node) error {
					if gotCtx != ctx {
						t.Fatal("save did not receive caller context")
					}
					saved = node
					return tt.saveError
				},
			}
			if err := NewCreate(repository).Execute(ctx, params); !errors.Is(err, tt.wantError) {
				t.Fatalf("error = %v, want %v", err, tt.wantError)
			}
			if !reflect.DeepEqual(reads, tt.wantReads) {
				t.Fatalf("lookups = %v, want %v", reads, tt.wantReads)
			}
			if (saved != nil) != tt.wantSave {
				t.Fatalf("saved = %v, want save %v", saved, tt.wantSave)
			}
			if tt.wantSave {
				want := &nodes.Node{ID: id, ParentId: tt.parentID, Type: nodes.Cashier, Name: "cashier", Address: "address", TimeZone: "America/Bogota", Currency: "COP", BalanceCents: 125, Active: true}
				if !reflect.DeepEqual(saved, want) {
					t.Fatalf("saved = %+v, want %+v", saved, want)
				}
			}
		})
	}
}
