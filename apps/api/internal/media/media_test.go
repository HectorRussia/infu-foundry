package media

import (
	"strings"
	"testing"
)

func TestAttachmentDisposition(t *testing.T) {
	cases := map[string]string{
		"clip.mp4":    `attachment; filename="clip.mp4"; filename*=UTF-8''clip.mp4`,
		"แมว ep1.mp4": `attachment; filename="___ ep1.mp4"; filename*=UTF-8''%E0%B9%81%E0%B8%A1%E0%B8%A7%20ep1.mp4`,
		`a"b\c.mov`:   `attachment; filename="a_b_c.mov"; filename*=UTF-8''a%22b%5Cc.mov`,
	}
	for in, want := range cases {
		if got := AttachmentDisposition(in); got != want {
			t.Errorf("AttachmentDisposition(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCoverKeys(t *testing.T) {
	id := "22222222-2222-2222-2222-222222222222"
	key, err := NewCoverKey(id, "image/webp")
	if err != nil || !CoverKeyOf(key, id) || !strings.HasSuffix(key, ".webp") {
		t.Fatalf("NewCoverKey = %q, %v", key, err)
	}
	for _, bad := range []string{"media/x.mp4", "covers/" + id + "/../secret.jpg", strings.Replace(key, "2222", "3333", 1)} {
		if CoverKeyOf(bad, id) {
			t.Errorf("CoverKeyOf(%q) must be false", bad)
		}
	}
	if ValidCover("image/heic", 10) || ValidCover("image/png", MaxCoverSize+1) || !ValidCover("image/png", MaxCoverSize) {
		t.Error("ValidCover limits are wrong")
	}
}

func TestCoverFilename(t *testing.T) {
	key := "covers/22222222-2222-2222-2222-222222222222/00000000000000000000000000000000.webp"
	for title, want := range map[string]string{
		"EP.1 อัศวินตัวจิ๋ว": "EP.1 อัศวินตัวจิ๋ว.webp",
		"a/b: c?": "a-b- c-.webp",
		"   ":     "cover.webp",
	} {
		if got := CoverFilename(title, key); got != want {
			t.Errorf("CoverFilename(%q) = %q, want %q", title, got, want)
		}
	}
}
