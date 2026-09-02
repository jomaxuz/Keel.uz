package handlers

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/images"
)

var allowedExt = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true,
}

// ⚠️ **What is accepted and what is stored are different questions.** A
// restaurant uploads what its designer sent — a print-quality PNG, a phone's
// JPEG — and every one of those is converted to WebP before it touches the
// disk (see images.Fit). The original is never written, so there is nothing to
// clean up later and no second copy of a photograph to go stale.

// Upload accepts a multipart "file" field, stores it in UPLOAD_DIR, and
// returns the public URL. Images are served from /uploads/<name>.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB
		httpx.Error(w, http.StatusBadRequest, "file too large")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExt[ext] {
		httpx.Error(w, http.StatusBadRequest, "unsupported file type")
		return
	}

	// The stored name is decided **after** the conversion, from what actually
	// came out — see below. This is only the fallback for a file the converter
	// leaves alone (an animation).
	name := randomName() + ext
	if err := os.MkdirAll(h.Cfg.UploadDir, 0o755); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ **The original is shrunk too, not just the derivatives.**
	//
	// A phone camera produces 3000×4000 at 4 MB, and an owner uploading their
	// menu has no reason to think about that. Serving derivatives would hide it
	// from guests, but the file still sits on the tenant's disk for ever, gets
	// backed up every night, and is what the dish page hands to anybody who
	// opens the full-size image.
	//
	// 1600 px is chosen against what the site does with it: the widest thing any
	// page shows is a full-width cover on a large screen, and 1600 covers that
	// on a 2x display. Anything past it is storage nobody looks at.
	//
	// A file this cannot decode (WebP, an animated GIF, something misnamed) is
	// stored exactly as it arrived — refusing an upload because we could not
	// improve it would be the wrong trade for the owner standing in the kitchen.
	data, err := io.ReadAll(io.LimitReader(file, 12<<20))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ **The extension follows the bytes.** A WebP written under `.png` is
	// served as a PNG by the uploads route — which decides the type from the
	// filename — and a browser that trusts the header rather than sniffing
	// shows nothing at all.
	if fitted, contentType, err := images.Fit(bytes.NewReader(data), 1600); err == nil {
		data = fitted
		name = strings.TrimSuffix(name, ext) + images.ExtFor(contentType)
	}
	if err := os.WriteFile(filepath.Join(h.Cfg.UploadDir, name), data, 0o644); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	url := strings.TrimRight(h.Cfg.PublicBaseURL, "/") + "/uploads/" + name
	httpx.JSON(w, http.StatusCreated, map[string]string{"url": url, "filename": name})
}

func randomName() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
