package events

import (
	"uuid"

	"github.com/cronos/keno4min/internal/common"
	"github.com/cronos/keno4min/internal/nodes"
)

var _ common.DomainEvent = (*NodeCreatedDomainEvent)(nil)

const NodeCreatedDomainEventType = "cronos.core.node.created"

type NodeCreatedDomainEvent struct {
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

func (n NodeCreatedDomainEvent) Type() string {
	return NodeCreatedDomainEventType
}
