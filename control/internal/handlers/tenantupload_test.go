package handlers

import (
	"net/http"
	"strings"
	"testing"
)

// ⚠️ **SVG is an image by `image/*` and an executable document by every other
// test**, and it would be served from the customer's own origin — the one place
// a script must not run. This is why the accepted types are a list rather than
// a prefix check.
func TestSvgIsNotAnImageForThisPurpose(t *testing.T) {
	if _, ok := uploadTypes["image/svg+xml"]; ok {
		t.Error("SVG is accepted; it executes on the customer's origin")
	}
	for _, want := range []string{"image/jpeg", "image/png", "image/webp"} {
		if _, ok := uploadTypes[want]; !ok {
			t.Errorf("%s is an ordinary photograph and is refused", want)
		}
	}
}

// ⚠️ **What decides is the bytes, not the browser's Content-Type**, which is a
// hint typed by whatever made the request. An HTML document named .jpg would
// otherwise be stored and then served from the customer's origin.
func TestTheTypeIsReadFromTheBytes(t *testing.T) {
	html := []byte("<!DOCTYPE html><script>alert(1)</script>")
	kind := http.DetectContentType(html)
	if i := strings.IndexByte(kind, ';'); i >= 0 {
		kind = kind[:i]
	}
	if _, ok := uploadTypes[kind]; ok {
		t.Errorf("an HTML document sniffed as %q and was accepted", kind)
	}

	png := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("\x00", 40))
	kind = http.DetectContentType(png)
	if i := strings.IndexByte(kind, ';'); i >= 0 {
		kind = kind[:i]
	}
	if _, ok := uploadTypes[kind]; !ok {
		t.Errorf("a PNG sniffed as %q and was refused", kind)
	}
}

// Two customers' briefs both contain "hero.jpg". A name derived from what was
// uploaded would overwrite a photograph already live on somebody's site.
func TestUploadNamesDoNotCollide(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		n := randomHex(10)
		if seen[n] {
			t.Fatal("a generated name repeated within 500 draws")
		}
		seen[n] = true
	}
}
