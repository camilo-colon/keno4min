package users

import (
	"uuid"
)

type User struct {
	ID       uuid.UUID
	Username string
	RoleId   uint
	NodeId   uuid.UUID
}

func New(id uuid.UUID, username string, roleId uint, nodeId uuid.UUID) *User {
	return &User{
		ID:       id,
		Username: username,
		RoleId:   roleId,
		NodeId:   nodeId,
	}
}
