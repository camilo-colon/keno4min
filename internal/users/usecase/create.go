package usecase

import (
	"context"
	"uuid"

	"github.com/cronos/keno4min/internal/users"
	"github.com/cronos/keno4min/internal/users/ports"
)

var ErrInvalidUser = users.ErrInvalidUser

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
	user, err := users.New(params.ID, params.Username, params.RoleId, params.NodeId)
	if err != nil {
		return err
	}
	return c.repository.Save(ctx, user)
}
