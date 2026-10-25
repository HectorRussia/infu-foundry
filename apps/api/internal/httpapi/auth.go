package httpapi

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"infu-foundry/apps/api/internal/auth"
)

const (
	sessionCookie    = "infu_session"
	maxLoginFailures = 10
	loginWindow      = 15 * time.Minute
)

type AuthStore interface {
	Login(ctx context.Context, email, password string) (auth.User, string, error)
	UserBySession(ctx context.Context, token string) (auth.User, error)
	Logout(ctx context.Context, token string) error
}

// loginLimiter blocks an email after repeated failures. In-memory is enough for a
// single API instance; it resets on restart.
type loginLimiter struct {
	mu       sync.Mutex
	failures map[string][]time.Time
}

func (l *loginLimiter) recent(email string, now time.Time) []time.Time {
	kept := l.failures[email][:0]
	for _, t := range l.failures[email] {
		if now.Sub(t) < loginWindow {
			kept = append(kept, t)
		}
	}
	l.failures[email] = kept
	return kept
}

func (l *loginLimiter) blocked(email string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.recent(email, time.Now())) >= maxLoginFailures
}

func (l *loginLimiter) fail(email string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	l.failures[email] = append(l.recent(email, now), now)
}

func (l *loginLimiter) reset(email string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, email)
}

func currentUser(c fiber.Ctx) auth.User {
	user, _ := c.Locals("user").(auth.User)
	return user
}

// requireSession rejects requests without a valid session cookie.
func requireSession(store AuthStore) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, err := store.UserBySession(c.Context(), c.Cookies(sessionCookie))
		if errors.Is(err, auth.ErrNoSession) {
			return fiber.ErrUnauthorized
		}
		if err != nil {
			return err
		}
		c.Locals("user", user)
		return c.Next()
	}
}

// requireJSON blocks form posts from other sites: a cross-site JSON request needs a
// CORS preflight, which only the allowed development origin passes.
func requireJSON(c fiber.Ctx) error {
	if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead && c.Method() != fiber.MethodDelete && !c.Is("json") {
		return fiber.ErrUnsupportedMediaType
	}
	return c.Next()
}

// requireGatewaySecret makes the API reachable only through the Cloudflare Worker.
func requireGatewaySecret(secret string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if c.Path() == "/api/v1/health" {
			return c.Next()
		}
		if subtle.ConstantTimeCompare([]byte(c.Get("X-Gateway-Secret")), []byte(secret)) != 1 {
			return fiber.ErrNotFound
		}
		return c.Next()
	}
}

func registerAuth(router fiber.Router, store AuthStore, secureCookie bool) {
	limiter := &loginLimiter{failures: map[string][]time.Time{}}
	setCookie := func(c fiber.Ctx, value string, expires time.Time) {
		c.Cookie(&fiber.Cookie{
			Name:     sessionCookie,
			Value:    value,
			Path:     "/api",
			Expires:  expires,
			HTTPOnly: true,
			Secure:   secureCookie,
			SameSite: fiber.CookieSameSiteLaxMode,
		})
	}

	group := router.Group("/auth", func(c fiber.Ctx) error {
		c.Set("Cache-Control", "no-store")
		return c.Next()
	}, requireJSON)
	group.Post("/login", func(c fiber.Ctx) error {
		var in struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := c.Bind().JSON(&in); err != nil {
			return fiber.ErrBadRequest
		}
		email := strings.ToLower(strings.TrimSpace(in.Email))
		if email == "" || in.Password == "" {
			return fiber.ErrUnauthorized
		}
		if limiter.blocked(email) {
			return fiber.ErrTooManyRequests
		}
		user, token, err := store.Login(c.Context(), email, in.Password)
		if errors.Is(err, auth.ErrInvalidCredentials) {
			limiter.fail(email)
			return fiber.ErrUnauthorized
		}
		if err != nil {
			return err
		}
		limiter.reset(email)
		setCookie(c, token, time.Now().Add(auth.SessionLifetime))
		return c.JSON(user)
	})
	group.Post("/logout", func(c fiber.Ctx) error {
		if token := c.Cookies(sessionCookie); token != "" {
			if err := store.Logout(c.Context(), token); err != nil {
				return err
			}
		}
		setCookie(c, "", time.Unix(0, 0))
		return c.SendStatus(fiber.StatusNoContent)
	})
	group.Get("/me", requireSession(store), func(c fiber.Ctx) error {
		return c.JSON(currentUser(c))
	})
}
