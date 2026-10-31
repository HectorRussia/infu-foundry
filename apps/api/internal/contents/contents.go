// Package contents stores the shared content library: every signed-in user sees
// every non-deleted content; author_id only records who wrote it.
// A content belongs to at most one shared category, stored in content_categories
// rather than on contents, so a later many-to-many tag model does not rewrite
// the contents table.
// Pin and manual order are shared too. Pinned rows stay above the rest, and
// sort_key orders each group. Editing a content does not change that order.
package contents

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
)

const (
	maxTitleLength = 300
	listLimit      = 200
	excerptLength  = 160
)

var (
	ErrNotFound = errors.New("content not found")
	ErrInvalid  = errors.New("invalid content")
)

type Content struct {
	ID         string    `json:"id"`
	AuthorID   string    `json:"authorId"`
	AuthorName string    `json:"authorName"`
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	// CoverKey is the R2 object of the cover image ("" = none); the handler turns
	// it into a short-lived CoverURL and never exposes the key itself.
	CoverKey string `json:"-"`
	CoverURL string `json:"coverUrl,omitempty"`
	// CoverDownloadURL saves the cover on phones (Content-Disposition: attachment).
	CoverDownloadURL string `json:"coverDownloadUrl,omitempty"`
	CoverPositionX   int    `json:"coverPositionX"`
	CoverPositionY   int    `json:"coverPositionY"`
	// CategoryID is empty when the content has no category. The name is joined
	// live so a rename shows up without rewriting contents.
	CategoryID   string `json:"categoryId,omitempty"`
	CategoryName string `json:"categoryName,omitempty"`
	// Pinned contents are listed before the others. The flag is shared.
	Pinned bool `json:"pinned"`
	// SortKey is the gap-based library position. It is not part of the API.
	SortKey int64 `json:"-"`
}

