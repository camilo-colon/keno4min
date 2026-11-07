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

type PayTicketHandler struct {
	useCase *usecase.Pay
}

func NewPayTicketHandler(useCase *usecase.Pay) *PayTicketHandler {
	return &PayTicketHandler{useCase: useCase}
}

type payTicketResponse struct {
	PrizeCents   int64 `json:"prizeCents"`
	BalanceCents int64 `json:"balanceCents"`
}

func (h *PayTicketHandler) Handle(ctx fiber.Ctx) error {
	identity, ok := auth.FromContext(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or invalid bearer token"})
	}
	ticketID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ticket id must be a valid UUID"})
	}
	result, err := h.useCase.Execute(ctx, usecase.PayParams{
		TicketID: ticketID,
		NodeID:   identity.NodeID,
		PaidBy:   identity.Username,
	})
	switch {
	case errors.Is(err, ports.ErrTicketNotFound):
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "ticket not found"})
	case errors.Is(err, tickets.ErrTicketAlreadyPaid):
		return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "ticket is already paid"})
	case errors.Is(err, tickets.ErrTicketNotPayable):
		return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "ticket has no prize to pay"})
	case errors.Is(err, ports.ErrBalanceUnavailable):
		return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "balance service is unavailable"})
	case err != nil:
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not pay ticket"})
	}
	return ctx.Status(fiber.StatusOK).JSON(payTicketResponse{PrizeCents: result.PrizeCents, BalanceCents: result.BalanceCents})
}
