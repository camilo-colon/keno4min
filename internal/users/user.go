package users

import (
	"errors"
	"strings"
	"uuid"
)

var ErrInvalidUser = errors.New("userId, username and nodeId are required")

type User struct {
	ID       uuid.UUID
	Username string
	RoleId   uint
	NodeId   uuid.UUID
}

func New(id uuid.UUID, username string, roleId uint, nodeId uuid.UUID) (*User, error) {
	u := &User{
		ID:       id,
		Username: username,
		RoleId:   roleId,
		NodeId:   nodeId,
	}
	if err := u.Validate(); err != nil {
		return nil, err
	}
	return u, nil
}

func (u *User) Validate() error {
	if u.ID == uuid.Nil() || strings.TrimSpace(u.Username) == "" || u.NodeId == uuid.Nil() {
		return ErrInvalidUser
	}
	return nil
}
