package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestHTTPBoundary(t *testing.T) {
	cases := []struct {
		name, method, path, origin string
		development                bool
		wantStatus                 int
		wantCORS                   string
	}{
		{"health", "GET", "/api/v1/health", "", true, 200, ""},
		{"head", "HEAD", "/api/v1/health", "", true, 200, ""},
		{"case sensitive", "GET", "/api/v1/HEALTH", "", true, 404, ""},
		{"strict path", "GET", "/api/v1/health/", "", true, 404, ""},
		{"allowed local origin", "GET", "/api/v1/health", "http://localhost:3000", true, 200, "http://localhost:3000"},
		{"foreign origin", "GET", "/api/v1/health", "https://example.com", true, 403, ""},
		{"preflight", "OPTIONS", "/api/v1/health", "http://localhost:3000", true, 204, "http://localhost:3000"},
		{"unknown API", "GET", "/api/missing", "", true, 404, ""},
		{"unsupported method", "POST", "/api/v1/health", "", true, 405, ""},
		{"production no local CORS", "GET", "/api/v1/health", "http://localhost:3000", false, 200, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if tc.method == http.MethodOptions {
				r.Header.Set("Access-Control-Request-Method", "GET")
			}
			app := New(slog.New(slog.NewTextHandler(io.Discard, nil)), tc.development)
			w, err := app.Test(r)
			if err != nil {
				t.Fatal(err)
			}
			defer w.Body.Close()
			if w.StatusCode != tc.wantStatus {
				t.Fatalf("status %d, want %d", w.StatusCode, tc.wantStatus)
			}
			if got := w.Header.Get("Access-Control-Allow-Origin"); got != tc.wantCORS {
				t.Fatalf("CORS %q, want %q", got, tc.wantCORS)
			}
			if tc.wantStatus == http.StatusOK {
				if w.Header.Get("Cache-Control") != "no-store" {
					t.Fatal("health must not be cached")
				}
				body, _ := io.ReadAll(w.Body)
				if tc.method == http.MethodHead {
					if len(body) != 0 {
						t.Fatal("HEAD must have no body")
					}
					return
				}
				var health map[string]string
				if err := json.Unmarshal(body, &health); err != nil || health["status"] != "ok" {
					t.Fatalf("unexpected health response %s", body)
				}
			}
		})
	}
}

func TestPanicIsSanitizedAndLoggedAsFailure(t *testing.T) {
	var logs bytes.Buffer
	app := New(slog.New(slog.NewJSONHandler(&logs, nil)), false)
	app.Get("/panic", func(c fiber.Ctx) error { panic("secret-password") })
	response, err := app.Test(httptest.NewRequest("GET", "/panic?token=private", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != 500 || strings.Contains(string(body)+logs.String(), "secret-password") || strings.Contains(logs.String(), "token=") {
		t.Fatalf("unsafe error handling: %d %s", response.StatusCode, body)
	}
	if !strings.Contains(logs.String(), `"status":500`) {
		t.Fatal("panic must be logged with its actual status")
	}
}
