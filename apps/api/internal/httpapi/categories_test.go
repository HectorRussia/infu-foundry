package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"infu-foundry/apps/api/internal/contents"
)

func TestCategoriesAndContentFilter(t *testing.T) {
	store := newFakeStore()
	cl := newClient(t, Options{Contents: store, Categories: store, Auth: &fakeAuth{}})
	session := []string{"Cookie", sessionCookie + "=" + testToken}

	cl.do("GET", "/api/v1/categories", "", 401)
	cl.do("POST", "/api/v1/categories", `{"name":"บทความ"}`, 401)
	cl.do("PATCH", "/api/v1/categories/"+testContentID, `{"name":"x"}`, 401)
	cl.do("DELETE", "/api/v1/categories/"+testContentID, "", 401)
	cl.do("POST", "/api/v1/categories", `{"name":"บทความ"}`, 415, append(session, "Content-Type", "text/plain")...)
	cl.do("POST", "/api/v1/categories", `{`, 400, session...)
	cl.do("POST", "/api/v1/categories", `{"name":"  "}`, 422, session...)
	cl.do("POST", "/api/v1/categories", `{"name":"`+strings.Repeat("ก", 81)+`"}`, 422, session...)
	cl.do("PATCH", "/api/v1/categories/not-a-uuid", `{"name":"x"}`, 404, session...)
	cl.do("DELETE", "/api/v1/categories/not-a-uuid", "", 404, session...)

	w := cl.do("POST", "/api/v1/categories", `{"name":"  บทความ  "}`, 201, session...)
	folder := decodeCategory(t, w)
	if folder.Name != "บทความ" || folder.ContentCount != 0 || folder.ID == "" {
		t.Fatalf("created category %+v", folder)
	}
	if w.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("category responses must not be cached")
	}
	cl.do("POST", "/api/v1/categories", `{"name":"บทความ"}`, 409, session...)
	cl.do("POST", "/api/v1/categories", `{"name":"  บทความ "}`, 409, session...)

	w = cl.do("POST", "/api/v1/categories", `{"name":"VIDEO"}`, 201, session...)
	other := decodeCategory(t, w)
	cl.do("POST", "/api/v1/categories", `{"name":"video"}`, 409, session...)

	w = cl.do("PATCH", "/api/v1/categories/"+folder.ID, `{"name":"บทความ"}`, 200, session...)
	if decodeCategory(t, w).Name != "บทความ" {
		t.Fatal("renaming to the same name must succeed")
	}
	cl.do("PATCH", "/api/v1/categories/"+other.ID, `{"name":"บทความ"}`, 409, session...)
	cl.do("PATCH", "/api/v1/categories/"+folder.ID, `{"name":" "}`, 422, session...)
	w = cl.do("PATCH", "/api/v1/categories/"+folder.ID, `{"name":"  เรื่องสั้น  "}`, 200, session...)
	if decodeCategory(t, w).Name != "เรื่องสั้น" {
		t.Fatal("rename must trim the name")
	}

	unknown := "99999999-9999-9999-9999-999999999999"
	cl.do("POST", "/api/v1/contents", `{"title":"x","body":"y","categoryId":"nope"}`, 422, session...)
	cl.do("POST", "/api/v1/contents", `{"title":"x","body":"y","categoryId":""}`, 422, session...)
	cl.do("POST", "/api/v1/contents", `{"title":"x","body":"y","categoryId":"`+unknown+`"}`, 422, session...)

	w = cl.do("POST", "/api/v1/contents", `{"title":"มีหมวด","body":"ต้นฉบับ","categoryId":"`+folder.ID+`"}`, 201, session...)
	item := decodeContent(t, w)
	if item.CategoryID != folder.ID || item.CategoryName != "เรื่องสั้น" || item.Title != "มีหมวด" {
		t.Fatalf("content must carry the category: %+v", item)
	}

	w = cl.do("PUT", "/api/v1/contents/"+item.ID, `{"title":"มีหมวด","body":"แก้โดยไม่ส่งหมวด"}`, 200, session...)
	item = decodeContent(t, w)
	if item.CategoryID != folder.ID || item.Body != "แก้โดยไม่ส่งหมวด" {
		t.Fatalf("omitted categoryId must keep the category: %+v", item)
	}
	w = cl.do("GET", "/api/v1/contents/"+item.ID, "", 200, session...)
	if decodeContent(t, w).CategoryName != "เรื่องสั้น" {
		t.Fatal("get must still return the category")
	}

	w = cl.do("PUT", "/api/v1/contents/"+item.ID, `{"title":"มีหมวด","body":"ล้าง","categoryId":null}`, 200, session...)
	cleared := readBody(t, w)
	if strings.Contains(cleared, "categoryId") || strings.Contains(cleared, "categoryName") {
		t.Fatalf("clearing the category must omit it: %s", cleared)
	}
	w = cl.do("PUT", "/api/v1/contents/"+item.ID, `{"title":"มีหมวด","body":"กลับ","categoryId":"`+folder.ID+`"}`, 200, session...)
	if decodeContent(t, w).CategoryID != folder.ID {
		t.Fatal("category must be assignable again")
	}
	cl.do("PUT", "/api/v1/contents/"+item.ID, `{"title":"ไม่ควรติด","body":"ไม่ควรติด","categoryId":"`+unknown+`"}`, 422, session...)
	w = cl.do("GET", "/api/v1/contents/"+item.ID, "", 200, session...)
	kept := decodeContent(t, w)
	if kept.CategoryID != folder.ID || kept.Title != "มีหมวด" || kept.Body != "กลับ" {
		t.Fatalf("a rejected category must not change the content: %+v", kept)
	}

	w = cl.do("POST", "/api/v1/contents", `{"title":"ลอย","body":"ไม่มีหมวด"}`, 201, session...)
	loose := decodeContent(t, w)
	if loose.CategoryID != "" {
		t.Fatalf("a content without categoryId must stay uncategorized: %+v", loose)
	}

	w = cl.do("GET", "/api/v1/categories", "", 200, session...)
	var listed struct{ Items []contents.Category }
	if err := json.Unmarshal([]byte(readBody(t, w)), &listed); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, cat := range listed.Items {
		counts[cat.Name] = cat.ContentCount
	}
	if counts["เรื่องสั้น"] != 1 || counts["VIDEO"] != 0 || len(listed.Items) != 2 {
		t.Fatalf("category counts: %+v", listed.Items)
	}

	all := listIDs(t, cl.do("GET", "/api/v1/contents", "", 200, session...))
	if len(all) != 2 {
		t.Fatalf("unfiltered list must keep every content, got %v", all)
	}
	inFolder := listIDs(t, cl.do("GET", "/api/v1/contents?category="+folder.ID, "", 200, session...))
	if len(inFolder) != 1 || inFolder[0] != item.ID {
		t.Fatalf("folder filter: %v", inFolder)
	}
	looseOnly := listIDs(t, cl.do("GET", "/api/v1/contents?category=uncategorized", "", 200, session...))
	if len(looseOnly) != 1 || looseOnly[0] != loose.ID {
		t.Fatalf("uncategorized filter: %v", looseOnly)
	}
	cl.do("GET", "/api/v1/contents?category=not-a-filter", "", 422, session...)
	empty := listIDs(t, cl.do("GET", "/api/v1/contents?category="+unknown, "", 200, session...))
	if len(empty) != 0 {
		t.Fatalf("an unknown category filter is an empty list, got %v", empty)
	}

	cl.do("DELETE", "/api/v1/categories/"+folder.ID, "", 204, session...)
	w = cl.do("GET", "/api/v1/contents/"+item.ID, "", 200, session...)
	survived := readBody(t, w)
	if strings.Contains(survived, "categoryId") || !strings.Contains(survived, "มีหมวด") {
		t.Fatalf("deleting a category must leave the content, uncategorized: %s", survived)
	}
	after := listIDs(t, cl.do("GET", "/api/v1/contents?category=uncategorized", "", 200, session...))
	if len(after) != 2 {
		t.Fatalf("both contents must be uncategorized after the folder is deleted, got %v", after)
	}
	if ids := listIDs(t, cl.do("GET", "/api/v1/contents?category="+folder.ID, "", 200, session...)); len(ids) != 0 {
		t.Fatalf("deleted category filter must be empty, got %v", ids)
	}
	w = cl.do("GET", "/api/v1/categories", "", 200, session...)
	if strings.Contains(readBody(t, w), folder.ID) {
		t.Fatal("deleted category must disappear from the list")
	}
	cl.do("DELETE", "/api/v1/categories/"+folder.ID, "", 404, session...)
	cl.do("PATCH", "/api/v1/categories/"+folder.ID, `{"name":"กลับมา"}`, 404, session...)

	// The other folder and the loose content are untouched.
	w = cl.do("GET", "/api/v1/contents/"+loose.ID, "", 200, session...)
	if decodeContent(t, w).Title != "ลอย" {
		t.Fatal("a content that was never categorized must still load")
	}
}

func decodeCategory(t *testing.T, w *http.Response) contents.Category {
	t.Helper()
	return decodeJSON[contents.Category](t, w)
}

func decodeContent(t *testing.T, w *http.Response) contents.Content {
	t.Helper()
	return decodeJSON[contents.Content](t, w)
}

func decodeJSON[T any](t *testing.T, w *http.Response) T {
	t.Helper()
	var item T
	if err := json.NewDecoder(w.Body).Decode(&item); err != nil {
		t.Fatal(err)
	}
	return item
}

func readBody(t *testing.T, w *http.Response) string {
	t.Helper()
	out, err := io.ReadAll(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func listIDs(t *testing.T, w *http.Response) []string {
	t.Helper()
	var body struct {
		Items []contents.Summary
	}
	if err := json.Unmarshal([]byte(readBody(t, w)), &body); err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(body.Items))
	for _, item := range body.Items {
		ids = append(ids, item.ID)
	}
	return ids
}
