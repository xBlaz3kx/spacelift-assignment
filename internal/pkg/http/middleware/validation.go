package middleware

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func ValidateContentType(acceptedContentType string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get the Content-Type header
		contentType := c.Get("Content-Type")
		zap.L().Info("Content-Type", zap.String("Content-Type", contentType))

		// Check if the Content-Type is valid
		if !strings.Contains(contentType, acceptedContentType) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Message: fmt.Sprintf("Invalid Content-Type. Expected %s", acceptedContentType),
			})
		}

		// If the Content-Type is valid, proceed to the next middleware
		return c.Next()
	}
}
