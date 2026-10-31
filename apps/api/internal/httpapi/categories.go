package httpapi

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v3"
	"infu-foundry/apps/api/internal/contents"
)

type CategoryStore interface {
	ListCategories(ctx context.Context) ([]contents.Category, error)
	CreateCategory(ctx context.Context, name string) (contents.Category, error)
	RenameCategory(ctx context.Context, id, name string) (contents.Category, error)
	DeleteCategory(ctx context.Context, id string) error
}

// registerCategories serves the shared folders. Deleting one only removes the
// folder; the store leaves every content in place, now uncategorized.
func registerCategories(router fiber.Router, store CategoryStore, guards ...any) {
	categoryID := func(c fiber.Ctx) (string, error) {
		id := c.Params("id")
		if !uuidPattern.MatchString(id) {
			return "", fiber.ErrNotFound
		}
		return id, nil
	}
	input := func(c fiber.Ctx) (contents.CategoryInput, error) {
		var in contents.CategoryInput
		if err := c.Bind().JSON(&in); err != nil {
			return in, fiber.ErrBadRequest
		}
		if err := in.Validate(); err != nil {
			return in, fiber.ErrUnprocessableEntity
		}
		return in, nil
	}
	storeError := func(err error) error {
		switch {
		case errors.Is(err, contents.ErrCategoryNotFound):
			return fiber.ErrNotFound
		case errors.Is(err, contents.ErrCategoryExists):
			return fiber.ErrConflict
		case errors.Is(err, contents.ErrInvalid):
			return fiber.ErrUnprocessableEntity
		default:
			return err
		}
	}

	group := router.Group("/categories", append([]any{func(c fiber.Ctx) error {
		c.Set("Cache-Control", "no-store")
		return c.Next()
	}}, guards...)...)

	group.Get("", func(c fiber.Ctx) error {
		items, err := store.ListCategories(c.Context())
		if err != nil {
			return err
		}
		return c.JSON(fiber.Map{"items": items})
	})
	group.Post("", func(c fiber.Ctx) error {
		in, err := input(c)
		if err != nil {
			return err
		}
		item, err := store.CreateCategory(c.Context(), in.Name)
		if err != nil {
			return storeError(err)
		}
		return c.Status(fiber.StatusCreated).JSON(item)
	})
	group.Patch("/:id", func(c fiber.Ctx) error {
		id, err := categoryID(c)
		if err != nil {
			return err
		}
		in, err := input(c)
		if err != nil {
			return err
		}
		item, err := store.RenameCategory(c.Context(), id, in.Name)
		if err != nil {
			return storeError(err)
		}
		return c.JSON(item)
	})
	group.Delete("/:id", func(c fiber.Ctx) error {
		id, err := categoryID(c)
		if err != nil {
			return err
		}
		if err := store.DeleteCategory(c.Context(), id); err != nil {
			return storeError(err)
		}
		return c.SendStatus(fiber.StatusNoContent)
	})
}
