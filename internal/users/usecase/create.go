package usecase

import (
	"context"
	"errors"
	"uuid"

	"github.com/cronos/keno4min/internal/users"
	"github.com/cronos/keno4min/internal/users/ports"
)

var ErrInvalidUser = errors.New("userId, username and nodeId are required")

type Create struct {
	repository ports.Repository
}

func NewCreate(repository ports.Repository) *Create {
	return &Create{repository: repository}
}

type CreateParams struct {
	ID       uuid.UUID
	Username string
	RoleId   uint
	NodeId   uuid.UUID
}

func (c *Create) Execute(ctx context.Context, params CreateParams) error {
	existing, err := c.repository.FindById(ctx, params.ID)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	return c.repository.Save(ctx, users.New(params.ID, params.Username, params.RoleId, params.NodeId))
}
