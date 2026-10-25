package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"infu-foundry/apps/api/internal/contents"
)

const (
	testAuthorID  = "11111111-1111-1111-1111-111111111111"
	testContentID = "22222222-2222-2222-2222-222222222222"
)

type fakeStore struct {
	items      map[string]contents.Content
	authorSeed bool
}

func newFakeStore() *fakeStore {
	return &fakeStore{items: map[string]contents.Content{}, authorSeed: true}
}

func (f *fakeStore) List(context.Context) ([]contents.Summary, error) {
	items := []contents.Summary{}
	for _, c := range f.items {
		items = append(items, contents.Summary{ID: c.ID, AuthorName: c.AuthorName, Title: c.Title, Excerpt: c.Body, UpdatedAt: c.UpdatedAt})
	}
	return items, nil
}

func (f *fakeStore) Get(_ context.Context, id string) (contents.Content, error) {
	c, ok := f.items[id]
	if !ok {
		return contents.Content{}, contents.ErrNotFound
	}
	return c, nil
}

func (f *fakeStore) Create(_ context.Context, authorID string, in contents.Input) (contents.Content, error) {
	c := contents.Content{ID: testContentID, AuthorID: authorID, AuthorName: "Dev", Title: in.Title, Body: in.Body, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	f.items[c.ID] = c
	return c, nil
}

func (f *fakeStore) Update(_ context.Context, id string, in contents.Input) (contents.Content, error) {
	c, ok := f.items[id]
	if !ok {
		return contents.Content{}, contents.ErrNotFound
	}
	c.Title, c.Body = in.Title, in.Body
	f.items[id] = c
	return c, nil
}

func (f *fakeStore) Delete(_ context.Context, id string) error {
	if _, ok := f.items[id]; !ok {
		return contents.ErrNotFound
	}
	delete(f.items, id)
	return nil
}

func (f *fakeStore) UserIDByEmail(_ context.Context, email string) (string, error) {
	if !f.authorSeed || email != DevAuthorEmail {
		return "", contents.ErrNotFound
	}
	return testAuthorID, nil
}

func TestContentRoutes(t *testing.T) {
	store := newFakeStore()
	app := New(slog.New(slog.NewTextHandler(io.Discard, nil)), true, store)
	do := func(method, path, body string, want int) string {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if body != "" {
			r.Header.Set("Content-Type", "application/json")
		}
		w, err := app.Test(r)
		if err != nil {
			t.Fatal(err)
		}
		defer w.Body.Close()
		out, _ := io.ReadAll(w.Body)
		if w.StatusCode != want {
			t.Fatalf("%s %s: status %d, want %d (%s)", method, path, w.StatusCode, want, out)
		}
		if w.Header.Get("Cache-Control") != "no-store" && want < 400 {
			t.Fatalf("%s %s must not be cached", method, path)
		}
		return string(out)
	}

	created := do("POST", "/api/v1/contents", `{"title":"  หัวข้อ  ","body":"เนื้อหา"}`, 201)
	var item contents.Content
	if err := json.Unmarshal([]byte(created), &item); err != nil || item.Title != "หัวข้อ" || item.AuthorID != testAuthorID {
		t.Fatalf("unexpected created content %s", created)
	}
	do("GET", "/api/v1/contents", "", 200)
	do("GET", "/api/v1/contents/"+testContentID, "", 200)
	do("PUT", "/api/v1/contents/"+testContentID, `{"title":"ใหม่","body":"แก้แล้ว"}`, 200)
	do("DELETE", "/api/v1/contents/"+testContentID, "", 204)
	do("GET", "/api/v1/contents/"+testContentID, "", 404)

	do("GET", "/api/v1/contents/not-a-uuid", "", 404)
	do("POST", "/api/v1/contents", `{"title":"","body":"   "}`, 422)
	do("POST", "/api/v1/contents", `{"title":"`+strings.Repeat("ก", 301)+`","body":"x"}`, 422)
	do("POST", "/api/v1/contents", `{bad json`, 400)

	store.authorSeed = false
	do("POST", "/api/v1/contents", `{"title":"x","body":"y"}`, 503)
}

func TestContentRoutesHiddenOutsideDevelopment(t *testing.T) {
	app := New(slog.New(slog.NewTextHandler(io.Discard, nil)), false, newFakeStore())
	w, err := app.Test(httptest.NewRequest("GET", "/api/v1/contents", nil))
	if err != nil {
		t.Fatal(err)
	}
	if w.StatusCode != 404 {
		t.Fatalf("production must not expose content routes, got %d", w.StatusCode)
	}
}
