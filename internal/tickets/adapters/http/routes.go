package http

import "github.com/gofiber/fiber/v3"

type Routes struct {
	createTicket *CreateTicketHandler
}

func NewRoutes(createTicket *CreateTicketHandler) *Routes {
	return &Routes{createTicket: createTicket}
}

func (r *Routes) Register(router fiber.Router) {
	router.Get("/tickets/create", r.createTicket.Handle)
}
