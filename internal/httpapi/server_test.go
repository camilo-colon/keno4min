package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
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

	var payload errorEnvelope
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error.Status != http.StatusNotFound || payload.Error.RequestID != "known-request-id" {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestRecoverDoesNotExposePanic(t *testing.T) {
	app := newTestServer(nil).App()
	app.Get("/panic", func(fiber.Ctx) error {
		panic("sensitive panic detail")
	})

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/panic", nil))
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
	if strings.Contains(string(body), "sensitive panic detail") {
		t.Fatalf("panic detail leaked in response: %s", body)
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
