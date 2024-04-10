package http

import (
	"github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/timeout"
	"github.com/spacelift-io/homework-object-storage/internal/gateway"
	"github.com/spacelift-io/homework-object-storage/internal/models/api"
	"github.com/spacelift-io/homework-object-storage/internal/pkg/http/middleware"
	"go.uber.org/zap"
	"time"
)

type Server struct {
	logger         *zap.Logger
	gatewayService gateway.Service
	app            *fiber.App
}

func NewServer(logger *zap.Logger, service gateway.Service) *Server {
	// Initialize a new Fiber app with a custom error handler
	fiberConfig := fiber.Config{ErrorHandler: middleware.FiberErrorHandler()}
	app := fiber.New(fiberConfig)

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

	recoveryConfig := recover.Config{
		EnableStackTrace: true,
	}
	// Add logger, recovery, timeout and health check middleware
	app.Use(fiberzap.New(config), recover.New(recoveryConfig), healthCheck)

	return &Server{
		logger:         logger,
		gatewayService: service,
		app:            app,
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

// gatewayRoutes defines the routes for the gateway gatewayService
func (s *Server) gatewayRoutes() {
	group := s.app.Group("/object")

	uploadHandler := func(c *fiber.Ctx) error {
		objectId := c.Params("id")
		// Validate objectId

		// Get file from form
		file, err := c.FormFile("fileUpload")
		if err != nil {
			return err
		}

		buffer, err := file.Open()
		if err != nil {
			return err
		}
		defer buffer.Close()

		// Call the gatewayService to upload the object
		err = s.gatewayService.AddOrUpdateObject(c.Context(), objectId, buffer)
		switch err {
		case nil:
			return c.Status(fiber.StatusCreated).JSON(api.ErrorResponse{Message: "Object uploaded successfully"})
		default:
			s.logger.Error("Failed to process request", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(api.ErrorResponse{Message: "Failed to upload object"})
		}
	}

	downloadHandler := func(c *fiber.Ctx) error {
		objectId := c.Params("id")

		// Call the gatewayService to download the object
		res, err := s.gatewayService.GetObject(c.Context(), objectId)
		switch err {
		case nil:
			return c.Status(fiber.StatusOK).SendStream(res)
		default:
			s.logger.Error("Failed to process request", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(api.ErrorResponse{Message: "Failed to download object"})
		}
	}

	group.Put("/:id", timeout.NewWithContext(uploadHandler, time.Second*30))
	group.Get("/:id", timeout.NewWithContext(downloadHandler, time.Second*30))
}
