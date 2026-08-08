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
	if fitted, _, err := images.Fit(bytes.NewReader(data), 1600); err == nil {
		data = fitted
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
