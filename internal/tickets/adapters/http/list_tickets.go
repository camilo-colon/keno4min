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

type ListTicketsRequest struct {
	From   time.Time `query:"from" validate:"required"`
	To     time.Time `query:"to" validate:"required"`
	Limit  int32     `query:"limit" validate:"min=1,max=100"`
	Cursor string    `query:"cursor" validate:"max=4096"`
}

type ListTicketsHandler struct {
	useCase *usecase.List
}

func NewListTicketsHandler(useCase *usecase.List) *ListTicketsHandler {
	return &ListTicketsHandler{useCase: useCase}
}

func (h *ListTicketsHandler) Handle(ctx fiber.Ctx) error {
	identity, ok := auth.FromContext(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or invalid bearer token"})
	}
	request := ListTicketsRequest{Limit: 50}
	if err := ctx.Bind().Query(&request); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	page, err := h.useCase.Execute(ctx, ports.ListParams{
		NodeID: identity.NodeID, From: request.From, To: request.To,
		Limit: request.Limit, Cursor: request.Cursor,
	})
	if errors.Is(err, usecase.ErrInvalidListParams) || errors.Is(err, ports.ErrInvalidCursor) {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not list tickets"})
	}
	items := make([]listTicketResponse, len(page.Items))
	for i, ticket := range page.Items {
		items[i] = toListTicketResponse(ticket)
	}
	return ctx.JSON(listTicketsResponse{Items: items, NextCursor: page.NextCursor})
}

type listTicketsResponse struct {
	Items      []listTicketResponse `json:"items"`
	NextCursor string               `json:"nextCursor,omitempty"`
}

type listTicketResponse struct {
	ID         string                  `json:"id"`
	Coupon     string                  `json:"coupon"`
	NodeID     string                  `json:"nodeId"`
	TIR        uint64                  `json:"tir"`
	Bets       []listTicketBetResponse `json:"bets"`
	TotalCents int64                   `json:"totalCents"`
	Status     tickets.Status          `json:"status"`
	PrizeCents int64                   `json:"prizeCents"`
	CreatedBy  string                  `json:"createdBy"`
	CreatedAt  time.Time               `json:"createdAt"`
}

type listTicketBetResponse struct {
	Nums       []int `json:"nums"`
	ValueCents int64 `json:"valueCents"`
}

func toListTicketResponse(ticket *tickets.Ticket) listTicketResponse {
	result := listTicketResponse{
		ID: ticket.ID.String(), Coupon: ticket.Coupon, NodeID: ticket.NodeID.String(),
		TIR: ticket.TIR, Status: ticket.Status, PrizeCents: ticket.PrizeCents, CreatedBy: ticket.CreatedBy, CreatedAt: ticket.CreatedAt,
		Bets: make([]listTicketBetResponse, len(ticket.Bets)),
	}
	for i, bet := range ticket.Bets {
		nums := make([]int, len(bet.Nums))
		for j, num := range bet.Nums {
			nums[j] = int(num)
		}
		result.Bets[i] = listTicketBetResponse{Nums: nums, ValueCents: bet.ValueCents}
		result.TotalCents += bet.ValueCents
	}
	return result
}
