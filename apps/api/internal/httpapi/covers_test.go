package httpapi

import (
	"encoding/json"
	"strings"
	"testing"

	"infu-foundry/apps/api/internal/contents"
	"infu-foundry/apps/api/internal/media"
)

func TestCoverFlow(t *testing.T) {
	store := newFakeStore()
	store.items[testContentID] = contents.Content{ID: testContentID, Title: "เรื่อง", AuthorName: "Horn"}
	storage := &fakeStorage{objects: map[string]int64{}}
	cl := newClient(t, Options{Auth: &fakeAuth{}, Contents: store, Media: &fakeMedia{items: map[string]media.Media{}}, MediaStorage: storage})
	session := []string{"Cookie", sessionCookie + "=" + testToken}
	base := "/api/v1/contents/" + testContentID + "/cover"

	cl.do("POST", base, `{"contentType":"image/jpeg","size":1000}`, 401)
	cl.do("POST", base, `{"contentType":"image/gif","size":1000}`, 422, session...)
	cl.do("POST", base, `{"contentType":"image/jpeg","size":10485761}`, 422, session...)
	cl.do("POST", "/api/v1/contents/"+otherContent+"/cover", `{"contentType":"image/png","size":10}`, 404, session...)

	upload := func() string {
		w := cl.do("POST", base, `{"contentType":"image/jpeg","size":2048}`, 201, session...)
		var out struct{ Key, UploadURL string }
		if err := json.NewDecoder(w.Body).Decode(&out); err != nil || !media.CoverKeyOf(out.Key, testContentID) || out.UploadURL == "" {
			t.Fatalf("unexpected cover upload response %+v", out)
		}
		return out.Key
	}

	first := upload()
	cl.do("PUT", base, `{"key":"`+first+`"}`, 409, session...) // not uploaded yet
	storage.objects[first] = 2048
	cl.do("PUT", base, `{"key":"media/`+testMediaID+`.mp4"}`, 422, session...)                                    // a video is never a cover
	cl.do("PUT", base, `{"key":"covers/`+otherContent+`/00000000000000000000000000000000.jpg"}`, 422, session...) // another content's key
	w := cl.do("PUT", base, `{"key":"`+first+`"}`, 200, session...)
	if store.items[testContentID].CoverKey != first {
		t.Fatal("cover key must be saved")
	}
	var set map[string]any
	if err := json.NewDecoder(w.Body).Decode(&set); err != nil || !strings.Contains(set["coverDownloadUrl"].(string), "download=เรื่อง.jpg") || set["coverPositionY"] != 50.0 {
		t.Fatalf("set cover must return a named download URL and a centred crop, got %+v", set)
	}

	cl.do("PATCH", base, `{"x":20,"y":101}`, 422, session...)
	cl.do("PATCH", base, `{"x":20}`, 422, session...)
	cl.do("PATCH", "/api/v1/contents/"+otherContent+"/cover", `{"x":20,"y":30}`, 404, session...)
	cl.do("PATCH", base, `{"x":20,"y":30}`, 200, session...)
	if got := store.items[testContentID]; got.CoverPositionX != 20 || got.CoverPositionY != 30 {
		t.Fatalf("crop position must be saved, got %d,%d", got.CoverPositionX, got.CoverPositionY)
	}

	w = cl.do("GET", "/api/v1/contents", "", 200, session...)
	var list struct{ Items []map[string]any }
	if err := json.NewDecoder(w.Body).Decode(&list); err != nil || !strings.Contains(list.Items[0]["coverUrl"].(string), first) {
		t.Fatalf("list must carry a signed cover URL, got %+v", list)
	}
	if list.Items[0]["coverPositionX"] != 20.0 {
		t.Fatalf("list must carry the crop position, got %+v", list.Items[0])
	}
	if _, leaked := list.Items[0]["coverKey"]; leaked {
		t.Fatal("cover key must not be exposed")
	}

	second := upload()
	storage.objects[second] = 2048
	cl.do("PUT", base, `{"key":"`+second+`"}`, 200, session...)
	if store.items[testContentID].CoverPositionX != 50 {
		t.Fatal("a new cover image must start centred")
	}
	if _, kept := storage.objects[first]; kept {
		t.Fatal("replacing a cover must delete the old image")
	}

	cl.do("DELETE", base, "", 204, session...)
	if store.items[testContentID].CoverKey != "" || len(storage.objects) != 0 {
		t.Fatal("removing the cover must clear the key and delete the image")
	}
}

func TestCoverRoutesNeedStorage(t *testing.T) {
	store := newFakeStore()
	store.items[testContentID] = contents.Content{ID: testContentID}
	cl := newClient(t, Options{Auth: &fakeAuth{}, Contents: store})
	cl.do("POST", "/api/v1/contents/"+testContentID+"/cover", `{"contentType":"image/png","size":10}`, 404, "Cookie", sessionCookie+"="+testToken)
}
