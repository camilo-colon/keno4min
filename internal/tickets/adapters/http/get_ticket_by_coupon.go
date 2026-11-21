package http

import (
	"errors"

	"github.com/cronos/keno4min/internal/platform/auth"
	"github.com/cronos/keno4min/internal/tickets/ports"
	"github.com/cronos/keno4min/internal/tickets/usecase"
	"github.com/gofiber/fiber/v3"
)

// GetTicketByCouponHandler answers with the same body as GetTicketHandler.
type GetTicketByCouponHandler struct {
	useCase *usecase.GetByCoupon
}

func NewGetTicketByCouponHandler(useCase *usecase.GetByCoupon) *GetTicketByCouponHandler {
	return &GetTicketByCouponHandler{useCase: useCase}
}

func (h *GetTicketByCouponHandler) Handle(ctx fiber.Ctx) error {
	identity, ok := auth.FromContext(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or invalid bearer token"})
	}
	ticket, err := h.useCase.Execute(ctx, usecase.GetByCouponParams{Coupon: ctx.Params("coupon"), NodeID: identity.NodeID})
	if errors.Is(err, ports.ErrTicketNotFound) {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "ticket not found"})
	}
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not get ticket"})
	}
	return ctx.JSON(toGetTicketResponse(ticket))
}
