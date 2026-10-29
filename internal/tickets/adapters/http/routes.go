package http

import "github.com/gofiber/fiber/v3"

type Routes struct {
	createTicket  *CreateTicketHandler
	listTickets   *ListTicketsHandler
	getTicket     *GetTicketHandler
	lookupTickets *LookupTicketsHandler
}

func NewRoutes(createTicket *CreateTicketHandler, listTickets *ListTicketsHandler, getTicket *GetTicketHandler, lookupTickets *LookupTicketsHandler) *Routes {
	return &Routes{createTicket: createTicket, listTickets: listTickets, getTicket: getTicket, lookupTickets: lookupTickets}
}

func (r *Routes) Register(router fiber.Router) {
	router.Get("/tickets/create", r.createTicket.Handle)
	router.Post("/tickets/lookup", r.lookupTickets.Handle)
	router.Get("/tickets", r.listTickets.Handle)
	router.Get("/tickets/:id", r.getTicket.Handle)
}
