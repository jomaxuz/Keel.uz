package handlers

import (
	"context"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"restaurant-backend/internal/escpos"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
)

// The restaurant's logo, as a printer sees it.
//
// ⚠️ **Rastered once and kept**, not built per receipt. Decoding a PNG,
// scaling it and dithering it is milliseconds — and a busy Friday is a thousand
// receipts, on a server shared by every restaurant on the box. The logo changes
// when somebody uploads a new one, which is what the cache key is for.
var (
	logoMu    sync.Mutex
	logoCache = map[string]logoEntry{}
)

type logoEntry struct {
	raster []byte
	at     time.Time
}

// logoRaster returns the ESC/POS bytes for this branch's logo at this width, or
// nil when there is nothing to print.
//
// ⚠️ **nil is a normal answer, not a failure.** A restaurant that never
// uploaded a logo, an upload that has since been deleted, a file the server
// cannot read — all of them print a receipt without a logo rather than no
// receipt at all. A missing picture must never cost a guest their bill.
func (h *Handler) logoRaster(ctx context.Context, widthMM int) []byte {
	url := h.logoURL(ctx)
	if url == "" {
		return nil
	}
	dots := escpos.DotsFor(widthMM)
	key := url + "|" + itoa(dots)

	logoMu.Lock()
	if e, ok := logoCache[key]; ok && time.Since(e.at) < 30*time.Minute {
		logoMu.Unlock()
		return e.raster
	}
	logoMu.Unlock()

	raster := h.buildLogo(url, dots)

	logoMu.Lock()
	logoCache[key] = logoEntry{raster: raster, at: time.Now()}
	logoMu.Unlock()
	return raster
}

// logoURL is the picture the site already shows.
//
// ⚠️ **The same file, not a second upload.** A "receipt logo" field would be one
// more thing to fill in and one more thing to forget, and the first restaurant
// to change its logo would have two of them, disagreeing.
func (h *Handler) logoURL(ctx context.Context) string {
	var rest models.Restaurant
	if err := h.Store.Restaurant.FindOne(ctx, bson.M{}).Decode(&rest); err != nil {
		return ""
	}
	return rest.LogoURL
}

// buildLogo reads the uploaded file and rasters it.
func (h *Handler) buildLogo(url string, dots int) []byte {
	path := h.uploadPath(url)
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		// ⚠️ WebP and SVG land here, and both are ordinary things to have
		// uploaded: the site shows them happily. There is no printer that can,
		// so the receipt goes out without a logo rather than with a stripe of
		// noise where one should be.
		return nil
	}
	return escpos.Logo(img, dots)
}

// uploadPath turns a stored URL back into the file on disk.
//
// ⚠️ **Only files under the upload directory.** The URL comes from a document
// an owner can edit, and `filepath.Clean` plus the prefix check is what stops
// "../../etc/passwd" being rastered onto a receipt — a small leak with a very
// odd delivery mechanism, but a leak.
func (h *Handler) uploadPath(url string) string {
	i := strings.Index(url, "/uploads/")
	if i < 0 {
		return ""
	}
	name := strings.TrimPrefix(url[i:], "/uploads/")
	if name == "" {
		return ""
	}
	root, err := filepath.Abs(h.Cfg.UploadDir)
	if err != nil {
		return ""
	}
	full := filepath.Join(root, filepath.Clean("/"+name))
	if !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return ""
	}
	return full
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
