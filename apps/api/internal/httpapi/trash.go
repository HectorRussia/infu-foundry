package httpapi

import (
	"context"
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"infu-foundry/apps/api/internal/contents"
)

type TrashStore interface {
	ListTrash(ctx context.Context) ([]contents.Trashed, error)
	Restore(ctx context.Context, id string) error
	TrashObjectKeys(ctx context.Context, id string) ([]string, error)
	Purge(ctx context.Context, id string) error
}

// registerTrash lists deleted contents and lets signed-in users restore them or
// delete them for good. A purge frees R2 first: if any object cannot be deleted
// the row stays in the trash so the user can retry, and nothing is orphaned.
func registerTrash(router fiber.Router, store TrashStore, storage MediaStorage, logger *slog.Logger, guards ...any) {
	notFound := func(err error) error {
		if errors.Is(err, contents.ErrNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}
	purge := func(c fiber.Ctx, id string) error {
		keys, err := store.TrashObjectKeys(c.Context(), id)
		if err != nil {
			return notFound(err)
		}
		for _, key := range keys {
			if err := storage.Delete(c.Context(), key); err != nil {
				logger.Error("trash purge: object not deleted", "content_id", id, "error", err)
				return err
			}
		}
		if err := store.Purge(c.Context(), id); err != nil {
			return notFound(err)
		}
		logger.Info("content purged", "content_id", id, "objects", len(keys))
		return nil
	}
	trashID := func(c fiber.Ctx) (string, error) {
		id := c.Params("id")
		if !uuidPattern.MatchString(id) {
			return "", fiber.ErrNotFound
		}
		return id, nil
	}

	group := router.Group("/trash", append([]any{func(c fiber.Ctx) error {
		c.Set("Cache-Control", "no-store")
		return c.Next()
	}}, guards...)...)

	group.Get("", func(c fiber.Ctx) error {
		items, err := store.ListTrash(c.Context())
		if err != nil {
			return err
		}
		for i := range items {
			if items[i].CoverURL, err = coverURL(c.Context(), storage, items[i].CoverKey); err != nil {
				return err
			}
		}
		return c.JSON(fiber.Map{"items": items})
	})

	group.Post("/:id/restore", func(c fiber.Ctx) error {
		id, err := trashID(c)
		if err != nil {
			return err
		}
		if err := store.Restore(c.Context(), id); err != nil {
			return notFound(err)
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	group.Delete("/:id", func(c fiber.Ctx) error {
		id, err := trashID(c)
		if err != nil {
			return err
		}
		if err := purge(c, id); err != nil {
			return err
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	// DELETE /trash empties the whole trash, one content at a time.
	group.Delete("", func(c fiber.Ctx) error {
		items, err := store.ListTrash(c.Context())
		if err != nil {
			return err
		}
		for _, item := range items {
			if err := purge(c, item.ID); err != nil && !errors.Is(err, fiber.ErrNotFound) {
				return err
			}
		}
		return c.SendStatus(fiber.StatusNoContent)
	})
}
