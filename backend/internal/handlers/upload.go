package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"restaurant-backend/internal/httpx"
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
	dst, err := os.Create(filepath.Join(h.Cfg.UploadDir, name))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
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
