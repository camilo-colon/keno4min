package nodes

import "uuid"

type NodeType string

const (
	Admin   NodeType = "Admin"
	Cashier NodeType = "Cashier"
)

type Node struct {
	ID           uuid.UUID
	ParentId     *uuid.UUID
	Type         NodeType
	Name         string
	Address      string
	TimeZone     string
	Currency     string
	BalanceCents int
	Active       bool
}

func New(id uuid.UUID, parentId *uuid.UUID, nodeType NodeType, name, address, timeZone, currency string, balanceCents int, active bool) *Node {
	return &Node{
		ID:           id,
		ParentId:     parentId,
		Type:         nodeType,
		Name:         name,
		TimeZone:     timeZone,
		Address:      address,
		Currency:     currency,
		BalanceCents: balanceCents,
		Active:       active,
	}
}
