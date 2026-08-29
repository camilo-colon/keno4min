package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cronos/keno4min/internal/config"
	"github.com/gofiber/fiber/v3"
)

func TestHealthEndpoints(t *testing.T) {
	app := newTestServer(nil).App()

	for _, path := range []string{"/health/live", "/health/ready"} {
		t.Run(path, func(t *testing.T) {
			response, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil))
			if err != nil {
				t.Fatalf("request %s: %v", path, err)
			}
			defer response.Body.Close()

			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
			}
			requireContentType(t, response, fiber.MIMEApplicationJSON)

			var payload map[string]string
			if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if payload["status"] != "ok" {
				t.Fatalf("payload = %#v", payload)
			}
			if response.Header.Get(fiber.HeaderXRequestID) == "" {
				t.Fatal("response does not include a request ID")
			}
		})
	}
}

func TestMetadataEndpoints(t *testing.T) {
	app := newTestServer(nil).App()

	tests := []struct {
		path     string
		field    string
		expected string
	}{
		{path: "/", field: "name", expected: "keno4min-test"},
		{path: "/api/v1", field: "version", expected: "v1"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			response, err := app.Test(httptest.NewRequest(http.MethodGet, tt.path, nil))
			if err != nil {
				t.Fatalf("request %s: %v", tt.path, err)
			}
			defer response.Body.Close()

			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
			}
			requireContentType(t, response, fiber.MIMEApplicationJSON)

			var payload map[string]string
			if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if payload[tt.field] != tt.expected {
				t.Fatalf("payload = %#v", payload)
			}
		})
	}
}

func TestErrorResponseKeepsRequestID(t *testing.T) {
	app := newTestServer(nil).App()
	request := httptest.NewRequest(http.MethodGet, "/missing", nil)
	request.Header.Set(fiber.HeaderXRequestID, "known-request-id")

	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request missing route: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusNotFound)
	}
	if response.Header.Get(fiber.HeaderXRequestID) != "known-request-id" {
		t.Fatalf("request ID header = %q", response.Header.Get(fiber.HeaderXRequestID))
	}
	requireContentType(t, response, "application/problem+json")

	var payload problemDetails
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Type != "about:blank" ||
		payload.Title != http.StatusText(http.StatusNotFound) ||
		payload.Status != http.StatusNotFound ||
		payload.Detail == "" ||
		payload.RequestID != "known-request-id" {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestRecoverDoesNotExposePanic(t *testing.T) {
	app := newTestServer(nil).App()
	app.Get("/panic", func(fiber.Ctx) error {
		panic("sensitive panic detail")
	})

	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	request.Header.Set(fiber.HeaderXRequestID, "panic-request-id")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request panic route: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusInternalServerError)
	}
	requireContentType(t, response, "application/problem+json")
	if response.Header.Get(fiber.HeaderXRequestID) != "panic-request-id" {
		t.Fatalf("request ID header = %q", response.Header.Get(fiber.HeaderXRequestID))
	}
	if strings.Contains(string(body), "sensitive panic detail") {
		t.Fatalf("panic detail leaked in response: %s", body)
	}

	var payload problemDetails
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Type != "about:blank" ||
		payload.Title != http.StatusText(http.StatusInternalServerError) ||
		payload.Status != http.StatusInternalServerError ||
		payload.Detail != http.StatusText(http.StatusInternalServerError) ||
		payload.RequestID != "panic-request-id" {
		t.Fatalf("payload = %+v", payload)
	}
}

func requireContentType(t *testing.T, response *http.Response, expected string) {
	t.Helper()

	contentType := response.Header.Get(fiber.HeaderContentType)
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		t.Fatalf("parse content type %q: %v", contentType, err)
	}
	if mediaType != expected {
		t.Fatalf("content type = %q, want %q", mediaType, expected)
	}
}

func TestConfiguredCORS(t *testing.T) {
	app := newTestServer([]string{"https://app.example.com"}).App()
	request := httptest.NewRequest(http.MethodOptions, "/health/live", nil)
	request.Header.Set(fiber.HeaderOrigin, "https://app.example.com")
	request.Header.Set(fiber.HeaderAccessControlRequestMethod, http.MethodGet)

	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("CORS preflight: %v", err)
	}
	defer response.Body.Close()

	if got := response.Header.Get(fiber.HeaderAccessControlAllowOrigin); got != "https://app.example.com" {
		t.Fatalf("allow origin = %q", got)
	}
}

func TestRunShutsDownWhenContextIsCanceled(t *testing.T) {
	server := newTestServer(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server.App().Hooks().OnListen(func(fiber.ListenData) error {
		cancel()
		return nil
	})

	if err := server.Run(ctx); err != nil {
		t.Fatalf("run server: %v", err)
	}
}

func newTestServer(origins []string) *Server {
	cfg := config.Config{
		AppName:     "keno4min-test",
		Environment: "test",
		HTTP: config.HTTPConfig{
			Address:         ":0",
			ReadTimeout:     time.Second,
			WriteTimeout:    time.Second,
			IdleTimeout:     time.Second,
			ShutdownTimeout: time.Second,
			BodyLimit:       1024,
			AllowedOrigins:  origins,
			ProxyHeader:     "X-Forwarded-For",
		},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(cfg, logger)
}
