package media

import "testing"

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
