package http

import "github.com/gofiber/fiber/v3"

type Routes struct {
	createTicket *CreateTicketHandler
	listTickets  *ListTicketsHandler
	getTicket    *GetTicketHandler
}

func NewRoutes(createTicket *CreateTicketHandler, listTickets *ListTicketsHandler, getTicket *GetTicketHandler) *Routes {
	return &Routes{createTicket: createTicket, listTickets: listTickets, getTicket: getTicket}
}

func (r *Routes) Register(router fiber.Router) {
	router.Get("/tickets/create", r.createTicket.Handle)
	router.Get("/tickets", r.listTickets.Handle)
	router.Get("/tickets/:id", r.getTicket.Handle)
}