type Summary struct {
	ID         string    `json:"id"`
	AuthorName string    `json:"authorName"`
	VideoCount int       `json:"videoCount"`
	Title      string    `json:"title"`
	Excerpt    string    `json:"excerpt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	CoverKey   string    `json:"-"`
	CoverURL   string    `json:"coverUrl,omitempty"`
	// CoverPositionX/Y are CSS object-position percentages for cropping the cover.
	CoverPositionX int    `json:"coverPositionX"`
	CoverPositionY int    `json:"coverPositionY"`
	CategoryID     string `json:"categoryId,omitempty"`
	CategoryName   string `json:"categoryName,omitempty"`
	Pinned         bool   `json:"pinned"`
	// SortKey orders the library. Callers that already have SQL ORDER BY can leave it zero.
	SortKey int64 `json:"-"`
}

// Uncategorized is the list filter for contents with no category.
const Uncategorized = "uncategorized"

// ListFilter selects which contents List returns. An empty Category is every
// live content. Uncategorized is those with no link. Anything else is a category id.
type ListFilter struct {
	Category string
}

type Input struct {
	Title    string     `json:"title"`
	Body     string     `json:"body"`
	Category CategoryID `json:"categoryId"`
}

// Validate trims the title and rejects empty or oversized content.
func (in *Input) Validate() error {
	in.Title = strings.TrimSpace(in.Title)
	if utf8.RuneCountInString(in.Title) > maxTitleLength {
		return ErrInvalid
	}
	if strings.TrimSpace(in.Body) == "" && in.Title == "" {
		return ErrInvalid
	}
	if !utf8.ValidString(in.Title) || !utf8.ValidString(in.Body) || strings.ContainsRune(in.Body, 0) {
		return ErrInvalid
	}
	// Omitted and JSON null are valid. Anything else must be a category UUID;
	// whether that category exists is checked when the row is written.
	if in.Category.Set && !in.Category.Null && !categoryIDPattern.MatchString(in.Category.Value) {
		return ErrInvalid
	}
	return nil
}

type Store struct{ db *gorm.DB }

func NewStore(db *gorm.DB) *Store { return &Store{db: db} }

const selectContent = `
SELECT c.id, c.author_id, u.display_name AS author_name, c.title, c.body, c.created_at, c.updated_at,
  coalesce(c.cover_key, '') AS cover_key, c.cover_position_x, c.cover_position_y,
  coalesce(CAST(cat.id AS text), '') AS category_id, coalesce(cat.name, '') AS category_name,
  c.pinned, c.sort_key
FROM contents c
JOIN users u ON u.id = c.author_id
LEFT JOIN content_categories cc ON cc.content_id = c.id
LEFT JOIN categories cat ON cat.id = cc.category_id
WHERE c.id = ? AND c.deleted_at IS NULL`

func (s *Store) List(ctx context.Context, filter ListFilter) ([]Summary, error) {
	items := []Summary{}
	// LEFT JOIN so contents with no category stay in the list. content_id is the
	// primary key of the link, so the join never duplicates a content.
	query := `
SELECT c.id, u.display_name AS author_name, c.title, left(c.body, ?) AS excerpt, c.updated_at,
  coalesce(c.cover_key, '') AS cover_key, c.cover_position_x, c.cover_position_y,
  coalesce(CAST(cat.id AS text), '') AS category_id, coalesce(cat.name, '') AS category_name,
  c.pinned, c.sort_key,
  (SELECT count(*) FROM media m
   WHERE m.content_id = c.id AND m.status = 'ready' AND m.deleted_at IS NULL) AS video_count
FROM contents c
JOIN users u ON u.id = c.author_id
LEFT JOIN content_categories cc ON cc.content_id = c.id
LEFT JOIN categories cat ON cat.id = cc.category_id
WHERE c.deleted_at IS NULL`
	args := []any{excerptLength}
	switch filter.Category {
	case "":
	case Uncategorized:
		query += ` AND cc.content_id IS NULL`
	default:
		query += ` AND cat.id = ?`
		args = append(args, filter.Category)
	}
	// A category filter only removes rows. Pinned still comes first, and the
	// saved order of whatever remains is unchanged.
	query += `
ORDER BY c.pinned DESC, c.sort_key ASC, c.id ASC
LIMIT ?`
	args = append(args, listLimit)
	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&items).Error
	return items, err
}

func (s *Store) Get(ctx context.Context, id string) (Content, error) {
	var items []Content
	if err := s.db.WithContext(ctx).Raw(selectContent, id).Scan(&items).Error; err != nil {
		return Content{}, err
	}
	if len(items) == 0 {
		return Content{}, ErrNotFound
	}
	return items[0], nil
}

// SetCover replaces the cover key ("" removes it), recentres the crop for the new
// image and returns the previous key so the caller can delete the old image.
func (s *Store) SetCover(ctx context.Context, id, key string) (string, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return "", err
	}
	var value any
	if key != "" {
		value = key
	}
	result := s.db.WithContext(ctx).Exec(
		`UPDATE contents SET cover_key = ?, cover_position_x = 50, cover_position_y = 50
WHERE id = ? AND deleted_at IS NULL`, value, id,
	)
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected == 0 {
		return "", ErrNotFound
	}
	return current.CoverKey, nil
}

// SetCoverPosition stores which part of the cover stays visible when cropped.
func (s *Store) SetCoverPosition(ctx context.Context, id string, x, y int) error {
	if !ValidCoverPosition(x, y) {
		return ErrInvalid
	}
	result := s.db.WithContext(ctx).Exec(
		`UPDATE contents SET cover_position_x = ?, cover_position_y = ? WHERE id = ? AND deleted_at IS NULL`, x, y, id,
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ValidCoverPosition matches the contents_cover_position_range check.
func ValidCoverPosition(x, y int) bool {
	return x >= 0 && x <= 100 && y >= 0 && y <= 100
}

func (s *Store) Create(ctx context.Context, authorID string, in Input) (Content, error) {
	var id string
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// New content lands at the top of the unpinned group, under the pins.
		if err := lockLibraryOrder(tx); err != nil {
			return err
		}
		slots, err := groupSlots(tx, false)
		if err != nil {
			return err
		}
		key, updates, err := KeyAtTop(slots)
		if err != nil {
			return err
		}
		if err := applySortKeys(tx, slots, updates); err != nil {
			return err
		}
		if err := tx.Raw(
			`INSERT INTO contents (author_id, title, body, pinned, sort_key) VALUES (?, ?, ?, false, ?) RETURNING id`,
			authorID, in.Title, in.Body, key,
		).Scan(&id).Error; err != nil {
			return err
		}
		categoryID, set := in.Category.Specified()
		if !set || categoryID == "" {
			return nil
		}
		return assignCategory(tx, id, categoryID)
	})
	if err != nil {
		return Content{}, err
	}
	return s.Get(ctx, id)
}

func (s *Store) Update(ctx context.Context, id string, in Input) (Content, error) {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec(
			`UPDATE contents SET title = ?, body = ?, updated_at = now() WHERE id = ? AND deleted_at IS NULL`,
			in.Title, in.Body, id,
		)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
		}
		// An omitted categoryId leaves the link alone, including contents that
		// have never had a category.
		categoryID, set := in.Category.Specified()
		if !set {
			return nil
		}
		return assignCategory(tx, id, categoryID)
	})
	if err != nil {
		return Content{}, err
	}
	return s.Get(ctx, id)
}

// Delete is a soft delete so mistakes can be recovered from the database.
func (s *Store) Delete(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Exec(
		`UPDATE contents SET deleted_at = now() WHERE id = ? AND deleted_at IS NULL`, id,
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
