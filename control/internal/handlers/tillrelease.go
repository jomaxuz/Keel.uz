package handlers

// ---- What version of the till a monoblock should be running ----
//
// ⚠️ **The binary is ours, so the answer comes from here.** Each restaurant runs
// its own tenant server; asking one of those what the current till build is
// would mean shipping the installer to every VPS in the country and keeping a
// hundred copies in step. A monoblock asks Keel directly, which is the one
// place that knows what was released.
//
// ⚠️ **Public, and it says nothing a download page would not.** A version
// number and a URL: whoever can reach this could already download the
// installer, which is a thing we hand to restaurants. Putting it behind a token
// would mean a till that cannot update itself the day its branch token is
// rotated — the exact moment somebody needs a fix.
//
// ⚠️ **The manifest is a file on disk, not a constant in this binary.** A
// release is a build of the *till*, and the control plane is deployed on its
// own schedule; tying them together would mean redeploying the console to
// publish a till fix. `TILL_RELEASE_DIR` holds `latest.json` beside the
// installer it names.

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// tillRelease is what a monoblock is told. Deliberately small: anything else it
// might want, it can ask for after it has decided to update.
type tillRelease struct {
	// "1.4.0" — compared against the version compiled into the till.
	Version string `json:"version"`
	// Where the installer is. Absolute, because the till may be on a different
	// host entirely and a relative path would resolve against its own server.
	URL string `json:"url"`
	// ⚠️ **Required, and the till refuses an update without it.** This file is
	// fetched over the open internet by a machine that will then run it as
	// administrator; a download nobody checked is a download somebody else can
	// substitute. Lowercase hex of the SHA-256.
	SHA256 string `json:"sha256"`
	// Shown to nobody automatically — kept so a support person can see what a
	// till was offered. Restaurants do not read release notes.
	Notes string `json:"notes,omitempty"`
	// ⚠️ When set, tills below this version must update before they are
	// offered anything else. Not used yet; here because the alternative is
	// discovering we need it during the incident that needs it.
	Minimum string `json:"minimum,omitempty"`
}

// TillRelease serves the manifest a paired till polls.
func (h *Handler) TillRelease(w http.ResponseWriter, r *http.Request) {
	dir := strings.TrimSpace(os.Getenv("TILL_RELEASE_DIR"))
	if dir == "" {
		// ⚠️ **404, not an empty manifest.** "No release configured" and "you
		// are up to date" are different facts, and a till told the second one
		// would stop asking. A 404 leaves it checking.
		http.NotFound(w, r)
		return
	}
	raw, err := os.ReadFile(filepath.Join(dir, "latest.json"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var rel tillRelease
	if err := json.Unmarshal(raw, &rel); err != nil {
		http.Error(w, "release manifest is not valid json", http.StatusInternalServerError)
		return
	}
	// ⚠️ Refused here rather than by the till. A manifest with no checksum is a
	// mistake on our side, and the till's own refusal would surface it as
	// "updates stopped working" on a hundred counters instead of once, here,
	// in a log we read.
	if strings.TrimSpace(rel.Version) == "" || strings.TrimSpace(rel.URL) == "" ||
		len(strings.TrimSpace(rel.SHA256)) != 64 {
		http.Error(w, "release manifest is incomplete", http.StatusInternalServerError)
		return
	}
	// Short, because a till checks a few times a day and a release should reach
	// the counters within one of those, not within a CDN's idea of an hour.
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(raw)
}

// TillDownload serves the installer itself.
//
// ⚠️ **Served from the same directory the manifest names**, so a release is one
// folder: drop the installer in, write `latest.json` beside it, and both halves
// are published at once. A manifest pointing at a file somebody forgot to
// upload is the failure this shape removes.
func (h *Handler) TillDownload(w http.ResponseWriter, r *http.Request) {
	dir := strings.TrimSpace(os.Getenv("TILL_RELEASE_DIR"))
	if dir == "" {
		http.NotFound(w, r)
		return
	}
	// ⚠️ The name is taken apart rather than trusted. This path is public and
	// the parameter reaches the filesystem: `..\..\etc` is the first thing an
	// automated scanner tries, and `filepath.Base` is what makes it a name
	// instead of a route.
	name := filepath.Base(strings.TrimSpace(r.URL.Query().Get("file")))
	if name == "" || name == "." || name == string(filepath.Separator) ||
		!strings.HasSuffix(strings.ToLower(name), ".exe") {
		http.NotFound(w, r)
		return
	}
	f, err := os.Open(filepath.Join(dir, name))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeContent(w, r, name, st.ModTime(), f)
}
