package middleware

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

var alphanumeric = regexp.MustCompile("^[a-zA-Z0-9_]{1,32}$")

func validateObjectId(id string) bool {
	return alphanumeric.MatchString(id)
}

func ValidateObjectId() fiber.Handler {
	return func(c *fiber.Ctx) error {
		objectId := c.Params("id")

		if !validateObjectId(objectId) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Message: "Invalid object ID",
			})
		}

		return c.Next()
	}
}

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
