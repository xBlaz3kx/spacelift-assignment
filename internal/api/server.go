package api

import (
	"github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	timeout2 "github.com/gofiber/fiber/v2/middleware/timeout"
	"github.com/spacelift-io/homework-object-storage/internal/gateway"
	"go.uber.org/zap"
	"time"
)

type Server struct {
	logger  *zap.Logger
	service gateway.Service
	app     *fiber.App
}

func NewServer(logger *zap.Logger, service gateway.Service) *Server {
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

	// todo timeout handler
	h := func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	}
	timeoutHandler := timeout2.NewWithContext(h, time.Second*10)

	// Add logger, timeout and health check middleware
	app.Use(fiberzap.New(config), timeoutHandler, healthCheck)

	return &Server{
		logger:  logger,
		service: service,
		app:     app,
	}
}

// Run starts the server that will listen on the given address
func (s *Server) Run(listenAddress string) {
	// Mount gateway routes
	s.gatewayRoutes()

	// Start the server on port 3000
	err := s.app.Listen(listenAddress)
	if err != nil {
		s.logger.Fatal("failed to start server", zap.Error(err))
	}
}

// gatewayRoutes defines the routes for the gateway service
func (s *Server) gatewayRoutes() {
	group := s.app.Group("/object")
	group.Put("/:id", func(c *fiber.Ctx) error {

		return nil
	})

	group.Get("/:id", func(c *fiber.Ctx) error {

		return nil
	})
}
