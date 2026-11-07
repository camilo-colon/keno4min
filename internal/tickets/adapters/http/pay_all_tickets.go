package http

import (
	"errors"
	"time"

	"github.com/cronos/keno4min/internal/platform/auth"
	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
	"github.com/cronos/keno4min/internal/tickets/usecase"
	"github.com/gofiber/fiber/v3"
)

type PayAllTicketsRequest struct {
	From time.Time `json:"from" validate:"required"`
	To   time.Time `json:"to" validate:"required"`
}

type PayAllTicketsHandler struct {
	useCase *usecase.PayAll
}

func NewPayAllTicketsHandler(useCase *usecase.PayAll) *PayAllTicketsHandler {
	return &PayAllTicketsHandler{useCase: useCase}
}

type payAllTicketsResponse struct {
	PaidCount       int                    `json:"paidCount"`
	TotalPrizeCents int64                  `json:"totalPrizeCents"`
	BalanceCents    *int64                 `json:"balanceCents,omitempty"`
	Paid            []paidTicketResponse   `json:"paid"`
	Failed          []failedTicketResponse `json:"failed"`
}

type paidTicketResponse struct {
	ID         string `json:"id"`
	Coupon     string `json:"coupon"`
	PrizeCents int64  `json:"prizeCents"`
}

type failedTicketResponse struct {
	ID     string `json:"id"`
	Coupon string `json:"coupon"`
	Reason string `json:"reason"`
}

func (h *PayAllTicketsHandler) Handle(ctx fiber.Ctx) error {
	identity, ok := auth.FromContext(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or invalid bearer token"})
	}
	var request PayAllTicketsRequest
	if err := ctx.Bind().Body(&request); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "from and to must be RFC 3339 dates"})
	}
	result, err := h.useCase.Execute(ctx, usecase.PayAllParams{
		NodeID: identity.NodeID, PaidBy: identity.Username, From: request.From, To: request.To,
	})
	if errors.Is(err, usecase.ErrInvalidPayAllParams) {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not pay tickets"})
	}

	response := payAllTicketsResponse{
		PaidCount:       len(result.Paid),
		TotalPrizeCents: result.TotalPrizeCents(),
		BalanceCents:    result.BalanceCents,
		Paid:            make([]paidTicketResponse, len(result.Paid)),
		Failed:          make([]failedTicketResponse, len(result.Failed)),
	}
	for i, paid := range result.Paid {
		response.Paid[i] = paidTicketResponse{ID: paid.TicketID.String(), Coupon: paid.Coupon, PrizeCents: paid.PrizeCents}
	}
	balanceUnavailable := false
	for i, failed := range result.Failed {
		response.Failed[i] = failedTicketResponse{ID: failed.TicketID.String(), Coupon: failed.Coupon, Reason: failureReason(failed.Err)}
		balanceUnavailable = balanceUnavailable || errors.Is(failed.Err, ports.ErrBalanceUnavailable)
	}

	status := fiber.StatusOK
	if len(result.Paid) == 0 && balanceUnavailable {
		status = fiber.StatusServiceUnavailable
	}
	return ctx.Status(status).JSON(response)
}

func failureReason(err error) string {
	switch {
	case errors.Is(err, ports.ErrBalanceUnavailable):
		return "balance_unavailable"
	case errors.Is(err, tickets.ErrTicketAlreadyPaid):
		return "already_paid"
	case errors.Is(err, tickets.ErrTicketNotPayable):
		return "not_payable"
	default:
		return "internal_error"
	}
}
