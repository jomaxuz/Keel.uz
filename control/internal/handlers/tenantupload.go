package handlers

// Putting a photograph into a customer's site from the console.
//
// ⚠️ **Why this exists at all.** The constructor could point a design at an
// image, but only by typing a path that already existed — so a layout built
// around a photograph could only be drawn for a customer who had already
// uploaded one through their own panel. For a shop that is backwards: the site
// is being drawn *before* the owner has ever logged in, and the pictures are in
// the brief the operator is working from.
//
// ⚠️ **It writes through Docker, not through a mount.** The uploads root is
// mounted read-only into this process on purpose — see provision.WriteUpload.
//
// ⚠️ **The name is generated here and never taken from the browser.** A
// client-supplied name reaches a shell command and a filesystem path; the only
// safe version of that question is not to ask it.

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"

	"keel-control/internal/httpx"
)

// maxUpload is what a design photograph may weigh.
//
// ⚠️ **A ceiling, because this lands on the disk every customer shares.** Ten
// megabytes is a generous photograph and a careless export; beyond that it is a
// file somebody meant to resize, and the site would serve it to every visitor
// on a phone.
const maxUpload = 10 << 20

// The types a browser can actually draw. ⚠️ An allowlist rather than a check
// for "image/*": SVG is an image by that test and an executable document by
// every other one, and it would be served from the customer's own origin.
var uploadTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
	"image/avif": "avif",
	"image/gif":  "gif",
}

// TenantUpload stores one photograph in a tenant's uploads and answers with the
// path a design can point at.
func (h *Handler) TenantUpload(w http.ResponseWriter, r *http.Request) {
	t := h.tenantFromURL(r)
	if t == nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	if h.Docker == nil {
		httpx.Error(w, http.StatusServiceUnavailable, "docker mavjud emas")
		return
	}

	// ⚠️ Capped before the body is read, not after: an unbounded read is a way
	// to fill the disk of the machine every restaurant runs on, and it costs
	// nothing to refuse early.
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload+(1<<20))
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "fayl yuborilmadi")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxUpload+1))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(data) > maxUpload {
		httpx.Error(w, http.StatusRequestEntityTooLarge, "rasm 10 MB dan katta")
		return
	}

	// ⚠️ **Sniffed, not trusted.** The browser's own Content-Type is a hint
	// typed by whatever produced the request; what decides here is the first
	// bytes of the file. A .jpg that is really an HTML document would be served
	// from the customer's origin, which is the one place it must not run.
	kind := http.DetectContentType(data)
	if i := strings.IndexByte(kind, ';'); i >= 0 {
		kind = kind[:i]
	}
	ext, ok := uploadTypes[kind]
	if !ok {
		httpx.Error(w, http.StatusBadRequest,
			fmt.Sprintf("bu turdagi fayl qo'llab-quvvatlanmaydi: %s", kind))
		return
	}

	name := "d" + randomHex(10) + "." + ext
	if err := h.Docker.WriteUpload(r.Context(), t.Slug, name, data); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// What the operator did, in the log every other console action lands in: a
	// photograph appearing on a customer's site is a change to their site, and
	// "who put that there" is a question somebody eventually asks.
	if actor, err := h.actor(r); err == nil {
		h.logConsole(r.Context(), actor, "tenant.upload", t.Slug,
			"dizaynga rasm: "+name+" ← "+header.Filename)
	}

	// ⚠️ The path as the **site** serves it, not as the disk holds it. The
	// design document stores exactly this string and `sanitizeImagePath`
	// requires the `/uploads/` prefix — handing back a host path would produce
	// a design that saves and renders nothing.
	httpx.JSON(w, http.StatusOK, map[string]any{
		"path": "/uploads/design/" + name,
		"size": len(data),
	})
}

// randomHex is the file's name. ⚠️ Random rather than derived from what was
// uploaded: two customers' briefs both contain "hero.jpg", and a name that
// collides overwrites a photograph that is already live on a site.
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// Never in practice; a predictable name is still better than a panic in
		// the middle of an upload.
		return "0000000000000000000000"[:n*2]
	}
	return hex.EncodeToString(b)
}
