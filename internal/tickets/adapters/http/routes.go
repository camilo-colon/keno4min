package http

import "github.com/gofiber/fiber/v3"

type Routes struct {
	createTicket *CreateTicketHandler
	listTickets  *ListTicketsHandler
}

func NewRoutes(createTicket *CreateTicketHandler, listTickets *ListTicketsHandler) *Routes {
	return &Routes{createTicket: createTicket, listTickets: listTickets}
}

func (r *Routes) Register(router fiber.Router) {
	router.Get("/tickets/create", r.createTicket.Handle)
	router.Get("/tickets", r.listTickets.Handle)
}
