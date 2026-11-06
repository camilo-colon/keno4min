package http

import "github.com/gofiber/fiber/v3"

type Routes struct {
	createTicket  *CreateTicketHandler
	listTickets   *ListTicketsHandler
	getTicket     *GetTicketHandler
	lookupTickets *LookupTicketsHandler
	repeatTicket  *RepeatTicketHandler
	cancelTicket  *CancelTicketHandler
}

func NewRoutes(createTicket *CreateTicketHandler, listTickets *ListTicketsHandler, getTicket *GetTicketHandler, lookupTickets *LookupTicketsHandler, repeatTicket *RepeatTicketHandler, cancelTicket *CancelTicketHandler) *Routes {
	return &Routes{createTicket: createTicket, listTickets: listTickets, getTicket: getTicket, lookupTickets: lookupTickets, repeatTicket: repeatTicket, cancelTicket: cancelTicket}
}

func (r *Routes) Register(router fiber.Router) {
	router.Get("/tickets/create", r.createTicket.Handle)
	router.Post("/tickets/lookup", r.lookupTickets.Handle)
	router.Get("/tickets", r.listTickets.Handle)
	router.Get("/tickets/:id", r.getTicket.Handle)
	router.Post("/tickets/:id/repeat", r.repeatTicket.Handle)
	router.Post("/tickets/:id/cancel", r.cancelTicket.Handle)
}
