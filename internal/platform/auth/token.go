package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"uuid"
)

var errInvalidToken = errors.New("invalid token")

type claims struct {
	Subject    uuid.UUID `json:"sub"`
	UniqueName string    `json:"unique_name"`
	Role       string    `json:"role"`
	NodeID     uuid.UUID `json:"node_id"`
}

// decodeToken reads the JWT payload without verifying its signature or dates:
// the API gateway validates the token before the request reaches the API.
func decodeToken(token string) (Identity, error) {
	segments := strings.Split(token, ".")
	if len(segments) != 3 {
		return Identity{}, errInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(segments[1])
	if err != nil {
		return Identity{}, errInvalidToken
	}
	var c claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return Identity{}, errInvalidToken
	}
	if c.Subject == uuid.Nil() || c.NodeID == uuid.Nil() || c.UniqueName == "" || c.Role == "" {
		return Identity{}, errInvalidToken
	}
	return Identity{UserID: c.Subject, Username: c.UniqueName, Role: Role(c.Role), NodeID: c.NodeID}, nil
}
