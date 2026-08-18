// Package httpapi provides the process-wide HTTP boundary.
package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

// Config contains limits applied directly by Fiber and its listener lifecycle.
type Config struct {
	BodyLimit       int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// Server owns the Fiber application and its graceful shutdown budget.
type Server struct {
	app             *fiber.App
	shutdownTimeout time.Duration
}

// New constructs a Fiber application without registering production routes.
func New(cfg Config, options ...option) *Server {
	settings := optionsConfig{requestID: newRequestID}
	for _, apply := range options {
		apply(&settings)
	}

	app := fiber.New(fiber.Config{
		BodyLimit:    cfg.BodyLimit,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
		ErrorHandler: problemErrorHandler(settings.requestID),
		TrustProxy:   false,
	})

	app.Use(localRequestID(settings.requestID))
	app.Use(recover.New(recover.Config{EnableStackTrace: false}))

	return &Server{app: app, shutdownTimeout: cfg.ShutdownTimeout}
}

// App exposes the transport application so versioned adapters can register
// contract-backed routes. Callers must not retain request-scoped Fiber data.
func (s *Server) App() *fiber.App {
	return s.app
}

// Serve owns the listener lifecycle. It does not return from cancellation until
// Fiber's shutdown and the listener loop have both completed.
func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	if ctx.Err() != nil {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			return fmt.Errorf("close HTTP listener before start: %w", err)
		}
		return nil
	}

	listenerWithReady := &readyListener{Listener: listener, ready: make(chan struct{})}
	serveResult := make(chan error, 1)
	go func() {
		serveResult <- normalizeServeError(s.app.Listener(listenerWithReady, fiber.ListenConfig{
			DisableStartupMessage: true,
		}))
	}()

	select {
	case err := <-serveResult:
		return err
	case <-listenerWithReady.ready:
	}

	select {
	case err := <-serveResult:
		return err
	case <-ctx.Done():
	}

	shutdownErr := s.app.ShutdownWithTimeout(s.shutdownTimeout)
	if shutdownErr != nil {
		shutdownErr = fmt.Errorf("shutdown HTTP server: %w", shutdownErr)
	}
	serveErr := <-serveResult
	return errors.Join(shutdownErr, serveErr)
}

// readyListener signals from Accept, after fasthttp has registered the
// listener as running and can therefore be shut down without a startup race.
type readyListener struct {
	net.Listener
	readyOnce sync.Once
	ready     chan struct{}
}

func (l *readyListener) Accept() (net.Conn, error) {
	l.readyOnce.Do(func() { close(l.ready) })
	return l.Listener.Accept()
}

func normalizeServeError(err error) error {
	if err == nil || errors.Is(err, http.ErrServerClosed) || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return fmt.Errorf("HTTP listener loop: %w", err)
}
