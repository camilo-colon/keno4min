package usecase

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/cronos/keno4min/internal/nodes"
	"github.com/cronos/keno4min/internal/nodes/ports"
)

var ErrParentNotFound = errors.New("parent node not found")

type Create struct {
	repository ports.Repository
}

func NewCreate(repository ports.Repository) *Create {
	return &Create{repository: repository}
}

type CreateParams struct {
	ID           uuid.UUID
	ParentId     *uuid.UUID
	Type         nodes.NodeType
	Name         string
	Address      string
	TimeZone     string
	Currency     string
	BalanceCents int
	Active       bool
}

func (c *Create) Execute(ctx context.Context, params CreateParams) error {
	existing, err := c.repository.FindById(ctx, params.ID)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	if params.ParentId != nil {
		parent, err := c.repository.FindById(ctx, *params.ParentId)
		if err != nil {
			return err
		}
		if parent == nil {
			return fmt.Errorf("%w: %s", ErrParentNotFound, *params.ParentId)
		}
	}
	node := nodes.New(params.ID, params.ParentId, params.Type, params.Name, params.Address, params.TimeZone, params.Currency, params.BalanceCents, params.Active)
	return c.repository.Save(ctx, node)
}
