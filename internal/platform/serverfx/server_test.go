package serverfx

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/cronos/keno4min/internal/config"
	"github.com/gofiber/fiber/v3"
)

type stubRoute struct{}

func (stubRoute) Register(router fiber.Router) {
	router.Get("/draws/runtime", func(ctx fiber.Ctx) error { return ctx.SendStatus(fiber.StatusOK) })
}

func TestServerRoutes(t *testing.T) {
	server := NewServer(&config.Config{AppName: "test", Environment: "test"})
	RegisterRoutes(RouteParams{Server: server, Routes: []Route{stubRoute{}}})

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{name: "health needs no token", path: "/keno4min/health", wantStatus: fiber.StatusOK, wantBody: `{"status":"ok"}`},
		{name: "prefixed route requires auth", path: "/keno4min/draws/runtime", wantStatus: fiber.StatusUnauthorized},
		{name: "unprefixed route is gone", path: "/draws/runtime", wantStatus: fiber.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := server.app.Test(httptest.NewRequest(fiber.MethodGet, tt.path, nil))
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			if tt.wantBody != "" {
				body, _ := io.ReadAll(resp.Body)
				if string(body) != tt.wantBody {
					t.Fatalf("body = %s, want %s", body, tt.wantBody)
				}
			}
		})
	}
}
