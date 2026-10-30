package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"infu-foundry/apps/api/internal/media"
)

const (
	testMediaID  = "33333333-3333-3333-3333-333333333333"
	otherContent = "44444444-4444-4444-4444-444444444444"
)

type fakeMedia struct{ items map[string]media.Media }

func (f *fakeMedia) CreatePending(_ context.Context, contentID, uploaderID string, r media.UploadRequest) (media.Media, error) {
	if contentID != testContentID {
		return media.Media{}, media.ErrNotFound
	}
	m := media.Media{ID: testMediaID, ContentID: contentID, UploaderID: uploaderID, ObjectKey: "media/" + testMediaID + ".mp4",
		Filename: r.Filename, ContentType: r.ContentType, SizeBytes: r.Size, Status: "pending"}
	f.items[m.ID] = m
	return m, nil
}

func (f *fakeMedia) Get(_ context.Context, id string) (media.Media, error) {
	m, ok := f.items[id]
	if !ok {
		return media.Media{}, media.ErrNotFound
	}
	return m, nil
}

func (f *fakeMedia) MarkReady(_ context.Context, id string) error {
	m := f.items[id]
	m.Status = "ready"
	f.items[id] = m
	return nil
}

func (f *fakeMedia) ListReady(_ context.Context, contentID string) ([]media.Media, error) {
	out := []media.Media{}
	for _, m := range f.items {
		if m.ContentID == contentID && m.Status == "ready" {
			out = append(out, m)
		}
	}
	return out, nil
}

func (f *fakeMedia) ListGallery(context.Context) ([]media.GalleryItem, error) {
	out := []media.GalleryItem{}
	for _, m := range f.items {
		if m.Status == "ready" {
			out = append(out, media.GalleryItem{Media: m, ContentTitle: "หัวข้อ", UploaderName: "Horn"})
		}
	}
	return out, nil
}

func (f *fakeMedia) SoftDelete(_ context.Context, id string) error {
	delete(f.items, id)
	return nil
}

type fakeStorage struct {
	objects   map[string]int64
	deleteErr error
}

func (f *fakeStorage) UploadURL(_ context.Context, key, _ string, _ int64) (string, error) {
	return "https://r2.example/" + key + "?upload", nil
}

func (f *fakeStorage) ViewURL(_ context.Context, key string) (string, error) {
	return "https://r2.example/" + key + "?view", nil
}

func (f *fakeStorage) DownloadURL(_ context.Context, key, filename string) (string, error) {
	return "https://r2.example/" + key + "?download=" + filename, nil
}

func (f *fakeStorage) Stat(_ context.Context, key string) (int64, string, error) {
	size, ok := f.objects[key]
	if !ok {
		return 0, "", media.ErrObjectMissing
	}
	return size, "video/mp4", nil
}

func (f *fakeStorage) Delete(_ context.Context, key string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	delete(f.objects, key)
	return nil
}

func TestMediaUploadFlow(t *testing.T) {
	store := &fakeMedia{items: map[string]media.Media{}}
	storage := &fakeStorage{objects: map[string]int64{}}
	cl := newClient(t, Options{Auth: &fakeAuth{}, Media: store, MediaStorage: storage})
	session := []string{"Cookie", sessionCookie + "=" + testToken}
	base := "/api/v1/contents/" + testContentID + "/media"
	upload := `{"filename":" clip.mp4 ","contentType":"video/mp4","size":3861420}`

	cl.do("POST", base, upload, 401)
	cl.do("POST", "/api/v1/contents/"+otherContent+"/media", upload, 404, session...)
	cl.do("POST", base, `{"filename":"x.mkv","contentType":"video/x-matroska","size":10}`, 422, session...)
	cl.do("POST", base, `{"filename":"big.mp4","contentType":"video/mp4","size":1073741825}`, 422, session...)
	cl.do("POST", base, `{"filename":"   ","contentType":"video/mp4","size":10}`, 422, session...)

	w := cl.do("POST", base, upload, 201, session...)
	var created struct {
		Media     map[string]any `json:"media"`
		UploadURL string         `json:"uploadUrl"`
	}
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Media["filename"] != "clip.mp4" || created.Media["uploaderId"] != testUser.ID || !strings.Contains(created.UploadURL, "upload") {
		t.Fatalf("unexpected upload response %+v", created)
	}
	if _, leaked := created.Media["objectKey"]; leaked {
		t.Fatal("object key must not be exposed")
	}

	complete := "/api/v1/media/" + testMediaID + "/complete"
	cl.do("POST", complete, `{}`, 409, session...) // not uploaded yet
	storage.objects["media/"+testMediaID+".mp4"] = 1
	cl.do("POST", complete, `{}`, 409, session...) // size mismatch
	storage.objects["media/"+testMediaID+".mp4"] = 3861420
	cl.do("POST", complete, `{}`, 200, session...)

	w = cl.do("GET", base, "", 200, session...)
	var list struct{ Items []map[string]any }
	if err := json.NewDecoder(w.Body).Decode(&list); err != nil || len(list.Items) != 1 || !strings.Contains(list.Items[0]["url"].(string), "view") {
		t.Fatalf("expected one ready video with a view URL, got %+v", list)
	}
	if dl, _ := list.Items[0]["downloadUrl"].(string); !strings.Contains(dl, "download=clip.mp4") {
		t.Fatalf("expected a download URL carrying the original filename, got %q", dl)
	}

	w = cl.do("GET", "/api/v1/media", "", 200, session...)
	var gallery struct{ Items []map[string]any }
	if err := json.NewDecoder(w.Body).Decode(&gallery); err != nil || len(gallery.Items) != 1 ||
		gallery.Items[0]["contentTitle"] != "หัวข้อ" || !strings.Contains(gallery.Items[0]["url"].(string), "view") {
		t.Fatalf("gallery must list the ready video with its title and view URL, got %+v", gallery)
	}
	if _, leaked := gallery.Items[0]["objectKey"]; leaked {
		t.Fatal("object key must not be exposed in the gallery")
	}
	cl.do("GET", "/api/v1/media", "", 401)

	storage.deleteErr = errors.New("r2 down")
	cl.do("DELETE", "/api/v1/media/"+testMediaID, "", 500, session...)
	if _, ok := store.items[testMediaID]; !ok {
		t.Fatal("row must survive a failed storage delete so it can be retried")
	}
	storage.deleteErr = nil
	cl.do("DELETE", "/api/v1/media/"+testMediaID, "", 204, session...)
	if len(storage.objects) != 0 {
		t.Fatal("deleting a video must remove the R2 object")
	}
	cl.do("DELETE", "/api/v1/media/"+testMediaID, "", 404, session...)
	cl.do("POST", "/api/v1/media/not-a-uuid/complete", `{}`, 404, session...)
}

func TestMediaRoutesNeedStorage(t *testing.T) {
	cl := newClient(t, Options{Auth: &fakeAuth{}, Media: &fakeMedia{items: map[string]media.Media{}}})
	cl.do("POST", "/api/v1/contents/"+testContentID+"/media", `{}`, 404, "Cookie", sessionCookie+"="+testToken)
}
