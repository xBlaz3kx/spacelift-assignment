package api

import (
	"github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/spacelift-io/homework-object-storage/internal/gateway"
	"go.uber.org/zap"
)

func NewServer(logger *zap.Logger, service gateway.Service, listenAddress string) {
	// Initialize a new Fiber app
	app := fiber.New()

	// Use zap logger middleware
	config := fiberzap.ConfigDefault
	config.Logger = logger

	// Create a new health check middleware
	healthCheck := healthcheck.New(healthcheck.Config{
		LivenessProbe: func(c *fiber.Ctx) bool {
			return true
		},
		LivenessEndpoint: "/live",
		ReadinessProbe: func(c *fiber.Ctx) bool {
			return service.Ready(c.Context())
		},
		ReadinessEndpoint: "/ready",
	})

	// Add logger and health check middleware
	app.Use(fiberzap.New(config), healthCheck)

	GatewayRoutes(app, service)

	// Start the server on port 3000
	err := app.Listen(listenAddress)
	if err != nil {
		logger.Fatal("failed to start server", zap.Error(err))
	}
}

func GatewayRoutes(app *fiber.App, service gateway.Service) {
	group := app.Group("/object")
	group.Put("/:id", func(c *fiber.Ctx) error {

		return nil
	})

	group.Get("/:id", func(c *fiber.Ctx) error {

		return nil
	})
}
