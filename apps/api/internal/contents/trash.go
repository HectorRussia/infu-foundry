package contents

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Trashed is a soft-deleted content shown in the trash. It can be restored, or
// purged: the row, its media rows and its R2 objects are removed for good.
type Trashed struct {
	ID             string    `json:"id"`
	AuthorName     string    `json:"authorName"`
	Title          string    `json:"title"`
	Excerpt        string    `json:"excerpt"`
	VideoCount     int       `json:"videoCount"`
	DeletedAt      time.Time `json:"deletedAt"`
	CoverKey       string    `json:"-"`
	CoverURL       string    `json:"coverUrl,omitempty"`
	CoverPositionX int       `json:"coverPositionX"`
	CoverPositionY int       `json:"coverPositionY"`
}

// ListTrash returns deleted contents, most recently deleted first. Nothing is
// purged automatically: items stay until someone restores or deletes them.
func (s *Store) ListTrash(ctx context.Context) ([]Trashed, error) {
	items := []Trashed{}
	err := s.db.WithContext(ctx).Raw(`
SELECT c.id, u.display_name AS author_name, c.title, left(c.body, ?) AS excerpt, c.deleted_at,
  coalesce(c.cover_key, '') AS cover_key, c.cover_position_x, c.cover_position_y,
  (SELECT count(*) FROM media m
   WHERE m.content_id = c.id AND m.status = 'ready' AND m.deleted_at IS NULL) AS video_count
FROM contents c JOIN users u ON u.id = c.author_id
WHERE c.deleted_at IS NOT NULL
ORDER BY c.deleted_at DESC
LIMIT ?`, excerptLength, listLimit).Scan(&items).Error
	return items, err
}

// Restore brings a deleted content back into the library.
func (s *Store) Restore(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Exec(
		`UPDATE contents SET deleted_at = NULL WHERE id = ? AND deleted_at IS NOT NULL`, id,
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// TrashObjectKeys lists every R2 object of a deleted content: its cover and all
// its videos (pending ones too). Deleting a key that is already gone is harmless.
func (s *Store) TrashObjectKeys(ctx context.Context, id string) ([]string, error) {
	var keys []string
	err := s.db.WithContext(ctx).Raw(`
SELECT c.cover_key FROM contents c WHERE c.id = ? AND c.deleted_at IS NOT NULL AND c.cover_key IS NOT NULL
UNION ALL
SELECT m.object_key FROM media m JOIN contents c ON c.id = m.content_id
WHERE c.id = ? AND c.deleted_at IS NOT NULL`, id, id).Scan(&keys).Error
	if err != nil {
		return nil, err
	}
	var found int64
	if err := s.db.WithContext(ctx).Raw(
		`SELECT count(*) FROM contents WHERE id = ? AND deleted_at IS NOT NULL`, id,
	).Scan(&found).Error; err != nil {
		return nil, err
	}
	if found == 0 {
		return nil, ErrNotFound
	}
	return keys, nil
}

// Purge permanently deletes a content that is already in the trash, with its
// media rows, in one transaction. Live contents are never touched.
func (s *Store) Purge(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			`DELETE FROM media WHERE content_id = ? AND EXISTS (
  SELECT 1 FROM contents WHERE id = ? AND deleted_at IS NOT NULL)`, id, id,
		).Error; err != nil {
			return err
		}
		result := tx.Exec(`DELETE FROM contents WHERE id = ? AND deleted_at IS NOT NULL`, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}
