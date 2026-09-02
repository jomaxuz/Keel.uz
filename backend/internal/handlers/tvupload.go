package handlers

// ---- Uploading a video for the wall ----
//
// Its own endpoint rather than a branch inside Upload, and the reason is the
// first line of that file: every image is converted to WebP before it touches
// the disk. A video handed to an image encoder is not a smaller video, it is a
// corrupt file — and it would be written under a name that says otherwise.
//
// ⚠️ **Streamed to disk, never read into memory.** A hundred megabytes through
// `io.ReadAll` is a hundred megabytes of a 1 GB VPS, and two managers uploading
// at once is the tenant's whole server. `MultipartReader` copies it through.

import (
	"bytes"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"restaurant-backend/internal/httpx"
)

// The ceiling on one video.
//
// ⚠️ **A number chosen against a dining room, not against generosity.** What
// plays on a wall is a loop of fifteen to sixty seconds; 120 MB covers a couple
// of minutes at 1080p with room to spare. The cost of being generous here is
// not ours: every television in the branch downloads every file, over the
// restaurant's own wifi, and the tenant's disk is the one that fills.
const tvVideoMaxBytes = 120 << 20

// What a television can play, and what we are willing to serve.
//
// ⚠️ Two formats, not "whatever ffmpeg reads". Nothing here transcodes — the
// file is played exactly as uploaded — so the list is what Android's own player
// is guaranteed to open. A .mov from an iPhone that plays on the manager's
// laptop and shows a black rectangle in the dining room is the failure this
// avoids, and it is one an owner cannot diagnose.
var tvVideoExt = map[string]string{
	".mp4":  "video/mp4",
	".webm": "video/webm",
}

// tvVideoSniff reports whether these opening bytes really are the format the
// name claims.
//
// ⚠️ **The extension is a claim, and this endpoint writes into a directory the
// whole internet reads.** An HTML file uploaded as `promo.mp4` would be stored,
// served under our own domain, and served as `video/mp4` by nothing — the
// uploads route names the type from the extension, and a browser that sniffs
// would run it. Checking the bytes costs one read.
func tvVideoSniff(head []byte, ext string) bool {
	switch ext {
	case ".mp4":
		// An ISO base-media file: a size, then the literal `ftyp` box.
		return len(head) >= 12 && bytes.Equal(head[4:8], []byte("ftyp"))
	case ".webm":
		// Matroska's EBML header.
		return len(head) >= 4 && bytes.Equal(head[:4], []byte{0x1A, 0x45, 0xDF, 0xA3})
	}
	return false
}

// ⚠️ **The two video types are registered by hand.** `http.ServeContent` names
// the type from the extension, and Go's built-in table — the only one there is
// inside a scratch or alpine container, where `/etc/mime.types` does not
// exist — has no entry for either. It would fall through to sniffing, which
// happens to get both right; "happens to" is not what a dining room's screen
// should depend on across a Go release.
func init() {
	for ext, typ := range tvVideoExt {
		_ = mime.AddExtensionType(ext, typ)
	}
}

// AdminTVUpload stores one video and returns the URL a slide points at.
func (h *Handler) AdminTVUpload(w http.ResponseWriter, r *http.Request) {
	// ⚠️ The limit is on the body, before anything is read: refusing after the
	// upload finished means the restaurant's connection carried it anyway, and
	// on a phone tethered in a kitchen that is the whole complaint.
	r.Body = http.MaxBytesReader(w, r.Body, tvVideoMaxBytes)

	mr, err := r.MultipartReader()
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "fayl yuborilmadi")
		return
	}
	var part *multipartFile
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "fayl yuborilmadi")
			return
		}
		if p.FormName() == "file" {
			part = &multipartFile{r: p, name: p.FileName()}
			break
		}
		_ = p.Close()
	}
	if part == nil {
		httpx.Error(w, http.StatusBadRequest, "fayl yuborilmadi")
		return
	}
	defer part.r.Close()

	ext := strings.ToLower(filepath.Ext(part.name))
	if _, ok := tvVideoExt[ext]; !ok {
		httpx.Error(w, http.StatusBadRequest, "faqat MP4 yoki WebM video")
		return
	}

	// The first bytes decide whether this is written at all, so they are read
	// before the file is created — a rejected upload must leave nothing behind.
	head := make([]byte, 32)
	n, _ := io.ReadFull(part.r, head)
	head = head[:n]
	if !tvVideoSniff(head, ext) {
		httpx.Error(w, http.StatusBadRequest, "bu fayl video emas")
		return
	}

	if err := os.MkdirAll(h.Cfg.UploadDir, 0o755); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	name := randomName() + ext
	full := filepath.Join(h.Cfg.UploadDir, name)
	f, err := os.Create(full)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	written, err := io.Copy(f, io.MultiReader(bytes.NewReader(head), part.r))
	cerr := f.Close()
	if err != nil || cerr != nil {
		// ⚠️ Removed, not left. A connection dropped halfway leaves a file that
		// no slide references and nothing will ever clean up — and the next one
		// is a hundred megabytes too.
		_ = os.Remove(full)
		// MaxBytesReader stops the copy: the honest message names the limit
		// rather than blaming the network the owner is standing next to.
		httpx.Error(w, http.StatusRequestEntityTooLarge,
			"video juda katta — 120 MB gacha")
		return
	}

	url := strings.TrimRight(h.Cfg.PublicBaseURL, "/") + "/uploads/" + name
	httpx.JSON(w, http.StatusCreated, map[string]any{
		"url":      url,
		"filename": name,
		"bytes":    written,
	})
}

// multipartFile is the one part we care about, kept so the loop above can break
// out with it.
type multipartFile struct {
	r    io.ReadCloser
	name string
}
