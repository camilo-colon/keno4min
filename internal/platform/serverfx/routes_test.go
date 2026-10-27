package serverfx

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

type routeFunc func(fiber.Router)

func (f routeFunc) Register(router fiber.Router) { f(router) }

func TestRegisterRoutesKeepsRootAndVersionedPaths(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(RouteParams{
		Server: &Server{app: app},
		RootRoutes: []Route{routeFunc(func(r fiber.Router) {
			r.Get("/tickets", func(c fiber.Ctx) error { return c.SendStatus(201) })
		})},
		Routes: []Route{routeFunc(func(r fiber.Router) {
			r.Get("/draws/runtime", func(c fiber.Ctx) error { return c.SendStatus(200) })
		})},
	})
	for path, status := range map[string]int{"/tickets": 201, "/v1/draws/runtime": 200, "/v1/tickets": 404, "/draws/runtime": 404} {
		response, err := app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != status {
			t.Fatalf("%s: status %d, want %d", path, response.StatusCode, status)
		}
	}
}
