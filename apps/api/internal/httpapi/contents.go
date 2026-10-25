package httpapi

import (
	"context"
	"errors"
	"regexp"

	"github.com/gofiber/fiber/v3"
	"infu-foundry/apps/api/internal/contents"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type ContentStore interface {
	List(ctx context.Context) ([]contents.Summary, error)
	Get(ctx context.Context, id string) (contents.Content, error)
	Create(ctx context.Context, authorID string, in contents.Input) (contents.Content, error)
	Update(ctx context.Context, id string, in contents.Input) (contents.Content, error)
	Delete(ctx context.Context, id string) error
}

// registerContents expects requireSession to run first: every signed-in user sees
// every content, and the session user becomes the author of new content.
func registerContents(router fiber.Router, store ContentStore, guards ...any) {
	noStore := func(c fiber.Ctx) error {
		c.Set("Cache-Control", "no-store")
		return c.Next()
	}
	contentID := func(c fiber.Ctx) (string, error) {
		id := c.Params("id")
		if !uuidPattern.MatchString(id) {
			return "", fiber.ErrNotFound
		}
		return id, nil
	}
	input := func(c fiber.Ctx) (contents.Input, error) {
		var in contents.Input
		if err := c.Bind().JSON(&in); err != nil {
			return in, fiber.ErrBadRequest
		}
		if err := in.Validate(); err != nil {
			return in, fiber.ErrUnprocessableEntity
		}
		return in, nil
	}
	storeError := func(err error) error {
		if errors.Is(err, contents.ErrNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	group := router.Group("/contents", append([]any{noStore}, guards...)...)
	group.Get("", func(c fiber.Ctx) error {
		items, err := store.List(c.Context())
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
		item, err := store.Create(c.Context(), currentUser(c).ID, in)
		if err != nil {
			return err
		}
		return c.Status(fiber.StatusCreated).JSON(item)
	})
	group.Get("/:id", func(c fiber.Ctx) error {
		id, err := contentID(c)
		if err != nil {
			return err
		}
		item, err := store.Get(c.Context(), id)
		if err != nil {
			return storeError(err)
		}
		return c.JSON(item)
	})
	group.Put("/:id", func(c fiber.Ctx) error {
		id, err := contentID(c)
		if err != nil {
			return err
		}
		in, err := input(c)
		if err != nil {
			return err
		}
		item, err := store.Update(c.Context(), id, in)
		if err != nil {
			return storeError(err)
		}
		return c.JSON(item)
	})
	group.Delete("/:id", func(c fiber.Ctx) error {
		id, err := contentID(c)
		if err != nil {
			return err
		}
		if err := store.Delete(c.Context(), id); err != nil {
			return storeError(err)
		}
		return c.SendStatus(fiber.StatusNoContent)
	})
}
