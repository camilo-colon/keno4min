package http

import (
	"errors"
	"uuid"

	"github.com/cronos/keno4min/internal/platform/auth"
	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/usecase"
	"github.com/gofiber/fiber/v3"
)

type LookupTicketsRequest struct {
	IDs []uuid.UUID `json:"ids"`
}

type LookupTicketsHandler struct {
	useCase *usecase.Lookup
}

func NewLookupTicketsHandler(useCase *usecase.Lookup) *LookupTicketsHandler {
	return &LookupTicketsHandler{useCase: useCase}
}

func (h *LookupTicketsHandler) Handle(ctx fiber.Ctx) error {
	identity, ok := auth.FromContext(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or invalid bearer token"})
	}
	var request LookupTicketsRequest
	if err := ctx.Bind().Body(&request); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ids must be a list of valid UUIDs"})
	}
	found, err := h.useCase.Execute(ctx, usecase.LookupParams{TicketIDs: request.IDs, NodeID: identity.NodeID})
	if errors.Is(err, usecase.ErrInvalidLookupParams) {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not lookup tickets"})
	}
	items := make([]lookupTicketResponse, len(found))
	for i, ticket := range found {
		items[i] = lookupTicketResponse{ID: ticket.ID.String(), Status: ticket.Status, PrizeCents: ticket.PrizeCents}
	}
	return ctx.JSON(lookupTicketsResponse{Items: items})
}

type lookupTicketsResponse struct {
	Items []lookupTicketResponse `json:"items"`
}

type lookupTicketResponse struct {
	ID         string         `json:"id"`
	Status     tickets.Status `json:"status"`
	PrizeCents int64          `json:"prizeCents"`
}
