package serverfx

import "github.com/gofiber/fiber/v3"

// Health is the liveness check for the ALB target group. It must stay free of
// dependencies (DynamoDB, core) so a downstream outage does not cycle tasks.
func Health(ctx fiber.Ctx) error {
	return ctx.JSON(fiber.Map{"status": "ok"})
}
