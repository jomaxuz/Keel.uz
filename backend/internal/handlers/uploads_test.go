package handlers

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"restaurant-backend/internal/config"

	"github.com/go-chi/chi/v5"
)

// The width parameter reaches this handler from the open internet, so the tests
// here are mostly about what it must refuse.
func TestServeUploadsGuardsTheWidthAndThePath(t *testing.T) {
	dir := t.TempDir()
	// Encoded here rather than hardcoded as bytes: a hand-written PNG header is
	// how this test first failed, and the encoder is the same one the resizer
	// decodes with.
	img := image.NewNRGBA(image.Rect(0, 0, 40, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 40; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x * 6), G: uint8(y * 8), B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.png"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "..", "outside"), 0o755); err == nil {
		_ = os.WriteFile(filepath.Join(dir, "..", "outside", "boo.txt"), []byte("no"), 0o644)
	}

	h := &Handler{Cfg: &config.Config{UploadDir: dir}}
	get := func(target string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rec := httptest.NewRecorder()
		r := chi.NewRouter()
		r.Get("/uploads/*", h.ServeUploads)
		r.ServeHTTP(rec, req)
		return rec
	}

	if rec := get("/uploads/a.png"); rec.Code != 200 {
		t.Fatalf("asl rasm berilmadi: %d", rec.Code)
	}

	// ⚠️ The reason the allowlist exists: an arbitrary width would let a
	// stranger fill the disk with one derivative per pixel value. An unknown
	// width falls back to the original rather than erroring, so a stale page
	// still shows the photograph.
	rec := get("/uploads/a.png?w=137")
	if rec.Code != 200 {
		t.Fatalf("noma'lum kenglik xato berdi: %d", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(dir, thumbDir, "137")); err == nil {
		t.Error("ruxsat etilmagan kenglik uchun kesh yaratildi")
	}

	// An allowed width is generated and cached exactly once.
	if rec := get("/uploads/a.png?w=300"); rec.Code != 200 {
		t.Fatalf("300px berilmadi: %d", rec.Code)
	}
	cached := filepath.Join(dir, thumbDir, "300", "a.png")
	if _, err := os.Stat(cached); err != nil {
		t.Fatalf("kesh yozilmadi: %v", err)
	}

	// The cache is not part of the public tree: serving it back would let a
	// crafted path walk into it.
	if rec := get("/uploads/" + thumbDir + "/300/a.png"); rec.Code != 404 {
		t.Errorf("kesh papkasi tashqariga berildi: %d", rec.Code)
	}

	// Traversal, in the two shapes that reach a handler.
	for _, p := range []string{
		"/uploads/../outside/boo.txt",
		"/uploads/%2e%2e/outside/boo.txt",
	} {
		if rec := get(p); rec.Code == 200 {
			t.Errorf("papkadan chiqish mumkin bo'ldi: %s", p)
		}
	}

	// A non-image is served as-is rather than 500ing: `?w=` is put on every
	// image URL by the site, which cannot know what each file really is.
	if rec := get("/uploads/secret.txt?w=600"); rec.Code != 200 {
		t.Errorf("rasm bo'lmagan fayl xato berdi: %d", rec.Code)
	}
}

// Repeat visits were re-validating sixteen images, each a round trip to France.
func TestServeUploadsCachesForeverExceptTheSeed(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "seed"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "x.jpg"), []byte("not really a jpeg"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "seed", "cover.jpg"), []byte("nor this"), 0o644)

	h := &Handler{Cfg: &config.Config{UploadDir: dir}}
	head := func(target string) string {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rec := httptest.NewRecorder()
		r := chi.NewRouter()
		r.Get("/uploads/*", h.ServeUploads)
		r.ServeHTTP(rec, req)
		return rec.Header().Get("Cache-Control")
	}

	if got := head("/uploads/x.jpg"); got != "public, max-age=31536000, immutable" {
		t.Errorf("tasodifiy nomli fayl: %q", got)
	}
	// The seed images have fixed names, so a future release could ship different
	// bytes under the same URL. A month, and no `immutable`.
	if got := head("/uploads/seed/cover.jpg"); got != "public, max-age=2592000" {
		t.Errorf("seed fayli: %q", got)
	}
}
