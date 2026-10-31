package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"infu-foundry/apps/api/internal/auth"
	"infu-foundry/apps/api/internal/contents"
)

const (
	testContentID = "22222222-2222-2222-2222-222222222222"
	testToken     = "valid-session-token"
	testPassword  = "correct-password"
)

var testUser = auth.User{ID: "11111111-1111-1111-1111-111111111111", Email: "horn@example.com", DisplayName: "Horn"}

type fakeStore struct {
	items      map[string]contents.Content
	categories map[string]contents.Category
	contentSeq int
	catSeq     int
}

func newFakeStore() *fakeStore {
	return &fakeStore{items: map[string]contents.Content{}, categories: map[string]contents.Category{}}
}

func (f *fakeStore) List(_ context.Context, filter contents.ListFilter) ([]contents.Summary, error) {
	items := []contents.Summary{}
	for _, c := range f.items {
		if filter.Category == contents.Uncategorized && c.CategoryID != "" {
			continue
		}
		if filter.Category != "" && filter.Category != contents.Uncategorized && c.CategoryID != filter.Category {
			continue
		}
		items = append(items, contents.Summary{
			ID: c.ID, AuthorName: c.AuthorName, Title: c.Title, Excerpt: c.Body, UpdatedAt: c.UpdatedAt,
			CoverKey: c.CoverKey, CoverPositionX: c.CoverPositionX, CoverPositionY: c.CoverPositionY,
			CategoryID: c.CategoryID, CategoryName: c.CategoryName,
			Pinned: c.Pinned, SortKey: c.SortKey,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return contents.Before(items[i].Pinned, items[i].SortKey, items[i].ID, items[j].Pinned, items[j].SortKey, items[j].ID)
	})
	return items, nil
}

func (f *fakeStore) Get(_ context.Context, id string) (contents.Content, error) {
	c, ok := f.items[id]
	if !ok {
		return contents.Content{}, contents.ErrNotFound
	}
	return c, nil
}

func (f *fakeStore) applyCategory(c *contents.Content, in contents.Input) error {
	id, set := in.Category.Specified()
	if !set {
		return nil
	}
	if id == "" {
		c.CategoryID, c.CategoryName = "", ""
		return nil
	}
	cat, ok := f.categories[id]
	if !ok {
		return contents.ErrCategoryNotFound
	}
	c.CategoryID, c.CategoryName = cat.ID, cat.Name
	return nil
}

func (f *fakeStore) slots(pinned bool) []contents.OrderSlot {
	slots := []contents.OrderSlot{}
	for _, item := range f.items {
		if item.Pinned == pinned {
			slots = append(slots, contents.OrderSlot{ID: item.ID, Key: item.SortKey})
		}
	}
	return slots
}

func (f *fakeStore) applyKeys(updates map[string]int64) {
	for id, key := range updates {
		item := f.items[id]
		item.SortKey = key
		f.items[id] = item
	}
}

func (f *fakeStore) Create(_ context.Context, authorID string, in contents.Input) (contents.Content, error) {
	f.contentSeq++
	id := testContentID
	if f.contentSeq > 1 {
		id = fmt.Sprintf("55555555-5555-5555-5555-%012d", f.contentSeq)
	}
	key, updates, err := contents.KeyAtTop(f.slots(false))
	if err != nil {
		f.contentSeq--
		return contents.Content{}, err
	}
	f.applyKeys(updates)
	c := contents.Content{
		ID: id, AuthorID: authorID, AuthorName: "Horn", Title: in.Title, Body: in.Body,
		CreatedAt: time.Now(), UpdatedAt: time.Now(), SortKey: key,
	}
	if err := f.applyCategory(&c, in); err != nil {
		f.contentSeq--
		return contents.Content{}, err
	}
	f.items[c.ID] = c
	return c, nil
}

func (f *fakeStore) Update(_ context.Context, id string, in contents.Input) (contents.Content, error) {
	c, ok := f.items[id]
	if !ok {
		return contents.Content{}, contents.ErrNotFound
	}
	c.Title, c.Body = in.Title, in.Body
	if err := f.applyCategory(&c, in); err != nil {
		return contents.Content{}, err
	}
	f.items[id] = c
	return c, nil
}

func (f *fakeStore) SetCover(_ context.Context, id, key string) (string, error) {
	c, ok := f.items[id]
	if !ok {
		return "", contents.ErrNotFound
	}
	old := c.CoverKey
	c.CoverKey, c.CoverPositionX, c.CoverPositionY = key, 50, 50
	f.items[id] = c
	return old, nil
}

func (f *fakeStore) SetCoverPosition(_ context.Context, id string, x, y int) error {
	c, ok := f.items[id]
	if !ok {
		return contents.ErrNotFound
	}
	c.CoverPositionX, c.CoverPositionY = x, y
	f.items[id] = c
	return nil
}

func (f *fakeStore) Delete(_ context.Context, id string) error {
	if _, ok := f.items[id]; !ok {
		return contents.ErrNotFound
	}
	delete(f.items, id)
	return nil
}

func (f *fakeStore) SetPinned(_ context.Context, id string, pinned bool) error {
	item, ok := f.items[id]
	if !ok {
		return contents.ErrNotFound
	}
	if item.Pinned == pinned {
		return nil
	}
	key, updates, err := contents.KeyAtTop(f.slots(pinned))
	if err != nil {
		return err
	}
	f.applyKeys(updates)
	item.Pinned = pinned
	item.SortKey = key
	f.items[id] = item
	return nil
}

func (f *fakeStore) Move(_ context.Context, id string, afterID *string) error {
	item, ok := f.items[id]
	if !ok {
		return contents.ErrNotFound
	}
	if afterID != nil {
		if *afterID == id {
			return contents.ErrInvalid
		}
		other, found := f.items[*afterID]
		if !found || other.Pinned != item.Pinned {
			return contents.ErrInvalid
		}
	}
	updates, err := contents.PlanMove(f.slots(item.Pinned), id, afterID)
	if err != nil {
		return err
	}
	f.applyKeys(updates)
	return nil
}

func (f *fakeStore) ListCategories(context.Context) ([]contents.Category, error) {
	items := []contents.Category{}
	for _, cat := range f.categories {
		cat.ContentCount = 0
		for _, item := range f.items {
			if item.CategoryID == cat.ID {
				cat.ContentCount++
			}
		}
		items = append(items, cat)
	}
	return items, nil
}

func (f *fakeStore) CreateCategory(_ context.Context, name string) (contents.Category, error) {
	for _, cat := range f.categories {
		if strings.EqualFold(strings.TrimSpace(cat.Name), name) {
			return contents.Category{}, contents.ErrCategoryExists
		}
	}
	f.catSeq++
	cat := contents.Category{
		ID:        fmt.Sprintf("33333333-3333-3333-3333-%012d", f.catSeq),
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	f.categories[cat.ID] = cat
	return cat, nil
}

func (f *fakeStore) RenameCategory(_ context.Context, id, name string) (contents.Category, error) {
	cat, ok := f.categories[id]
	if !ok {
		return contents.Category{}, contents.ErrCategoryNotFound
	}
	for _, other := range f.categories {
		if other.ID != id && strings.EqualFold(other.Name, name) {
			return contents.Category{}, contents.ErrCategoryExists
		}
	}
	cat.Name = name
	cat.UpdatedAt = time.Now()
	f.categories[id] = cat
	for key, item := range f.items {
		if item.CategoryID == id {
			item.CategoryName = name
			f.items[key] = item
		}
	}
	for _, item := range f.items {
		if item.CategoryID == id {
			cat.ContentCount++
		}
	}
	return cat, nil
}

// DeleteCategory mirrors ON DELETE CASCADE of the link only: the content stays
// and its category is cleared.
func (f *fakeStore) DeleteCategory(_ context.Context, id string) error {
	if _, ok := f.categories[id]; !ok {
		return contents.ErrCategoryNotFound
	}
	delete(f.categories, id)
	for key, item := range f.items {
		if item.CategoryID == id {
			item.CategoryID, item.CategoryName = "", ""
			f.items[key] = item
		}
	}
	return nil
}

type fakeAuth struct{ loggedOut bool }

func (f *fakeAuth) Login(_ context.Context, email, password string) (auth.User, string, error) {
	if email != testUser.Email || password != testPassword {
		return auth.User{}, "", auth.ErrInvalidCredentials
	}
	return testUser, testToken, nil
}

func (f *fakeAuth) UserBySession(_ context.Context, token string) (auth.User, error) {
	if token != testToken || f.loggedOut {
		return auth.User{}, auth.ErrNoSession
	}
	return testUser, nil
}

func (f *fakeAuth) Logout(context.Context, string) error {
	f.loggedOut = true
	return nil
}

type client struct {
	t   *testing.T
	app *fiber.App
}

func (cl client) do(method, path, body string, want int, headers ...string) *http.Response {
	cl.t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(headers); i += 2 {
		r.Header.Set(headers[i], headers[i+1])
	}
	w, err := cl.app.Test(r)
	if err != nil {
		cl.t.Fatal(err)
	}
	if w.StatusCode != want {
		out, _ := io.ReadAll(w.Body)
		cl.t.Fatalf("%s %s: status %d, want %d (%s)", method, path, w.StatusCode, want, out)
	}
	return w
}

func newClient(t *testing.T, opts Options) client {
	return client{t: t, app: New(slog.New(slog.NewTextHandler(io.Discard, nil)), opts)}
}

func TestContentRoutesRequireSession(t *testing.T) {
	cl := newClient(t, Options{Contents: newFakeStore(), Auth: &fakeAuth{}})
	session := []string{"Cookie", sessionCookie + "=" + testToken}

	cl.do("GET", "/api/v1/contents", "", 401)
	cl.do("POST", "/api/v1/contents", `{"title":"x","body":"y"}`, 401)
	cl.do("GET", "/api/v1/contents", "", 401, "Cookie", sessionCookie+"=forged")

	w := cl.do("POST", "/api/v1/contents", `{"title":"  หัวข้อ  ","body":"เนื้อหา"}`, 201, session...)
	var item contents.Content
	if err := json.NewDecoder(w.Body).Decode(&item); err != nil || item.Title != "หัวข้อ" || item.AuthorID != testUser.ID {
		t.Fatalf("author must be the session user, got %+v", item)
	}
	if w.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("content responses must not be cached")
	}
	cl.do("GET", "/api/v1/contents", "", 200, session...)
	cl.do("GET", "/api/v1/contents/"+testContentID, "", 200, session...)
	cl.do("PUT", "/api/v1/contents/"+testContentID, `{"title":"ใหม่","body":"แก้แล้ว"}`, 200, session...)
	cl.do("DELETE", "/api/v1/contents/"+testContentID, "", 204, session...)
	cl.do("GET", "/api/v1/contents/"+testContentID, "", 404, session...)

	cl.do("GET", "/api/v1/contents/not-a-uuid", "", 404, session...)
	cl.do("POST", "/api/v1/contents", `{"title":"","body":"   "}`, 422, session...)
	cl.do("POST", "/api/v1/contents", `{"title":"`+strings.Repeat("ก", 301)+`","body":"x"}`, 422, session...)
	cl.do("POST", "/api/v1/contents", `{bad json`, 400, session...)
	cl.do("POST", "/api/v1/contents", `title=x`, 415, append(session, "Content-Type", "application/x-www-form-urlencoded")...)
}

func TestLoginLogoutAndMe(t *testing.T) {
	fa := &fakeAuth{}
	cl := newClient(t, Options{Contents: newFakeStore(), Auth: fa})

	cl.do("GET", "/api/v1/auth/me", "", 401)
	cl.do("POST", "/api/v1/auth/login", `{"email":"horn@example.com","password":"wrong"}`, 401)
	w := cl.do("POST", "/api/v1/auth/login", `{"email":" Horn@Example.com ","password":"`+testPassword+`"}`, 200)
	cookie := w.Header.Get("Set-Cookie")
	for _, want := range []string{sessionCookie + "=" + testToken, "HttpOnly", "SameSite=Lax", "Path=/api", "Secure"} {
		if !strings.Contains(strings.ToLower(cookie), strings.ToLower(want)) {
			t.Fatalf("session cookie %q missing %q", cookie, want)
		}
	}
	session := []string{"Cookie", sessionCookie + "=" + testToken}
	cl.do("GET", "/api/v1/auth/me", "", 200, session...)
	cl.do("POST", "/api/v1/auth/logout", `{}`, 204, session...)
	cl.do("GET", "/api/v1/auth/me", "", 401, session...)
}

func TestLoginRateLimitPerEmail(t *testing.T) {
	cl := newClient(t, Options{Auth: &fakeAuth{}})
	for range maxLoginFailures {
		cl.do("POST", "/api/v1/auth/login", `{"email":"horn@example.com","password":"wrong"}`, 401)
	}
	cl.do("POST", "/api/v1/auth/login", `{"email":"horn@example.com","password":"`+testPassword+`"}`, 429)
	cl.do("POST", "/api/v1/auth/login", `{"email":"other@example.com","password":"wrong"}`, 401)
}

func TestDevelopmentCookieAllowsLocalHTTP(t *testing.T) {
	cl := newClient(t, Options{Development: true, Auth: &fakeAuth{}})
	w := cl.do("POST", "/api/v1/auth/login", `{"email":"horn@example.com","password":"`+testPassword+`"}`, 200)
	if strings.Contains(strings.ToLower(w.Header.Get("Set-Cookie")), "secure") {
		t.Fatal("development cookie must work on http://localhost")
	}
	pre := cl.do("OPTIONS", "/api/v1/auth/login", "", 204, "Origin", "http://localhost:3000", "Access-Control-Request-Method", "POST")
	if pre.Header.Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("local web must be allowed to send the session cookie")
	}
}

