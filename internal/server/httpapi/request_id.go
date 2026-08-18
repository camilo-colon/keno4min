package httpapi

import (
	"crypto/rand"

	"github.com/gofiber/fiber/v3"
)

type requestIDKey struct{}

type option func(*optionsConfig)

type optionsConfig struct {
	requestID func() string
}

func withRequestIDGenerator(generator func() string) option {
	return func(cfg *optionsConfig) {
		cfg.requestID = generator
	}
}

func newRequestID() string {
	return rand.Text()
}

func localRequestID(generator func() string) fiber.Handler {
	return func(c fiber.Ctx) error {
		setRequestID(c, generateRequestID(generator))
		return c.Next()
	}
}

func setRequestID(c fiber.Ctx, requestID string) {
	fiber.Locals(c, requestIDKey{}, requestID)
}

func requestIDFromContext(c fiber.Ctx) string {
	return fiber.Locals[string](c, requestIDKey{})
}

func generateRequestID(generator func() string) string {
	if generator != nil {
		if requestID := generator(); validRequestID(requestID) {
			return requestID
		}
	}
	return newRequestID()
}

func validRequestID(requestID string) bool {
	if requestID == "" {
		return false
	}
	for index := range len(requestID) {
		if requestID[index] < 0x20 || requestID[index] > 0x7e {
			return false
		}
	}
	return true
}
