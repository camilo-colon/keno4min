package events

import (
	"uuid"

	"github.com/cronos/keno4min/internal/common"
	"github.com/cronos/keno4min/internal/nodes"
)

var _ common.DomainEvent = (*NodeCreated)(nil)

const NodeCreatedType = "cronos.core.node.created"

type NodeCreated struct {
	common.BaseEvent

	NodeId       uuid.UUID      `json:"nodeId"`
	ParentId     *uuid.UUID     `json:"parentId"`
	NodeType     nodes.NodeType `json:"nodeType"`
	Number       int            `json:"number"`
	Name         string         `json:"name"`
	Address      string         `json:"address"`
	TimeZone     string         `json:"timeZone"`
	Currency     string         `json:"currency"`
	BalanceCents int            `json:"balanceCents"`
	Active       bool           `json:"active"`
}

func (n NodeCreated) Type() string {
	return NodeCreatedType
}
