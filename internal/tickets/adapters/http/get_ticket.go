package http

import (
	"errors"
	"time"
	"uuid"

	"github.com/cronos/keno4min/internal/platform/auth"
	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/ports"
	"github.com/cronos/keno4min/internal/tickets/usecase"
	"github.com/gofiber/fiber/v3"
)

type GetTicketHandler struct {
	useCase *usecase.Get
}

func NewGetTicketHandler(useCase *usecase.Get) *GetTicketHandler {
	return &GetTicketHandler{useCase: useCase}
}

func (h *GetTicketHandler) Handle(ctx fiber.Ctx) error {
	identity, ok := auth.FromContext(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or invalid bearer token"})
	}
	ticketID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ticket id must be a valid UUID"})
	}
	detail, err := h.useCase.Execute(ctx, usecase.GetParams{TicketID: ticketID, NodeID: identity.NodeID})
	if errors.Is(err, ports.ErrTicketNotFound) {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "ticket not found"})
	}
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not get ticket"})
	}
	return ctx.JSON(toGetTicketResponse(detail))
}

type getTicketResponse struct {
	ID         string                 `json:"id"`
	Coupon     string                 `json:"coupon"`
	NodeID     string                 `json:"nodeId"`
	TIR        uint64                 `json:"tir"`
	Bets       []getTicketBetResponse `json:"bets"`
	TotalCents int64                  `json:"totalCents"`
	Status     tickets.Status         `json:"status"`
	PrizeCents int64                  `json:"prizeCents"`
	Balls      []int                  `json:"balls"`
	CreatedBy  string                 `json:"createdBy"`
	CreatedAt  time.Time              `json:"createdAt"`
}

type getTicketBetResponse struct {
	Nums       []int `json:"nums"`
	ValueCents int64 `json:"valueCents"`
}

func toGetTicketResponse(detail *usecase.TicketDetail) getTicketResponse {
	ticket := detail.Ticket
	result := getTicketResponse{
		ID: ticket.ID.String(), Coupon: ticket.Coupon, NodeID: ticket.NodeID.String(),
		TIR: ticket.TIR, Status: ticket.Status, PrizeCents: ticket.PrizeCents, CreatedBy: ticket.CreatedBy, CreatedAt: ticket.CreatedAt,
		Bets:  make([]getTicketBetResponse, len(ticket.Bets)),
		Balls: append([]int{}, detail.Balls...),
	}
	for i, bet := range ticket.Bets {
		nums := make([]int, len(bet.Nums))
		for j, num := range bet.Nums {
			nums[j] = int(num)
		}
		result.Bets[i] = getTicketBetResponse{Nums: nums, ValueCents: bet.ValueCents}
		result.TotalCents += bet.ValueCents
	}
	return result
}
