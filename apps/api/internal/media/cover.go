package media

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
)

// MaxCoverSize keeps cover images light enough for phones and the free R2 tier.
const MaxCoverSize = 10 << 20 // 10 MB

var coverExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

var coverKeyPattern = regexp.MustCompile(`^covers/([0-9a-f-]{36})/[0-9a-f]{32}\.(jpg|png|webp)$`)

func ValidCover(contentType string, size int64) bool {
	_, ok := coverExtensions[contentType]
	return ok && size > 0 && size <= MaxCoverSize
}

// NewCoverKey returns covers/<content id>/<random>.<ext>. A fresh name per upload
// means browsers never show a cached old cover after it is replaced.
func NewCoverKey(contentID, contentType string) (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "covers/" + strings.ToLower(contentID) + "/" + hex.EncodeToString(raw) + coverExtensions[contentType], nil
}

// CoverKeyOf reports whether key is a cover key issued for contentID, so a client
// can never attach another content's object (or a video) as a cover.
func CoverKeyOf(key, contentID string) bool {
	m := coverKeyPattern.FindStringSubmatch(key)
	return m != nil && m[1] == strings.ToLower(contentID)
}

// CoverFilename names a downloaded cover after its content title, keeping the
// stored extension; characters that phones and Windows reject become "-".
func CoverFilename(title, key string) string {
	name := strings.Map(func(r rune) rune {
		if r < 0x20 || strings.ContainsRune(`/\:*?"<>|`, r) {
			return '-'
		}
		return r
	}, strings.TrimSpace(title))
	if runes := []rune(name); len(runes) > 80 {
		name = string(runes[:80])
	}
	if name == "" {
		name = "cover"
	}
	ext := key[strings.LastIndex(key, "."):]
	return name + ext
}
