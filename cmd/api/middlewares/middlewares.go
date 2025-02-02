package middlewares

import (
	"github.com/gofiber/fiber/v2"
	"github.com/vingarcia/ddd-go-layout/domain"
)

func HandleRequestID() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		key, requestID := domain.GenerateRequestID()
		c.Locals(key, requestID)
		return c.Next()
	}
}

func HandleError(logger domain.LogProvider) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		if err == nil {
			return nil
		}

		req := c.Request()
		status, body := domain.HandleDomainErrAsHTTP(
			c.Context(),
			logger,
			err,
			string(req.Header.Method()),
			string(req.RequestURI()),
		)
		c.Status(status).Send(body)
		return nil
	}
}
