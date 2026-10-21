package dynamodb

import (
	"fmt"
	"uuid"

	"github.com/cronos/keno4min/internal/nodes"
)

type nodeItem struct {
	PK           string `dynamodbav:"PK"`
	SK           string `dynamodbav:"SK"`
	GSI1PK       string `dynamodbav:"GSI1PK,omitempty"`
	GSI1SK       string `dynamodbav:"GSI1SK,omitempty"`
	ID           string `dynamodbav:"id"`
	ParentID     string `dynamodbav:"parentId,omitempty"`
	Type         string `dynamodbav:"type"`
	Name         string `dynamodbav:"name"`
	Address      string `dynamodbav:"address"`
	TimeZone     string `dynamodbav:"timeZone"`
	Currency     string `dynamodbav:"currency"`
	BalanceCents int    `dynamodbav:"balanceCents"`
	Active       bool   `dynamodbav:"active"`
}

func toItem(node *nodes.Node) nodeItem {
	item := nodeItem{
		PK:           "NODE#" + node.ID.String(),
		SK:           "META",
		ID:           node.ID.String(),
		Type:         string(node.Type),
		Name:         node.Name,
		Address:      node.Address,
		TimeZone:     node.TimeZone,
		Currency:     node.Currency,
		BalanceCents: node.BalanceCents,
		Active:       node.Active,
	}
	if node.ParentId != nil {
		item.ParentID = node.ParentId.String()
		item.GSI1PK = "NODE#" + item.ParentID
		item.GSI1SK = item.PK
	}
	return item
}

func fromItem(item nodeItem) (*nodes.Node, error) {
	id, err := uuid.Parse(item.ID)
	if err != nil {
		return nil, fmt.Errorf("parse id: %w", err)
	}
	var parentID *uuid.UUID
	if item.ParentID != "" {
		parsed, err := uuid.Parse(item.ParentID)
		if err != nil {
			return nil, fmt.Errorf("parse parentId: %w", err)
		}
		parentID = &parsed
	}
	return nodes.New(id, parentID, nodes.NodeType(item.Type), item.Name,
		item.Address, item.TimeZone, item.Currency, item.BalanceCents, item.Active), nil
}
