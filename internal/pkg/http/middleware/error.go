package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"
)

// FiberErrorHandler is a middleware that handles errors returned by the handlers
func FiberErrorHandler() func(ctx *fiber.Ctx, err error) error {
	return func(ctx *fiber.Ctx, err error) error {
		// Status code defaults to 500
		code := fiber.StatusInternalServerError

		// Retrieve the custom status code if it's a *fiber.Error
		var e *fiber.Error
		if errors.As(err, &e) {
			code = e.Code
		}

		// Return from handler
		return ctx.Status(code).JSON(fiber.Map{})
	}
}
