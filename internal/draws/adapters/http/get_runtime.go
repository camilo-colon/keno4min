package http

import (
	"github.com/cronos/keno4min/internal/draws/usecase"
	"github.com/gofiber/fiber/v3"
)

type GetRuntimeHandler struct {
	useCase *usecase.GetRuntime
}

func NewGetRuntimeHandler(useCase *usecase.GetRuntime) *GetRuntimeHandler {
	return &GetRuntimeHandler{useCase: useCase}
}

func (h *GetRuntimeHandler) Handle(ctx fiber.Ctx) error {
	result, err := h.useCase.Execute(ctx)
	if err != nil {
		return fiber.ErrInternalServerError
	}
	return ctx.Status(fiber.StatusOK).JSON(result)
}
