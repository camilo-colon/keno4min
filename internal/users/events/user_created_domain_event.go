package events

import (
	"uuid"

	"github.com/cronos/keno4min/internal/common"
)

var _ common.DomainEvent = (*UserCreatedDomainEvent)(nil)

const UserCreatedDomainEventType = "cronos.core.user.created"

type UserCreatedDomainEvent struct {
	common.BaseEvent
	UserID   uuid.UUID `json:"userId"`
	Username string    `json:"username"`
	RoleId   uint      `json:"roleId"`
	NodeId   uuid.UUID `json:"nodeId"`
}

func (u UserCreatedDomainEvent) Type() string {
	return UserCreatedDomainEventType
}
