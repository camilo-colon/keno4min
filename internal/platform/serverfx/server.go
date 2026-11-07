package serverfx

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/cronos/keno4min/internal/config"
	"github.com/cronos/keno4min/internal/platform/auth"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"go.uber.org/fx"
)

// BasePath prefixes every HTTP route. The shared ALB and API Gateway forward
// the full path, so the service owns the prefix instead of a path rewrite.
const BasePath = "/keno4min"

type Server struct {
	app *fiber.App
	// api is the BasePath group, already behind the auth middleware.
	api fiber.Router
	cfg *config.Config
}

func NewServer(cfg *config.Config) *Server {
	app := fiber.New(fiber.Config{
		AppName:             cfg.AppName,
		PassLocalsToContext: true,
		StructValidator:     NewStructValidator(),
	})

	app.Use(requestid.New())
	app.Use(recover.New(recover.Config{EnableStackTrace: cfg.IsDevelopment()}))

	api := app.Group(BasePath)
	// Registered before auth so the ALB target group health check needs no token.
	api.Get("/health", Health)
	api.Use(auth.New())

	return &Server{app: app, api: api, cfg: cfg}
}

func Run(lc fx.Lifecycle, shutdowner fx.Shutdowner, s *Server) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Bind synchronously so an unavailable address fails fx startup
			// and the process exits non-zero.
			ln, err := net.Listen(fiber.NetworkTCP4, s.cfg.HTTP.Address)
			if err != nil {
				return fmt.Errorf("listen on %s: %w", s.cfg.HTTP.Address, err)
			}
			fmt.Println("Starting HTTP server at", ln.Addr())
			go func() {
				if err := s.app.Listener(ln); err != nil {
					log.Printf("HTTP server stopped: %v", err)
					if err := shutdowner.Shutdown(fx.ExitCode(1)); err != nil {
						log.Printf("shut down after HTTP server failure: %v", err)
					}
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return s.app.ShutdownWithContext(ctx)
		},
	})
}
