package httpapi

import (
	"context"
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"infu-foundry/apps/api/internal/media"
)

type MediaStore interface {
	CreatePending(ctx context.Context, contentID, uploaderID string, r media.UploadRequest) (media.Media, error)
	Get(ctx context.Context, id string) (media.Media, error)
	MarkReady(ctx context.Context, id string) error
	ListReady(ctx context.Context, contentID string) ([]media.Media, error)
	ListGallery(ctx context.Context) ([]media.GalleryItem, error)
	SoftDelete(ctx context.Context, id string) error
}

// MediaStorage is the object store (Cloudflare R2) that browsers reach through presigned URLs.
type MediaStorage interface {
	UploadURL(ctx context.Context, key, contentType string, size int64) (string, error)
	ViewURL(ctx context.Context, key string) (string, error)
	Stat(ctx context.Context, key string) (int64, string, error)
	Delete(ctx context.Context, key string) error
}

type mediaWithURL struct {
	media.Media
	URL string `json:"url"`
}

// registerMedia expects requireSession in guards: any signed-in user may attach,
// view and delete videos, matching the shared content library.
func registerMedia(router fiber.Router, store MediaStore, storage MediaStorage, logger *slog.Logger, guards ...any) {
	noStore := func(c fiber.Ctx) error {
		c.Set("Cache-Control", "no-store")
		return c.Next()
	}
	handlers := append([]any{noStore}, guards...)
	idParam := func(c fiber.Ctx, name string) (string, error) {
		id := c.Params(name)
		if !uuidPattern.MatchString(id) {
			return "", fiber.ErrNotFound
		}
		return id, nil
	}
	notFound := func(err error) error {
		if errors.Is(err, media.ErrNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	contents := router.Group("/contents/:id/media", handlers...)
	contents.Post("", func(c fiber.Ctx) error {
		contentID, err := idParam(c, "id")
		if err != nil {
			return err
		}
		var in media.UploadRequest
		if err := c.Bind().JSON(&in); err != nil {
			return fiber.ErrBadRequest
		}
		if err := in.Validate(); err != nil {
			return fiber.ErrUnprocessableEntity
		}
		item, err := store.CreatePending(c.Context(), contentID, currentUser(c).ID, in)
		if err != nil {
			return notFound(err)
		}
		url, err := storage.UploadURL(c.Context(), item.ObjectKey, item.ContentType, item.SizeBytes)
		if err != nil {
			return err
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"media": item, "uploadUrl": url})
	})
	contents.Get("", func(c fiber.Ctx) error {
		contentID, err := idParam(c, "id")
		if err != nil {
			return err
		}
		items, err := store.ListReady(c.Context(), contentID)
		if err != nil {
			return err
		}
		out := make([]mediaWithURL, 0, len(items))
		for _, item := range items {
			url, err := storage.ViewURL(c.Context(), item.ObjectKey)
			if err != nil {
				return err
			}
			out = append(out, mediaWithURL{Media: item, URL: url})
		}
		return c.JSON(fiber.Map{"items": out})
	})

	group := router.Group("/media", handlers...)
	group.Get("", func(c fiber.Ctx) error {
		items, err := store.ListGallery(c.Context())
		if err != nil {
			return err
		}
		type galleryWithURL struct {
			media.GalleryItem
			URL string `json:"url"`
		}
		out := make([]galleryWithURL, 0, len(items))
		for _, item := range items {
			url, err := storage.ViewURL(c.Context(), item.ObjectKey)
			if err != nil {
				return err
			}
			out = append(out, galleryWithURL{GalleryItem: item, URL: url})
		}
		return c.JSON(fiber.Map{"items": out})
	})
	group.Post("/:mediaId/complete", func(c fiber.Ctx) error {
		id, err := idParam(c, "mediaId")
		if err != nil {
			return err
		}
		item, err := store.Get(c.Context(), id)
		if err != nil {
			return notFound(err)
		}
		size, _, err := storage.Stat(c.Context(), item.ObjectKey)
		if errors.Is(err, media.ErrObjectMissing) {
			return fiber.ErrConflict
		}
		if err != nil {
			return err
		}
		if size != item.SizeBytes {
			// Signed Content-Length should make this impossible; never trust a mismatch.
			logger.Warn("uploaded object size mismatch", "media_id", item.ID)
			return fiber.ErrConflict
		}
		if err := store.MarkReady(c.Context(), id); err != nil {
			return err
		}
		item.Status = "ready"
		url, err := storage.ViewURL(c.Context(), item.ObjectKey)
		if err != nil {
			return err
		}
		return c.JSON(mediaWithURL{Media: item, URL: url})
	})
	group.Delete("/:mediaId", func(c fiber.Ctx) error {
		id, err := idParam(c, "mediaId")
		if err != nil {
			return err
		}
		item, err := store.Get(c.Context(), id)
		if err != nil {
			return notFound(err)
		}
		// Free storage first: a failed delete keeps the row so the user can retry.
		if err := storage.Delete(c.Context(), item.ObjectKey); err != nil {
			return err
		}
		if err := store.SoftDelete(c.Context(), id); err != nil {
			return err
		}
		return c.SendStatus(fiber.StatusNoContent)
	})
}
