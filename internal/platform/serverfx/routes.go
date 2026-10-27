package serverfx

import (
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
)

type Route interface {
	Register(router fiber.Router)
}

type RouteParams struct {
	fx.In

	Server     *Server
	Routes     []Route `group:"routes"`
	RootRoutes []Route `group:"rootRoutes"`
}

func RegisterRoutes(params RouteParams) {
	for _, route := range params.RootRoutes {
		route.Register(params.Server.app)
	}
	api := params.Server.app.Group("/v1")
	for _, route := range params.Routes {
		route.Register(api)
	}
}
