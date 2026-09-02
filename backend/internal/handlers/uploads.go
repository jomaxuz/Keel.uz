package handlers

import (
	"bytes"
	"errors"
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"restaurant-backend/internal/images"
)

// Serving uploaded photographs, at the size the page actually shows them.
//
// It replaced `http.FileServer`, which did two things badly enough to matter:
// it served every image at full size, and it set no caching headers at all — so
// a guest scrolling back to the menu re-validated sixteen images, each costing a
// round trip to France.
//
// The width is asked for in the URL (`?w=600`) rather than baked into a second
// filename at upload time. That choice is the whole design:
//
//   - Every image already on every tenant's disk gets smaller **today**.
//     Upload-time variants would only have helped photographs uploaded after
//     the change, and a page asking for a `-600` file that predates it would
//     show a broken card.
//   - The site decides the size, which is where the knowledge lives: a menu card
//     is ~350 px, a dish page is full width, a logo is tiny. The backend does
//     not need to guess how any of them are laid out.
//
// Derivatives are cached on disk beside the originals, so the resize happens
// once per image per width, ever.

// The widths the site is allowed to ask for.
//
// ⚠️ **An allowlist, not a range.** `?w=` reaches this from the open internet,
// and an arbitrary width would let anybody spend the server's CPU and the
// tenant's disk one pixel at a time — a few thousand requests for w=101,
// w=102 … and the cache directory holds a thousand copies of every photograph.
// Three sizes cover every layout in the app.
var thumbWidths = map[int]bool{300: true, 600: true, 1200: true}

// Where derivatives live. A dotted prefix so it sorts out of the way, and
// refused as a request path below — the cache is not part of the public tree.
const thumbDir = ".thumb"

// ServeUploads serves an uploaded file, optionally resized.
func (h *Handler) ServeUploads(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/uploads/")
	if name == "" {
		http.NotFound(w, r)
		return
	}
	// Cleaned as an absolute path and then made relative again: this collapses
	// `..` before anything touches the filesystem. `os.Root` below refuses an
	// escape as well, so this is the belt to its braces — the file being asked
	// for is the one piece of this request that comes from a stranger.
	name = strings.TrimPrefix(path.Clean("/"+name), "/")
	if name == "" || strings.HasPrefix(name, thumbDir) {
		http.NotFound(w, r)
		return
	}

	root, err := os.OpenRoot(h.Cfg.UploadDir)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer root.Close()

	width := 0
	if raw := r.URL.Query().Get("w"); raw != "" {
		n, err := strconv.Atoi(raw)
		// An unknown width serves the original rather than failing: a stale page
		// or a hand-edited URL should show the photograph, not a 400.
		if err == nil && thumbWidths[n] {
			width = n
		}
	}

	if width == 0 {
		h.serveOriginal(w, r, root, name)
		return
	}
	h.serveThumb(w, r, root, name, width)
}

func (h *Handler) serveOriginal(w http.ResponseWriter, r *http.Request, root *os.Root, name string) {
	f, err := root.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	setImageCache(w, name)
	http.ServeContent(w, r, name, info.ModTime(), f)
}

// The extensions a derivative can have been written under, newest first.
//
// ⚠️ **The cached file carries its own extension, and that is what fixes a
// silent mismatch.** The derivative is served with `http.ServeContent`, which
// reads the content type off the name it is given — so a WebP cached under the
// original's `.png` was served as a PNG. Naming the cache file for what is
// actually in it also means a derivative written before the encoder changed is
// simply a different path: the old ones are ignored rather than served as
// something they are not.
var thumbExts = []string{".webp", ".jpg", ".png"}

// serveThumb serves the cached derivative, generating it the first time.
func (h *Handler) serveThumb(w http.ResponseWriter, r *http.Request, root *os.Root, name string, width int) {
	base := filepath.Join(thumbDir, strconv.Itoa(width), name)
	for _, ext := range thumbExts {
		f, err := root.Open(base + ext)
		if err != nil {
			continue
		}
		defer f.Close()
		if info, err := f.Stat(); err == nil && !info.IsDir() {
			setImageCache(w, name)
			// ⚠️ Named for the cache file, not for the original: this is the
			// argument ServeContent reads the content type from.
			http.ServeContent(w, r, base+ext, info.ModTime(), f)
			return
		}
	}

	src, err := root.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer src.Close()

	data, contentType, err := images.Fit(src, width)
	if err != nil {
		// A format this package does not resize (WebP, animated GIF) and a file
		// that is not an image at all end up here, and both want the same
		// answer: hand over what was asked for. Falling back rather than failing
		// is what lets the site put `?w=` on every image without knowing what
		// any of them are.
		// `image.ErrFormat` belongs with ErrUnsupported here: both mean "this is
		// not something to resize", and logging every text file somebody asks
		// for with `?w=` would bury the failures that matter.
		if !errors.Is(err, images.ErrUnsupported) && !errors.Is(err, image.ErrFormat) {
			log.Printf("uploads: resize %s: %v", name, err)
		}
		if _, err := src.Seek(0, io.SeekStart); err == nil {
			if info, err := src.Stat(); err == nil {
				setImageCache(w, name)
				http.ServeContent(w, r, name, info.ModTime(), src)
				return
			}
		}
		http.NotFound(w, r)
		return
	}

	// Written before serving, and atomically: two guests can arrive on the same
	// cold image at once, and a half-written file left behind by the loser would
	// be served as a broken photograph for as long as the cache lives.
	cacheRel := base + images.ExtFor(contentType)
	if err := writeCached(h.Cfg.UploadDir, cacheRel, data); err != nil {
		log.Printf("uploads: cache %s: %v", cacheRel, err)
	}

	w.Header().Set("Content-Type", contentType)
	setImageCache(w, name)
	http.ServeContent(w, r, name, time.Now(), bytes.NewReader(data))
}

func writeCached(root, rel string, data []byte) error {
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(full), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), full)
}

// setImageCache tells the browser it never has to ask again.
//
// Uploaded files are given random names, so a replaced photograph is a new URL —
// which is what makes `immutable` honest here rather than a gamble.
//
// The seeded demo images are the exception: they have fixed names (`seed/…`) and
// a future release could ship different bytes under the same one. They get a
// month and no `immutable`, so a change heals by itself instead of sitting in
// caches for a year.
func setImageCache(w http.ResponseWriter, name string) {
	if strings.HasPrefix(name, "seed/") {
		w.Header().Set("Cache-Control", "public, max-age=2592000")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
}
