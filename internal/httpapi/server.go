// Package httpapi configures and runs the public HTTP API.
package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/cronos/keno4min/internal/config"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

// Server owns the Fiber application and its process lifecycle configuration.
type Server struct {
	app    *fiber.App
	cfg    config.Config
	logger *slog.Logger
}

// New creates a fully configured, but not yet listening, HTTP server.
func New(cfg config.Config, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	trustProxy := len(cfg.HTTP.TrustedProxies) > 0
	proxyHeader := ""
	if trustProxy {
		proxyHeader = cfg.HTTP.ProxyHeader
	}

	app := fiber.New(fiber.Config{
		AppName:            cfg.AppName,
		BodyLimit:          cfg.HTTP.BodyLimit,
		ReadTimeout:        cfg.HTTP.ReadTimeout,
		WriteTimeout:       cfg.HTTP.WriteTimeout,
		IdleTimeout:        cfg.HTTP.IdleTimeout,
		ProxyHeader:        proxyHeader,
		TrustProxy:         trustProxy,
		TrustProxyConfig:   fiber.TrustProxyConfig{Proxies: cfg.HTTP.TrustedProxies},
		EnableIPValidation: trustProxy,
		ErrorHandler:       errorHandler,
	})

	app.Use(requestid.New())
	app.Use(accessLog(logger))
	app.Use(recover.New(recover.Config{EnableStackTrace: cfg.IsDevelopment()}))
	app.Use(helmet.New())

	if len(cfg.HTTP.AllowedOrigins) > 0 {
		app.Use(cors.New(cors.Config{
			AllowOrigins: cfg.HTTP.AllowedOrigins,
			AllowHeaders: []string{
				fiber.HeaderOrigin,
				fiber.HeaderContentType,
				fiber.HeaderAccept,
				fiber.HeaderAuthorization,
				fiber.HeaderXRequestID,
			},
			ExposeHeaders: []string{fiber.HeaderXRequestID},
			MaxAge:        300,
		}))
	}

	registerRoutes(app, cfg)

	return &Server{app: app, cfg: cfg, logger: logger}
}

// App exposes the Fiber application for route registration and in-memory tests.
func (s *Server) App() *fiber.App {
	return s.app
}

// Run listens until the supplied context is canceled or the server fails.
func (s *Server) Run(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.app.Hooks().OnPostShutdown(func(err error) error {
		if err != nil {
			s.logger.Error("HTTP server shutdown failed", "error", err)
			return nil
		}
		s.logger.Info("HTTP server stopped")
		return nil
	})

	s.logger.Info("HTTP server starting", "address", s.cfg.HTTP.Address)
	listener, err := net.Listen("tcp", s.cfg.HTTP.Address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.cfg.HTTP.Address, err)
	}

	readyListener := &readinessListener{
		Listener: listener,
		ready:    make(chan struct{}),
	}
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- s.app.Listener(readyListener, fiber.ListenConfig{DisableStartupMessage: true})
	}()

	select {
	case <-readyListener.ready:
	case err := <-serverErr:
		return listenError(s.cfg.HTTP.Address, err)
	case <-ctx.Done():
		_ = listener.Close()
		return listenError(s.cfg.HTTP.Address, <-serverErr)
	}

	select {
	case err := <-serverErr:
		return listenError(s.cfg.HTTP.Address, err)
	case <-ctx.Done():
		s.logger.Info("HTTP server shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.HTTP.ShutdownTimeout)
		defer cancel()

		shutdownErr := s.app.ShutdownWithContext(shutdownCtx)
		listenErr := <-serverErr
		if shutdownErr != nil {
			return fmt.Errorf("shut down HTTP server: %w", shutdownErr)
		}
		return listenError(s.cfg.HTTP.Address, listenErr)
	}
}

type readinessListener struct {
	net.Listener
	once  sync.Once
	ready chan struct{}
}

func (l *readinessListener) Accept() (net.Conn, error) {
	l.once.Do(func() { close(l.ready) })
	return l.Listener.Accept()
}

func listenError(address string, err error) error {
	if err == nil || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return fmt.Errorf("serve on %s: %w", address, err)
}

func registerRoutes(app *fiber.App, cfg config.Config) {
	app.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"name": cfg.AppName,
		})
	})

	health := app.Group("/health")
	health.Get("/live", healthResponse)
	health.Get("/ready", healthResponse)

	api := app.Group("/api")
	api.Get("/v1", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"version": "v1"})
	})
}

func healthResponse(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

type errorEnvelope struct {
	Error errorResponse `json:"error"`
}

type errorResponse struct {
	Status    int    `json:"status"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

func errorHandler(c fiber.Ctx, err error) error {
	status := errorStatus(err)
	message := http.StatusText(status)

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) && fiberErr.Message != "" {
		message = fiberErr.Message
	}

	return c.Status(status).JSON(errorEnvelope{Error: errorResponse{
		Status:    status,
		Message:   message,
		RequestID: requestid.FromContext(c),
	}})
}

func accessLog(logger *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		started := time.Now()
		err := c.Next()
		status := c.Response().StatusCode()
		if err != nil {
			status = errorStatus(err)
		}

		level := slog.LevelInfo
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case status >= http.StatusBadRequest:
			level = slog.LevelWarn
		}

		attributes := []slog.Attr{
			slog.String("request_id", requestid.FromContext(c)),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.Duration("duration", time.Since(started)),
			slog.String("ip", c.IP()),
		}
		if err != nil {
			attributes = append(attributes, slog.String("error", err.Error()))
		}
		logger.LogAttrs(context.Background(), level, "HTTP request", attributes...)

		return err
	}
}

func errorStatus(err error) int {
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return fiberErr.Code
	}
	return http.StatusInternalServerError
}
