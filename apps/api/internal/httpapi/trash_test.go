package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"infu-foundry/apps/api/internal/contents"
	"infu-foundry/apps/api/internal/media"
)

// fakeTrash keeps deleted contents and the R2 keys that belong to each one.
type fakeTrash struct {
	items map[string]contents.Trashed
	keys  map[string][]string
	live  map[string]bool
}

func (f *fakeTrash) ListTrash(context.Context) ([]contents.Trashed, error) {
	out := []contents.Trashed{}
	for _, item := range f.items {
		out = append(out, item)
	}
	return out, nil
}

func (f *fakeTrash) Restore(_ context.Context, id string) error {
	if _, ok := f.items[id]; !ok {
		return contents.ErrNotFound
	}
	delete(f.items, id)
	f.live[id] = true
	return nil
}

func (f *fakeTrash) TrashObjectKeys(_ context.Context, id string) ([]string, error) {
	if _, ok := f.items[id]; !ok {
		return nil, contents.ErrNotFound
	}
	return f.keys[id], nil
}

func (f *fakeTrash) Purge(_ context.Context, id string) error {
	if _, ok := f.items[id]; !ok {
		return contents.ErrNotFound
	}
	delete(f.items, id)
	delete(f.keys, id)
	return nil
}

func TestTrashFlow(t *testing.T) {
	const second = "33333333-3333-3333-3333-333333333333"
	cover := "covers/" + testContentID + "/00000000000000000000000000000000.jpg"
	video := "media/" + testMediaID + ".mp4"
	trash := &fakeTrash{
		items: map[string]contents.Trashed{
			testContentID: {ID: testContentID, Title: "ลบแล้ว", CoverKey: cover, DeletedAt: time.Now()},
			second:        {ID: second, Title: "อีกอัน"},
		},
		keys: map[string][]string{testContentID: {cover, video}, second: {"media/other.mp4"}},
		live: map[string]bool{},
	}
	storage := &fakeStorage{objects: map[string]int64{cover: 10, video: 10, "media/other.mp4": 10}}
	cl := newClient(t, Options{Auth: &fakeAuth{}, Contents: newFakeStore(), Trash: trash, MediaStorage: storage, Media: &fakeMedia{items: map[string]media.Media{}}})
	session := []string{"Cookie", sessionCookie + "=" + testToken}

	cl.do("GET", "/api/v1/trash", "", 401)
	w := cl.do("GET", "/api/v1/trash", "", 200, session...)
	var list struct{ Items []map[string]any }
	if err := json.NewDecoder(w.Body).Decode(&list); err != nil || len(list.Items) != 2 {
		t.Fatalf("trash must list deleted contents, got %+v", list)
	}
	for _, item := range list.Items {
		if _, leaked := item["coverKey"]; leaked {
			t.Fatal("cover key must not be exposed")
		}
	}

	cl.do("POST", "/api/v1/trash/"+second+"/restore", "", 415, session...)
	cl.do("POST", "/api/v1/trash/"+second+"/restore", "{}", 204, session...)
	if !trash.live[second] {
		t.Fatal("restore must bring the content back")
	}
	cl.do("POST", "/api/v1/trash/"+second+"/restore", "{}", 404, session...)
	cl.do("DELETE", "/api/v1/trash/"+second, "", 404, session...) // live content is never purged
	cl.do("DELETE", "/api/v1/trash/not-a-uuid", "", 404, session...)

	storage.deleteErr = errors.New("r2 down")
	cl.do("DELETE", "/api/v1/trash/"+testContentID, "", 500, session...)
	if _, kept := trash.items[testContentID]; !kept {
		t.Fatal("a failed R2 delete must keep the content in the trash")
	}
	storage.deleteErr = nil
	cl.do("DELETE", "/api/v1/trash/"+testContentID, "", 204, session...)
	if _, kept := trash.items[testContentID]; kept || len(storage.objects) != 1 {
		t.Fatalf("purge must delete the row and its cover and videos, objects left %v", storage.objects)
	}
}

func TestEmptyTrash(t *testing.T) {
	trash := &fakeTrash{
		items: map[string]contents.Trashed{testContentID: {ID: testContentID}, testMediaID: {ID: testMediaID}},
		keys:  map[string][]string{testContentID: {"media/a.mp4"}, testMediaID: {"media/b.mp4"}},
		live:  map[string]bool{},
	}
	storage := &fakeStorage{objects: map[string]int64{"media/a.mp4": 1, "media/b.mp4": 1}}
	cl := newClient(t, Options{Auth: &fakeAuth{}, Contents: newFakeStore(), Trash: trash, MediaStorage: storage})
	cl.do("DELETE", "/api/v1/trash", "", 204, "Cookie", sessionCookie+"="+testToken)
	if len(trash.items) != 0 || len(storage.objects) != 0 {
		t.Fatal("emptying the trash must purge every content and object")
	}
}

func TestTrashNeedsStorage(t *testing.T) {
	cl := newClient(t, Options{Auth: &fakeAuth{}, Contents: newFakeStore(), Trash: &fakeTrash{}})
	cl.do("GET", "/api/v1/trash", "", 404, "Cookie", sessionCookie+"="+testToken)
}
