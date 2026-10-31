package contents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

const maxCategoryName = 80

var (
	ErrCategoryNotFound = errors.New("category not found")
	ErrCategoryExists   = errors.New("category name taken")
	categoryIDPattern   = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

// Category is one shared folder. ContentCount is live contents only; trashed
// contents keep their link but are not counted here.
type Category struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	ContentCount int       `json:"contentCount"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// CategoryID is the category named by a content write.
// Omitted (Set false): leave the link unchanged on update, and leave a new
// content uncategorized. JSON null: clear the link. A UUID: assign that category.
type CategoryID struct {
	Set   bool
	Null  bool
	Value string
}

// Specified reports whether the JSON included categoryId.
// The id is empty when the value was null, which clears the category.
func (c CategoryID) Specified() (id string, ok bool) {
	if !c.Set {
		return "", false
	}
	return c.Value, true
}

func (c *CategoryID) UnmarshalJSON(data []byte) error {
	c.Set = true
	c.Null = false
	c.Value = ""
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		c.Null = true
		return nil
	}
	return json.Unmarshal(data, &c.Value)
}

type CategoryInput struct {
	Name string `json:"name"`
}

// Validate trims the name and rejects a blank or oversized one.
func (in *CategoryInput) Validate() error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || utf8.RuneCountInString(in.Name) > maxCategoryName {
		return ErrInvalid
	}
	if !utf8.ValidString(in.Name) || strings.ContainsRune(in.Name, 0) || strings.ContainsAny(in.Name, "\r\n") {
		return ErrInvalid
	}
	return nil
}

const selectCategory = `
SELECT cat.id, cat.name, cat.created_at, cat.updated_at,
  (SELECT count(*) FROM content_categories cc
   JOIN contents c ON c.id = cc.content_id AND c.deleted_at IS NULL
   WHERE cc.category_id = cat.id) AS content_count
FROM categories cat`

func (s *Store) ListCategories(ctx context.Context) ([]Category, error) {
	items := []Category{}
	err := s.db.WithContext(ctx).Raw(selectCategory + `
ORDER BY lower(cat.name), cat.id`).Scan(&items).Error
	return items, err
}

func (s *Store) categoryByID(ctx context.Context, id string) (Category, error) {
	var items []Category
	err := s.db.WithContext(ctx).Raw(selectCategory+`
WHERE cat.id = ?`, id).Scan(&items).Error
	if err != nil {
		return Category{}, err
	}
	if len(items) == 0 {
		return Category{}, ErrCategoryNotFound
	}
	return items[0], nil
}

func (s *Store) CreateCategory(ctx context.Context, name string) (Category, error) {
	var id string
	err := s.db.WithContext(ctx).Raw(
		`INSERT INTO categories (name) VALUES (?) RETURNING id`, name,
	).Scan(&id).Error
	if err != nil {
		return Category{}, mapCategoryWriteErr(err)
	}
	return s.categoryByID(ctx, id)
}

func (s *Store) RenameCategory(ctx context.Context, id, name string) (Category, error) {
	result := s.db.WithContext(ctx).Exec(
		`UPDATE categories SET name = ?, updated_at = now() WHERE id = ?`, name, id,
	)
	if result.Error != nil {
		return Category{}, mapCategoryWriteErr(result.Error)
	}
	if result.RowsAffected == 0 {
		return Category{}, ErrCategoryNotFound
	}
	return s.categoryByID(ctx, id)
}

// DeleteCategory removes the folder only. The database cascades that to the
// link rows, so contents stay in the library and become uncategorized.
func (s *Store) DeleteCategory(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Exec(`DELETE FROM categories WHERE id = ?`, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCategoryNotFound
	}
	return nil
}

// assignCategory sets or clears the single category of a content. An empty id
// clears it. The category must exist; the write is rolled back with the caller.
func assignCategory(tx *gorm.DB, contentID, categoryID string) error {
	if categoryID == "" {
		return tx.Exec(`DELETE FROM content_categories WHERE content_id = ?`, contentID).Error
	}
	var found int64
	if err := tx.Raw(`SELECT count(*) FROM categories WHERE id = ?`, categoryID).Scan(&found).Error; err != nil {
		return err
	}
	if found == 0 {
		return ErrCategoryNotFound
	}
	err := tx.Exec(`
INSERT INTO content_categories (content_id, category_id) VALUES (?, ?)
ON CONFLICT (content_id) DO UPDATE SET category_id = EXCLUDED.category_id`,
		contentID, categoryID,
	).Error
	return mapCategoryWriteErr(err)
}

func mapCategoryWriteErr(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "23505":
		return ErrCategoryExists
	case "23514":
		return ErrInvalid
	case "23503":
		return ErrCategoryNotFound
	default:
		return err
	}
}
