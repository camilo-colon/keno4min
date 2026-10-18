package ports

import (
	"context"

	"github.com/cronos/keno4min/internal/nodes"
)

type Repository interface {
	Save(context.Context, *nodes.Node) error
}
