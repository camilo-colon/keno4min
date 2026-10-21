package dynamodb

import (
	"fmt"
	"uuid"

	"github.com/cronos/keno4min/internal/users"
)

type userItem struct {
	ID       string `dynamodbav:"id"`
	PK       string `dynamodbav:"PK"`
	SK       string `dynamodbav:"SK"`
	GSI1PK   string `dynamodbav:"GSI1PK"`
	GSI1SK   string `dynamodbav:"GSI1SK"`
	Username string `dynamodbav:"username"`
	RoleId   uint   `dynamodbav:"roleId"`
	NodeId   string `dynamodbav:"nodeId"`
}

func toItem(user *users.User) userItem {
	return userItem{
		PK: "USER#" + user.ID.String(), SK: "META",
		GSI1PK: "USER#" + user.Username, GSI1SK: "META",
		ID: user.ID.String(), Username: user.Username, RoleId: user.RoleId, NodeId: user.NodeId.String(),
	}
}

func fromItem(item userItem) (*users.User, error) {
	if item.Username == "" {
		return nil, fmt.Errorf("missing username")
	}
	id, err := uuid.Parse(item.ID)
	if err != nil {
		return nil, fmt.Errorf("parse id: %w", err)
	}
	nodeID, err := uuid.Parse(item.NodeId)
	if err != nil {
		return nil, fmt.Errorf("parse nodeId: %w", err)
	}
	return users.New(id, item.Username, item.RoleId, nodeID), nil
}
