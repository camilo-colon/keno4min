package serverfx

import (
	"context"
	"fmt"

	"github.com/cronos/keno4min/internal/config"
	"github.com/cronos/keno4min/internal/platform/auth"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"go.uber.org/fx"
)

type Server struct {
	app *fiber.App
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
	app.Use(auth.New())

	return &Server{app: app, cfg: cfg}
}

func Run(lc fx.Lifecycle, s *Server) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			fmt.Println("Starting HTTP server at", s.cfg.HTTP.Address)
			go s.app.Listen(s.cfg.HTTP.Address)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return s.app.ShutdownWithContext(ctx)
		},
	})
}