func TestGatewaySecret(t *testing.T) {
	cl := newClient(t, Options{GatewaySecret: "shh", Contents: newFakeStore(), Auth: &fakeAuth{}})
	cl.do("GET", "/api/v1/health", "", 200)
	cl.do("GET", "/api/v1/auth/me", "", 404)
	cl.do("GET", "/api/v1/auth/me", "", 404, "X-Gateway-Secret", "wrong")
	cl.do("GET", "/api/v1/auth/me", "", 401, "X-Gateway-Secret", "shh")
}

func TestNoDatabaseMeansNoAuthOrContentRoutes(t *testing.T) {
	cl := newClient(t, Options{})
	cl.do("GET", "/api/v1/contents", "", 404)
	cl.do("GET", "/api/v1/categories", "", 404)
	cl.do("POST", "/api/v1/categories", `{"name":"บทความ"}`, 404)
	cl.do("POST", "/api/v1/auth/login", `{"email":"a","password":"b"}`, 404)
}

func TestPinAndManualOrder(t *testing.T) {
	store := newFakeStore()
	cl := newClient(t, Options{Contents: store, Categories: store, Auth: &fakeAuth{}})
	session := []string{"Cookie", sessionCookie + "=" + testToken}
	missing := "44444444-4444-4444-4444-444444444444"

	cl.do("PATCH", "/api/v1/contents/"+testContentID+"/pin", `{"pinned":true}`, 401)
	cl.do("PATCH", "/api/v1/contents/"+testContentID+"/order", `{"afterId":null}`, 401)

	first := decodeContent(t, cl.do("POST", "/api/v1/contents", `{"title":"เก่า","body":"หนึ่ง"}`, 201, session...))
	second := decodeContent(t, cl.do("POST", "/api/v1/contents", `{"title":"ใหม่","body":"สอง"}`, 201, session...))
	if first.Pinned || second.Pinned {
		t.Fatal("new content starts unpinned")
	}
	if ids := listIDs(t, cl.do("GET", "/api/v1/contents", "", 200, session...)); len(ids) != 2 || ids[0] != second.ID || ids[1] != first.ID {
		t.Fatalf("new content must lead the unpinned group, got %v", ids)
	}

	cl.do("PATCH", "/api/v1/contents/"+second.ID+"/pin", `{}`, 422, session...)
	cl.do("PATCH", "/api/v1/contents/"+missing+"/pin", `{"pinned":true}`, 404, session...)
	cl.do("PATCH", "/api/v1/contents/not-a-uuid/pin", `{"pinned":true}`, 404, session...)
	cl.do("PATCH", "/api/v1/contents/"+second.ID+"/pin", `{"pinned":true}`, 204, session...)
	cl.do("PATCH", "/api/v1/contents/"+second.ID+"/pin", `{"pinned":true}`, 204, session...)

	third := decodeContent(t, cl.do("POST", "/api/v1/contents", `{"title":"ถัดไป","body":"สาม"}`, 201, session...))
	if ids := listIDs(t, cl.do("GET", "/api/v1/contents", "", 200, session...)); join(ids) != second.ID+","+third.ID+","+first.ID {
		t.Fatalf("pinned, then newest unpinned, got %v", ids)
	}

	cl.do("PATCH", "/api/v1/contents/"+first.ID+"/order", `{"afterId":null}`, 204, session...)
	if ids := listIDs(t, cl.do("GET", "/api/v1/contents", "", 200, session...)); join(ids) != second.ID+","+first.ID+","+third.ID {
		t.Fatalf("move to top of the unpinned group, got %v", ids)
	}
	cl.do("PATCH", "/api/v1/contents/"+second.ID+"/order", `{"afterId":"`+first.ID+`"}`, 422, session...)
	cl.do("PATCH", "/api/v1/contents/"+first.ID+"/order", `{"afterId":"nope"}`, 422, session...)
	cl.do("PATCH", "/api/v1/contents/"+first.ID+"/order", `{"afterId":"`+missing+`"}`, 422, session...)
	cl.do("PATCH", "/api/v1/contents/"+first.ID+"/order", `{"afterId":"`+first.ID+`"}`, 422, session...)
	cl.do("PATCH", "/api/v1/contents/"+missing+"/order", `{"afterId":null}`, 404, session...)

	cl.do("PATCH", "/api/v1/contents/"+first.ID+"/order", `{"afterId":"`+third.ID+`"}`, 204, session...)
	if ids := listIDs(t, cl.do("GET", "/api/v1/contents", "", 200, session...)); join(ids) != second.ID+","+third.ID+","+first.ID {
		t.Fatalf("move after the other unpinned content, got %v", ids)
	}
	// Already in that place: still 204 and the order does not change.
	cl.do("PATCH", "/api/v1/contents/"+first.ID+"/order", `{"afterId":"`+third.ID+`"}`, 204, session...)

	folder := decodeJSON[contents.Category](t, cl.do("POST", "/api/v1/categories", `{"name":"เรื่อง"}`, 201, session...))
	cl.do("PUT", "/api/v1/contents/"+third.ID, `{"title":"ถัดไป","body":"สาม","categoryId":"`+folder.ID+`"}`, 200, session...)
	cl.do("PUT", "/api/v1/contents/"+first.ID, `{"title":"เก่า","body":"หนึ่ง","categoryId":"`+folder.ID+`"}`, 200, session...)
	// Editing must not change the manual order. The filter hides the pinned row and keeps the rest.
	if ids := listIDs(t, cl.do("GET", "/api/v1/contents?category="+folder.ID, "", 200, session...)); join(ids) != third.ID+","+first.ID {
		t.Fatalf("category filter must keep relative order, got %v", ids)
	}
	var listed struct{ Items []contents.Summary }
	if err := json.NewDecoder(cl.do("GET", "/api/v1/contents", "", 200, session...).Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Items) != 3 || !listed.Items[0].Pinned || listed.Items[0].ID != second.ID || listed.Items[1].Pinned {
		t.Fatalf("list pin flags: %+v", listed.Items)
	}

	cl.do("PATCH", "/api/v1/contents/"+second.ID+"/pin", `{"pinned":false}`, 204, session...)
	if ids := listIDs(t, cl.do("GET", "/api/v1/contents", "", 200, session...)); ids[0] != second.ID {
		t.Fatalf("unpin puts the content at the top of the unpinned group, got %v", ids)
	}
}

func join(ids []string) string {
	return strings.Join(ids, ",")
}
