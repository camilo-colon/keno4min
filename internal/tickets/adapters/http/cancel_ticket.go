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

type CancelTicketHandler struct {
	useCase *usecase.Cancel
}

func NewCancelTicketHandler(useCase *usecase.Cancel) *CancelTicketHandler {
	return &CancelTicketHandler{useCase: useCase}
}

func (h *CancelTicketHandler) Handle(ctx fiber.Ctx) error {
	identity, ok := auth.FromContext(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or invalid bearer token"})
	}
	ticketID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ticket id must be a valid UUID"})
	}
	result, err := h.useCase.Execute(ctx, usecase.CancelParams{
		TicketID:    ticketID,
		NodeID:      identity.NodeID,
		CancelledBy: identity.Username,
	})
	switch {
	case errors.Is(err, ports.ErrTicketNotFound):
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "ticket not found"})
	case errors.Is(err, tickets.ErrBettingClosed):
		return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "ticket draw is closed"})
	case errors.Is(err, tickets.ErrTicketAlreadyCancelled):
		return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "ticket is already cancelled"})
	case errors.Is(err, tickets.ErrTicketNotCancellable):
		return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "ticket can no longer be cancelled"})
	case errors.Is(err, usecase.ErrDrawUnavailable):
		return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "current draw is unavailable"})
	case errors.Is(err, ports.ErrBalanceUnavailable):
		return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "balance service is unavailable"})
	case err != nil:
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not cancel ticket"})
	}
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"balanceCents": result.BalanceCents})
}
