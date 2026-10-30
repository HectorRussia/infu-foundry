// Package contents stores the shared content library: every signed-in user sees
// every non-deleted content; author_id only records who wrote it.
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
}

type Summary struct {
	ID         string    `json:"id"`
	AuthorName string    `json:"authorName"`
	VideoCount int       `json:"videoCount"`
	Title      string    `json:"title"`
	Excerpt    string    `json:"excerpt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Input struct {
	Title string `json:"title"`
	Body  string `json:"body"`
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
	return nil
}

type Store struct{ db *gorm.DB }

func NewStore(db *gorm.DB) *Store { return &Store{db: db} }

const selectContent = `
SELECT c.id, c.author_id, u.display_name AS author_name, c.title, c.body, c.created_at, c.updated_at
FROM contents c JOIN users u ON u.id = c.author_id
WHERE c.id = ? AND c.deleted_at IS NULL`

func (s *Store) List(ctx context.Context) ([]Summary, error) {
	items := []Summary{}
	err := s.db.WithContext(ctx).Raw(`
SELECT c.id, u.display_name AS author_name, c.title, left(c.body, ?) AS excerpt, c.updated_at,
  (SELECT count(*) FROM media m
   WHERE m.content_id = c.id AND m.status = 'ready' AND m.deleted_at IS NULL) AS video_count
FROM contents c JOIN users u ON u.id = c.author_id
WHERE c.deleted_at IS NULL
ORDER BY c.updated_at DESC
LIMIT ?`, excerptLength, listLimit).Scan(&items).Error
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

func (s *Store) Create(ctx context.Context, authorID string, in Input) (Content, error) {
	var id string
	err := s.db.WithContext(ctx).Raw(
		`INSERT INTO contents (author_id, title, body) VALUES (?, ?, ?) RETURNING id`,
		authorID, in.Title, in.Body,
	).Scan(&id).Error
	if err != nil {
		return Content{}, err
	}
	return s.Get(ctx, id)
}

func (s *Store) Update(ctx context.Context, id string, in Input) (Content, error) {
	result := s.db.WithContext(ctx).Exec(
		`UPDATE contents SET title = ?, body = ?, updated_at = now() WHERE id = ? AND deleted_at IS NULL`,
		in.Title, in.Body, id,
	)
	if result.Error != nil {
		return Content{}, result.Error
	}
	if result.RowsAffected == 0 {
		return Content{}, ErrNotFound
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
