package ports

import (
	"context"
	"uuid"

	"github.com/cronos/keno4min/internal/nodes"
)

type Repository interface {
	// FindById returns (nil, nil) when the node does not exist.
	FindById(context.Context, uuid.UUID) (*nodes.Node, error)
	Save(context.Context, *nodes.Node) error
}
