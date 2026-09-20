package handlers

// ---- The restaurant's own advert, as they made it ----
//
// ⚠️ **Not every restaurant wants a plan.** A place with somebody doing its
// social media already shoots the video, cuts the poster and knows what it
// wants to say; what it does not want is Ads Manager — the objectives, the
// audiences, the bid strategies, the six screens between a finished video and
// an advert that runs. That is the part this section exists to remove, and
// tying every campaign to a dish out of our plan put a different obstacle in
// exactly the same place.
//
// ⚠️ **Stored exactly as it arrived.** The panel's ordinary image upload fits
// and re-encodes everything it can decode, often to WebP — which is right for a
// menu photograph on a phone and wrong here: Meta's advert images are JPG and
// PNG, and a WebP under a confident name is refused at the creative with a
// message about the image, long after the owner chose the file. So this is its
// own endpoint with its own list, and it changes no bytes.

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"restaurant-backend/internal/httpx"
)

// ⚠️ **Videos are large and a kitchen's connection is not.** The ceiling is on
// the body before anything is read: refusing afterwards means the restaurant's
// connection carried the file anyway, which is the whole complaint.
const adsMediaMaxBytes = 200 << 20

// What Meta accepts for an advert, and nothing else.
//
// ⚠️ **No WebP and no GIF**, however well they would serve a web page: Meta
// takes JPG and PNG for images, and MP4 or MOV for video. Accepting a format
// it will refuse only moves the refusal to the step that spends money.
var adsMediaExt = map[string]string{
	".jpg": "image", ".jpeg": "image", ".png": "image",
	".mp4": "video", ".mov": "video",
}

// AdminAdsMedia takes a picture or a video the restaurant made itself.
func (h *Handler) AdminAdsMedia(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, adsMediaMaxBytes)

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
	defer func() { _ = part.r.Close() }()

	ext := strings.ToLower(filepath.Ext(part.name))
	kind, ok := adsMediaExt[ext]
	if !ok {
		httpx.Error(w, http.StatusBadRequest,
			"faqat JPG, PNG rasm yoki MP4, MOV video")
		return
	}
	// The first bytes decide whether this is written at all, so they are read
	// before the file is created — a rejected upload must leave nothing behind.
	head := make([]byte, 32)
	n, _ := io.ReadFull(part.r, head)
	head = head[:n]
	if kind == "video" && !tvVideoSniff(head, ".mp4") && !tvVideoSniff(head, ext) {
		httpx.Error(w, http.StatusBadRequest, "bu fayl video emas")
		return
	}

	// ⚠️ Its own folder, so a sweeper that one day tidies unused uploads can
	// tell an advert's material from a dish photograph without guessing.
	dir := filepath.Join(h.Cfg.UploadDir, "ads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	name := randomName() + ext
	full := filepath.Join(dir, name)
	f, err := os.Create(full)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	_, err = io.Copy(f, io.MultiReader(bytes.NewReader(head), part.r))
	cerr := f.Close()
	if err != nil || cerr != nil {
		// ⚠️ Removed, not left: a connection dropped halfway leaves a file no
		// campaign references and nothing will ever clean up.
		_ = os.Remove(full)
		httpx.Error(w, http.StatusRequestEntityTooLarge,
			"fayl juda katta — 200 MB gacha")
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"url":  "/uploads/ads/" + name,
		"kind": kind,
	})
}
