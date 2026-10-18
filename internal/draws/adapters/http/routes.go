package http

import "github.com/gofiber/fiber/v3"

type Routes struct {
	getRuntime *GetRuntimeHandler
}

func NewRoutes(getRuntime *GetRuntimeHandler) *Routes {
	return &Routes{
		getRuntime: getRuntime,
	}
}

func (r *Routes) Register(router fiber.Router) {
	draw := router.Group("/draws")
	draw.Get("/runtime", r.getRuntime.Handle)
}
