package auth

import (
	"strings"
	"uuid"

	"github.com/gofiber/fiber/v3"
)

type Role string

type Identity struct {
	UserID   uuid.UUID
	Username string
	Role     Role
	NodeID   uuid.UUID
}

type contextKey int

const identityKey contextKey = iota

const bearerScheme = "Bearer"

func New() fiber.Handler {
	return func(ctx fiber.Ctx) error {
		token, ok := bearerToken(ctx.Get(fiber.HeaderAuthorization))
		if !ok {
			return unauthorized(ctx)
		}
		identity, err := decodeToken(token)
		if err != nil {
			return unauthorized(ctx)
		}
		fiber.StoreInContext(ctx, identityKey, identity)
		return ctx.Next()
	}
}

// FromContext accepts a fiber.Ctx or the context.Context returned by
// ctx.Context(); the latter requires PassLocalsToContext in fiber.Config.
func FromContext(ctx any) (Identity, bool) {
	return fiber.ValueFromContext[Identity](ctx, identityKey)
}

func bearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, bearerScheme) {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}

func unauthorized(ctx fiber.Ctx) error {
	ctx.Set(fiber.HeaderWWWAuthenticate, bearerScheme)
	return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or invalid bearer token"})
}
