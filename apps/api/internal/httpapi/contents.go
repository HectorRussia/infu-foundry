package httpapi

import (
	"context"
	"errors"
	"regexp"

	"github.com/gofiber/fiber/v3"
	"infu-foundry/apps/api/internal/contents"
	"infu-foundry/apps/api/internal/media"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type ContentStore interface {
	List(ctx context.Context, filter contents.ListFilter) ([]contents.Summary, error)
	Get(ctx context.Context, id string) (contents.Content, error)
	Create(ctx context.Context, authorID string, in contents.Input) (contents.Content, error)
	Update(ctx context.Context, id string, in contents.Input) (contents.Content, error)
	Delete(ctx context.Context, id string) error
	SetCover(ctx context.Context, id, key string) (string, error)
	SetCoverPosition(ctx context.Context, id string, x, y int) error
}

// coverURL signs a cover key for viewing; without storage (R2 off) covers are hidden.
func coverURL(ctx context.Context, storage MediaStorage, key string) (string, error) {
	if storage == nil || key == "" {
		return "", nil
	}
	return storage.ViewURL(ctx, key)
}

// withCoverURLs signs the view and download URLs of a content's cover.
func withCoverURLs(ctx context.Context, storage MediaStorage, item contents.Content) (contents.Content, error) {
	var err error
	if item.CoverURL, err = coverURL(ctx, storage, item.CoverKey); err != nil || item.CoverURL == "" {
		return item, err
	}
	item.CoverDownloadURL, err = storage.DownloadURL(ctx, item.CoverKey, media.CoverFilename(item.Title, item.CoverKey))
	return item, err
}

// registerContents expects requireSession to run first: every signed-in user sees
// every content, and the session user becomes the author of new content.
// storage may be nil; then no cover URLs are returned.
func registerContents(router fiber.Router, store ContentStore, storage MediaStorage, guards ...any) {
	withCover := func(c fiber.Ctx, item contents.Content) error {
		item, err := withCoverURLs(c.Context(), storage, item)
		if err != nil {
			return err
		}
		return c.JSON(item)
	}
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
		switch {
		case errors.Is(err, contents.ErrNotFound):
			return fiber.ErrNotFound
		case errors.Is(err, contents.ErrCategoryNotFound):
			// 422, not 404: a missing category must not look like a missing content.
			return fiber.ErrUnprocessableEntity
		default:
			return err
		}
	}

	group := router.Group("/contents", append([]any{noStore}, guards...)...)
	group.Get("", func(c fiber.Ctx) error {
		filter := contents.ListFilter{Category: c.Query("category")}
		if filter.Category != "" && filter.Category != contents.Uncategorized && !uuidPattern.MatchString(filter.Category) {
			return fiber.ErrUnprocessableEntity
		}
		items, err := store.List(c.Context(), filter)
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
	group.Post("", func(c fiber.Ctx) error {
		in, err := input(c)
		if err != nil {
			return err
		}
		item, err := store.Create(c.Context(), currentUser(c).ID, in)
		if err != nil {
			return storeError(err)
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
		return withCover(c, item)
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
		return withCover(c, item)
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
