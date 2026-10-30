// Package media stores metadata for videos attached to contents. The files live
// in Cloudflare R2; the database keeps only each object's key.
package media

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
)

const (
	MaxSize           = 1 << 30 // 1 GB, also enforced by a database CHECK
	maxFilenameLength = 255
	galleryLimit      = 200
)

var (
	ErrNotFound = errors.New("media not found")
	ErrInvalid  = errors.New("invalid media")
)

// extensions lists the accepted video types and the extension used in object keys.
var extensions = map[string]string{
	"video/mp4":       ".mp4",
	"video/quicktime": ".mov",
	"video/webm":      ".webm",
}

type Media struct {
	ID          string    `json:"id"`
	ContentID   string    `json:"contentId"`
	UploaderID  string    `json:"uploaderId"`
	ObjectKey   string    `json:"-"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"contentType"`
	SizeBytes   int64     `json:"sizeBytes"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

// GalleryItem is a ready video with the content it belongs to, for the gallery page.
type GalleryItem struct {
	Media
	ContentTitle string `json:"contentTitle"`
	UploaderName string `json:"uploaderName"`
}

type UploadRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}

// Validate trims the filename and rejects unsupported types and sizes.
func (r *UploadRequest) Validate() error {
	r.Filename = strings.TrimSpace(r.Filename)
	if r.Filename == "" || utf8.RuneCountInString(r.Filename) > maxFilenameLength || !utf8.ValidString(r.Filename) {
		return ErrInvalid
	}
	if _, ok := extensions[r.ContentType]; !ok {
		return ErrInvalid
	}
	if r.Size <= 0 || r.Size > MaxSize {
		return ErrInvalid
	}
	return nil
}

type Store struct{ db *gorm.DB }

func NewStore(db *gorm.DB) *Store { return &Store{db: db} }

const columns = `id, content_id, uploader_id, object_key, filename, content_type, size_bytes, status, created_at`

// CreatePending records an upload for a live content; the object key is
// media/<id><ext>, so names never collide or reveal the original filename.
func (s *Store) CreatePending(ctx context.Context, contentID, uploaderID string, r UploadRequest) (Media, error) {
	var items []Media
	err := s.db.WithContext(ctx).Raw(`
WITH new AS (SELECT gen_random_uuid() AS id)
INSERT INTO media (id, content_id, uploader_id, object_key, filename, content_type, size_bytes)
SELECT new.id, ?, ?, 'media/' || new.id || ?, ?, ?, ?
FROM new
WHERE EXISTS (SELECT 1 FROM contents WHERE id = ? AND deleted_at IS NULL)
RETURNING `+columns,
		contentID, uploaderID, extensions[r.ContentType], r.Filename, r.ContentType, r.Size, contentID,
	).Scan(&items).Error
	if err != nil {
		return Media{}, err
	}
	if len(items) == 0 {
		return Media{}, ErrNotFound
	}
	return items[0], nil
}

func (s *Store) Get(ctx context.Context, id string) (Media, error) {
	var items []Media
	err := s.db.WithContext(ctx).Raw(`SELECT `+columns+` FROM media WHERE id = ? AND deleted_at IS NULL`, id).Scan(&items).Error
	if err != nil {
		return Media{}, err
	}
	if len(items) == 0 {
		return Media{}, ErrNotFound
	}
	return items[0], nil
}

func (s *Store) MarkReady(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Exec(`UPDATE media SET status = 'ready' WHERE id = ? AND deleted_at IS NULL`, id).Error
}

// ListReady returns uploaded videos of a content, oldest first.
func (s *Store) ListReady(ctx context.Context, contentID string) ([]Media, error) {
	items := []Media{}
	err := s.db.WithContext(ctx).Raw(`SELECT `+columns+` FROM media
WHERE content_id = ? AND status = 'ready' AND deleted_at IS NULL ORDER BY created_at`, contentID).Scan(&items).Error
	return items, err
}

// ListGallery returns ready videos of live contents, newest first.
func (s *Store) ListGallery(ctx context.Context) ([]GalleryItem, error) {
	// GORM leaves embedded structs empty on Raw scans, so scan flat rows first.
	var rows []struct {
		ID, ContentID, UploaderID, ObjectKey, Filename, ContentType, Status string
		SizeBytes                                                           int64
		CreatedAt                                                           time.Time
		ContentTitle, UploaderName                                          string
	}
	err := s.db.WithContext(ctx).Raw(`
SELECT m.id, m.content_id, m.uploader_id, m.object_key, m.filename, m.content_type, m.size_bytes,
  m.status, m.created_at, c.title AS content_title, u.display_name AS uploader_name
FROM media m
JOIN contents c ON c.id = m.content_id AND c.deleted_at IS NULL
JOIN users u ON u.id = m.uploader_id
WHERE m.status = 'ready' AND m.deleted_at IS NULL
ORDER BY m.created_at DESC
LIMIT ?`, galleryLimit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	items := make([]GalleryItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, GalleryItem{
			Media: Media{ID: r.ID, ContentID: r.ContentID, UploaderID: r.UploaderID, ObjectKey: r.ObjectKey,
				Filename: r.Filename, ContentType: r.ContentType, SizeBytes: r.SizeBytes, Status: r.Status, CreatedAt: r.CreatedAt},
			ContentTitle: r.ContentTitle,
			UploaderName: r.UploaderName,
		})
	}
	return items, nil
}

func (s *Store) SoftDelete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Exec(`UPDATE media SET deleted_at = now() WHERE id = ? AND deleted_at IS NULL`, id).Error
}
