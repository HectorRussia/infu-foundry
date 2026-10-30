package httpapi

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v3"
	"infu-foundry/apps/api/internal/contents"
	"infu-foundry/apps/api/internal/media"
)

// registerCovers lets signed-in users set one cover image per content. Like
// videos, the browser PUTs the image straight to R2; Go only issues the URL,
// checks the uploaded object and records its key.
func registerCovers(router fiber.Router, store ContentStore, storage MediaStorage, logger *slog.Logger, guards ...any) {
	contentID := func(c fiber.Ctx) (string, error) {
		id := c.Params("id")
		if !uuidPattern.MatchString(id) {
			return "", fiber.ErrNotFound
		}
		return id, nil
	}
	notFound := func(err error) error {
		if errors.Is(err, contents.ErrNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}
	// Removing the old image is best effort: the new cover is already saved.
	deleteOld := func(c fiber.Ctx, key string) {
		if key == "" {
			return
		}
		if err := storage.Delete(c.Context(), key); err != nil {
			logger.Warn("old cover not deleted", "content_id", c.Params("id"), "error", err)
		}
	}

	// coverResponse returns the cover fields of the content as now stored.
	coverResponse := func(c fiber.Ctx, id string) error {
		item, err := store.Get(c.Context(), id)
		if err != nil {
			return notFound(err)
		}
		if item, err = withCoverURLs(c.Context(), storage, item); err != nil {
			return err
		}
		return c.JSON(fiber.Map{
			"coverUrl":         item.CoverURL,
			"coverDownloadUrl": item.CoverDownloadURL,
			"coverPositionX":   item.CoverPositionX,
			"coverPositionY":   item.CoverPositionY,
		})
	}

	group := router.Group("/contents/:id/cover", append([]any{func(c fiber.Ctx) error {
		c.Set("Cache-Control", "no-store")
		return c.Next()
	}}, guards...)...)

	// POST issues an upload URL for a new cover image.
	group.Post("", func(c fiber.Ctx) error {
		id, err := contentID(c)
		if err != nil {
			return err
		}
		var in struct {
			ContentType string `json:"contentType"`
			Size        int64  `json:"size"`
		}
		if err := c.Bind().JSON(&in); err != nil {
			return fiber.ErrBadRequest
		}
		if !media.ValidCover(in.ContentType, in.Size) {
			return fiber.ErrUnprocessableEntity
		}
		if _, err := store.Get(c.Context(), id); err != nil {
			return notFound(err)
		}
		key, err := media.NewCoverKey(id, in.ContentType)
		if err != nil {
			return err
		}
		url, err := storage.UploadURL(c.Context(), key, in.ContentType, in.Size)
		if err != nil {
			return err
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"key": key, "uploadUrl": url})
	})

	// PUT makes an uploaded image the cover after checking it in R2.
	group.Put("", func(c fiber.Ctx) error {
		id, err := contentID(c)
		if err != nil {
			return err
		}
		var in struct {
			Key string `json:"key"`
		}
		if err := c.Bind().JSON(&in); err != nil {
			return fiber.ErrBadRequest
		}
		if !media.CoverKeyOf(in.Key, id) {
			return fiber.ErrUnprocessableEntity
		}
		size, contentType, err := storage.Stat(c.Context(), in.Key)
		if errors.Is(err, media.ErrObjectMissing) {
			return fiber.ErrConflict
		}
		if err != nil {
			return err
		}
		if !media.ValidCover(strings.TrimSpace(strings.Split(contentType, ";")[0]), size) {
			return fiber.ErrUnprocessableEntity
		}
		old, err := store.SetCover(c.Context(), id, in.Key)
		if err != nil {
			return notFound(err)
		}
		if old != in.Key {
			deleteOld(c, old)
		}
		return coverResponse(c, id)
	})

	// PATCH moves the crop: which part of the image stays visible on cards.
	group.Patch("", func(c fiber.Ctx) error {
		id, err := contentID(c)
		if err != nil {
			return err
		}
		var in struct {
			X *int `json:"x"`
			Y *int `json:"y"`
		}
		if err := c.Bind().JSON(&in); err != nil {
			return fiber.ErrBadRequest
		}
		if in.X == nil || in.Y == nil || !contents.ValidCoverPosition(*in.X, *in.Y) {
			return fiber.ErrUnprocessableEntity
		}
		if err := store.SetCoverPosition(c.Context(), id, *in.X, *in.Y); err != nil {
			return notFound(err)
		}
		return coverResponse(c, id)
	})

	group.Delete("", func(c fiber.Ctx) error {
		id, err := contentID(c)
		if err != nil {
			return err
		}
		old, err := store.SetCover(c.Context(), id, "")
		if err != nil {
			return notFound(err)
		}
		deleteOld(c, old)
		return c.SendStatus(fiber.StatusNoContent)
	})
}
