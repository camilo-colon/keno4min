package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v3"
)

const problemMediaType = "application/problem+json"

type problemDetails struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail"`
	Instance  string `json:"instance"`
	Code      string `json:"code"`
	RequestID string `json:"requestId"`
}

func problemErrorHandler(generator func() string) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		status := fiber.StatusInternalServerError
		var fiberError *fiber.Error
		if errors.As(err, &fiberError) {
			status = fiberError.Code
		}

		requestID := requestIDFromContext(c)
		if requestID == "" {
			requestID = generateRequestID(generator)
			setRequestID(c, requestID)
		}

		problem := problemForStatus(status)
		problem.Instance = c.Path()
		problem.RequestID = requestID

		body, marshalErr := json.Marshal(problem)
		if marshalErr != nil {
			return marshalErr
		}

		c.Set(fiber.HeaderContentType, problemMediaType)
		return c.Status(status).Send(body)
	}
}

func problemForStatus(status int) problemDetails {
	problem := problemDetails{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Code:   "http_error",
		Detail: "The request could not be processed.",
	}

	switch status {
	case fiber.StatusBadRequest:
		problem.Code = "bad_request"
	case fiber.StatusUnauthorized:
		problem.Code = "unauthorized"
	case fiber.StatusForbidden:
		problem.Code = "forbidden"
	case fiber.StatusNotFound:
		problem.Code = "not_found"
		problem.Detail = "The requested resource was not found."
	case fiber.StatusMethodNotAllowed:
		problem.Code = "method_not_allowed"
		problem.Detail = "The requested method is not allowed for this resource."
	case fiber.StatusConflict:
		problem.Code = "conflict"
	case fiber.StatusRequestEntityTooLarge:
		problem.Code = "payload_too_large"
		problem.Detail = "The request body exceeds the configured limit."
	case fiber.StatusUnsupportedMediaType:
		problem.Code = "unsupported_media_type"
	case fiber.StatusUnprocessableEntity:
		problem.Code = "unprocessable_content"
	case fiber.StatusTooManyRequests:
		problem.Code = "too_many_requests"
	case fiber.StatusServiceUnavailable:
		problem.Code = "service_unavailable"
	case fiber.StatusInternalServerError:
		problem.Code = "internal_server_error"
		problem.Detail = "An unexpected error occurred."
	}

	if problem.Title == "" {
		problem.Title = "HTTP Error"
	}
	return problem
}
