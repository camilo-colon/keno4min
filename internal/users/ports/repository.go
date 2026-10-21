package ports

import (
	"context"
	"uuid"

	"github.com/cronos/keno4min/internal/users"
)

type Repository interface {
	// FindById returns (nil, nil) when the user does not exist.
	FindById(context.Context, uuid.UUID) (*users.User, error)
	// FindByUsername returns the first match from an eventually consistent index,
	// or (nil, nil) when the user does not exist.
	FindByUsername(context.Context, string) (*users.User, error)
	Save(context.Context, *users.User) error
}
