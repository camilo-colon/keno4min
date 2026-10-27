package http

import (
	"errors"
	"strconv"
	"strings"
	"time"
	"uuid"

	"github.com/cronos/keno4min/internal/tickets"
	"github.com/cronos/keno4min/internal/tickets/usecase"
	"github.com/gofiber/fiber/v3"
)

type CreateTicketRequest struct {
	UserID string `query:"userId"`
	Stv    string `query:"stv"`
	Sum    string `query:"sum"`
}

type CreateTicketHandler struct {
	useCase *usecase.Create
}

func NewCreateTicketHandler(useCase *usecase.Create) *CreateTicketHandler {
	return &CreateTicketHandler{useCase: useCase}
}

func (h *CreateTicketHandler) Handle(ctx fiber.Ctx) error {
	params, err := parseCreateTicket(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	ticket, err := h.useCase.Execute(ctx, params)
	if errors.Is(err, tickets.ErrBettingClosed) {
		return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "draw is not accepting tickets"})
	}
	if errors.Is(err, usecase.ErrUserNotFound) {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}
	if errors.Is(err, usecase.ErrDrawUnavailable) {
		return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "current draw is unavailable"})
	}
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create ticket"})
	}
	return ctx.Status(fiber.StatusCreated).JSON(toResponse(ticket))
}

func parseCreateTicket(ctx fiber.Ctx) (usecase.CreateParams, error) {
	query := ctx.RequestCtx().QueryArgs()
	if query.Len() != 3 || !query.Has("userId") || !query.Has("stv") || !query.Has("sum") {
		return usecase.CreateParams{}, errors.New("provide only userId, stv and sum, exactly once each")
	}
	var request CreateTicketRequest
	if err := ctx.Bind().Query(&request); err != nil {
		return usecase.CreateParams{}, err
	}
	userID, err := uuid.Parse(request.UserID)
	if err != nil || userID == uuid.Nil() {
		return usecase.CreateParams{}, errors.New("userId must be a non-zero UUID")
	}
	selections, values := strings.Split(request.Stv, "_"), strings.Split(request.Sum, "_")
	if len(selections) != len(values) {
		return usecase.CreateParams{}, errors.New("stv and sum must contain the same number of bets")
	}
	bets := make([]tickets.Bet, len(selections))
	for i, selection := range selections {
		numbers := strings.Split(selection, ";")
		bets[i].Nums = make([]uint8, len(numbers))
		for j, number := range numbers {
			value, err := strconv.ParseUint(number, 10, 8)
			if err != nil {
				return usecase.CreateParams{}, errors.New("bet numbers must be positive integers between 1 and 80")
			}
			bets[i].Nums[j] = uint8(value)
		}
		value, err := strconv.ParseUint(values[i], 10, 63)
		if err != nil {
			return usecase.CreateParams{}, errors.New("bet value must be a positive integer within int64 range")
		}
		bets[i].ValueCents = int64(value)
	}
	if err := tickets.ValidateBets(bets); err != nil {
		return usecase.CreateParams{}, err
	}
	return usecase.CreateParams{UserID: userID, Bets: bets}, nil
}

type betResponse struct {
	Nums       []int `json:"nums"`
	ValueCents int64 `json:"valueCents"`
}

type createTicketResponse struct {
	ID         string         `json:"id"`
	Coupon     string         `json:"coupon"`
	NodeID     string         `json:"nodeId"`
	TIR        uint64         `json:"tir"`
	Bets       []betResponse  `json:"bets"`
	TotalCents int64          `json:"totalCents"`
	Status     tickets.Status `json:"status"`
	CreatedBy  string         `json:"createdBy"`
	CreatedAt  time.Time      `json:"createdAt"`
}

func toResponse(ticket *tickets.Ticket) createTicketResponse {
	result := createTicketResponse{
		ID: ticket.ID.String(), Coupon: ticket.Coupon, NodeID: ticket.NodeID.String(),
		TIR: ticket.TIR, Status: ticket.Status, CreatedBy: ticket.CreatedBy, CreatedAt: ticket.CreatedAt,
		Bets: make([]betResponse, len(ticket.Bets)),
	}
	for i, bet := range ticket.Bets {
		nums := make([]int, len(bet.Nums))
		for j, num := range bet.Nums {
			nums[j] = int(num)
		}
		result.Bets[i] = betResponse{Nums: nums, ValueCents: bet.ValueCents}
		result.TotalCents += bet.ValueCents
	}
	return result
}
