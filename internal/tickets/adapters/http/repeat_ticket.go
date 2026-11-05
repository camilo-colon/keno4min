package http

import (
	"errors"
	"uuid"

	"github.com/cronos/keno4min/internal/platform/auth"
	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
	"github.com/cronos/keno4min/internal/tickets/usecase"
	"github.com/gofiber/fiber/v3"
)

type RepeatTicketHandler struct {
	useCase *usecase.Repeat
}

func NewRepeatTicketHandler(useCase *usecase.Repeat) *RepeatTicketHandler {
	return &RepeatTicketHandler{useCase: useCase}
}

func (h *RepeatTicketHandler) Handle(ctx fiber.Ctx) error {
	identity, ok := auth.FromContext(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or invalid bearer token"})
	}
	ticketID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ticket id must be a valid UUID"})
	}
	ticket, err := h.useCase.Execute(ctx, usecase.RepeatParams{
		TicketID:  ticketID,
		NodeID:    identity.NodeID,
		CreatedBy: identity.Username,
	})
	if errors.Is(err, ports.ErrTicketNotFound) {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "ticket not found"})
	}
	if errors.Is(err, tickets.ErrBettingClosed) {
		return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "draw is not accepting tickets"})
	}
	if errors.Is(err, usecase.ErrDrawUnavailable) {
		return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "current draw is unavailable"})
	}
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not repeat ticket"})
	}
	return ctx.Status(fiber.StatusCreated).JSON(toResponse(ticket))
}
