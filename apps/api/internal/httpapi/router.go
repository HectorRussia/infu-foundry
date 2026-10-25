package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
)

func New(logger *slog.Logger, development bool) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:       "Infu Foundry API",
		StrictRouting: true,
		CaseSensitive: true,
		Immutable:     true,
		ReadTimeout:   10 * time.Second,
		WriteTimeout:  15 * time.Second,
		IdleTimeout:   60 * time.Second,
		BodyLimit:     1 << 20,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			var fiberError *fiber.Error
			if errors.As(err, &fiberError) {
				code = fiberError.Code
			}
			// Never expose arbitrary errors (including database/panic details).
			return c.Status(code).JSON(fiber.Map{"error": http.StatusText(code)})
		},
	})
	app.Use(func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		if err != nil {
			err = c.App().ErrorHandler(c, err)
		}
		logger.Info("http request", "method", c.Method(), "path", c.Path(), "status", c.Response().StatusCode(), "duration_ms", time.Since(start).Milliseconds())
		return err
	})
	app.Use(recoverer.New())
	if development {
		app.Use(func(c fiber.Ctx) error {
			c.Vary("Origin")
			origin := c.Get("Origin")
			if origin != "" && origin != "http://localhost:3000" {
				return fiber.ErrForbidden
			}
			return c.Next()
		})
		app.Use(cors.New(cors.Config{
			AllowOrigins: []string{"http://localhost:3000"},
			AllowMethods: []string{"GET", "HEAD", "OPTIONS"},
		}))
	}
	app.Add([]string{fiber.MethodGet, fiber.MethodHead}, "/api/v1/health", func(c fiber.Ctx) error {
		c.Set("Cache-Control", "no-store")
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.All("/api/v1/health", func(c fiber.Ctx) error {
		c.Set("Allow", "GET, HEAD")
		return fiber.ErrMethodNotAllowed
	})
	return app
}
