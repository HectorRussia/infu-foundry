// Package auth verifies email/password logins and stores server-side sessions.
// There is no sign-up: accounts are created with cmd/createuser.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	SessionLifetime = 30 * 24 * time.Hour
	bcryptCost      = 12
	MinPassword     = 10
	maxPassword     = 72 // bcrypt only uses the first 72 bytes
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrNoSession          = errors.New("no valid session")
	ErrWeakPassword       = errors.New("password must be 10-72 bytes")
)

// dummyHash keeps login timing similar whether or not the email exists.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer-password"), bcryptCost)

type User struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

type Store struct{ db *gorm.DB }

func NewStore(db *gorm.DB) *Store { return &Store{db: db} }

func HashPassword(password string) (string, error) {
	if len(password) < MinPassword || len(password) > maxPassword {
		return "", ErrWeakPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(hash), err
}

// Login returns the user and a new opaque session token for the cookie.
func (s *Store) Login(ctx context.Context, email, password string) (User, string, error) {
	var rows []struct {
		ID           string
		Email        string
		DisplayName  string
		PasswordHash *string
	}
	err := s.db.WithContext(ctx).Raw(
		`SELECT id, email, display_name, password_hash FROM users WHERE lower(email) = lower(?)`,
		strings.TrimSpace(email),
	).Scan(&rows).Error
	if err != nil {
		return User{}, "", err
	}
	if len(rows) == 0 || rows[0].PasswordHash == nil {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return User{}, "", ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(*rows[0].PasswordHash), []byte(password)) != nil {
		return User{}, "", ErrInvalidCredentials
	}
	token, err := s.createSession(ctx, rows[0].ID)
	if err != nil {
		return User{}, "", err
	}
	return User{ID: rows[0].ID, Email: rows[0].Email, DisplayName: rows[0].DisplayName}, token, nil
}

func (s *Store) createSession(ctx context.Context, userID string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	err := s.db.WithContext(ctx).Exec(
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (decode(?, 'hex'), ?, ?)`,
		hashToken(token), userID, time.Now().Add(SessionLifetime),
	).Error
	return token, err
}

func (s *Store) UserBySession(ctx context.Context, token string) (User, error) {
	if token == "" {
		return User{}, ErrNoSession
	}
	var users []User
	err := s.db.WithContext(ctx).Raw(`
SELECT u.id, u.email, u.display_name
FROM sessions s JOIN users u ON u.id = s.user_id
WHERE s.token_hash = decode(?, 'hex') AND s.expires_at > now()`, hashToken(token)).Scan(&users).Error
	if err != nil {
		return User{}, err
	}
	if len(users) == 0 {
		return User{}, ErrNoSession
	}
	return users[0], nil
}

// Logout deletes this session and any expired ones.
func (s *Store) Logout(ctx context.Context, token string) error {
	return s.db.WithContext(ctx).Exec(
		`DELETE FROM sessions WHERE token_hash = decode(?, 'hex') OR expires_at <= now()`, hashToken(token),
	).Error
}

// hashToken returns hex because GORM expands []byte arguments like a list.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
