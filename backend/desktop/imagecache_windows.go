//go:build windows

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Pictures, kept on the machine that draws them.
//
// ⚠️ **The problem is measured, not assumed.** A menu tile asks for
// `/uploads/<file>?w=300`; on a restaurant's connection that is a round trip
// per dish, and the grid a cashier opens two hundred times an evening filled in
// visibly — tile by tile, every time, because a webview's memory cache does not
// survive the till being restarted at eight in the morning. The pictures
// themselves never change: an uploaded file has a random name, so a replaced
// photograph is a *different* URL (see the `immutable` header the server sends
// with them). A file that can never change is a file worth keeping.
//
// ⚠️ **Cached in the proxy rather than in the screens.** Every image already
// passes through here (proxy_windows.go forwards `/uploads/`), so the shared
// components need no second way of naming a picture and the browser till is
// unaffected. A cache the screens knew about would be a second URL scheme to
// get wrong, and it would have to be undone for the tablet on the floor.
//
// ⚠️ **`seed/` files are the one exception the server itself makes** — their
// names are fixed, so a new release can ship different bytes under the same
// name. They are cached for a day rather than forever; everything else has a
// random name and is kept until somebody clears it.
type imageCache struct {
	dir string
	// One flight per file. Two tiles asking for the same picture at once is
	// the ordinary case on a grid, and without this both would fetch it.
	mu      sync.Mutex
	pending map[string]chan struct{}
}

func newImageCache() *imageCache {
	dir := filepath.Join(configDir(), "images")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("rasm keshi ochilmadi: %v", err)
		return nil
	}
	return &imageCache{dir: dir, pending: map[string]chan struct{}{}}
}

// seedTTL is how long a fixed-name file is trusted.
const seedTTL = 24 * time.Hour

// path on disk for one request.
//
// ⚠️ **The query is part of the key.** `?w=300` and `?w=600` are different
// pictures of the same photograph — the server renders and caches them
// separately — and a key that ignored it would serve a card-sized image into
// the dish page, or a full-sized one into every tile of a grid.
func (c *imageCache) file(reqPath, query string) string {
	sum := sha256.Sum256([]byte(reqPath + "?" + query))
	name := hex.EncodeToString(sum[:])
	// The extension is kept so the folder can be looked at by a person, and so
	// the content type can be recovered without a second file beside each one.
	if ext := strings.ToLower(path.Ext(reqPath)); ext != "" && len(ext) <= 5 {
		name += ext
	}
	return filepath.Join(c.dir, name)
}

// fresh reports whether a cached file may still be used.
func fresh(reqPath string, info os.FileInfo) bool {
	if strings.Contains(reqPath, "/seed/") {
		return time.Since(info.ModTime()) < seedTTL
	}
	return true
}

// serve answers from disk when it can, and returns false when it cannot.
func (c *imageCache) serve(w http.ResponseWriter, r *http.Request) bool {
	if c == nil || r.Method != http.MethodGet {
		return false
	}
	name := c.file(r.URL.Path, r.URL.RawQuery)
	info, err := os.Stat(name)
	if err != nil || !fresh(r.URL.Path, info) {
		return false
	}
	f, err := os.Open(name)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	if ct := typeOf(r.URL.Path); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	// ⚠️ Told to the webview as well, so a second draw of the same grid does not
	// even reach this process.
	w.Header().Set("Cache-Control", "public, max-age=604800")
	http.ServeContent(w, r, name, info.ModTime(), f)
	return true
}

// store writes a fetched body to disk.
//
// ⚠️ **Written to a temporary file and renamed.** Two tiles arriving at once,
// or a till switched off mid-download, would otherwise leave a half-written
// picture that is served as a broken image for as long as the cache lives —
// the same reason the server's own thumbnail cache is atomic.
func (c *imageCache) store(reqPath, query string, body []byte) {
	if c == nil || len(body) == 0 {
		return
	}
	name := c.file(reqPath, query)
	tmp := name + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return
	}
	if err := os.Rename(tmp, name); err != nil {
		_ = os.Remove(tmp)
	}
}

func typeOf(p string) string {
	switch strings.ToLower(path.Ext(p)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	}
	return ""
}

// ---- Warming the cache ----

// warmImages downloads every picture the till draws, once, in the background.
//
// ⚠️ **On a schedule nobody waits for.** This runs after pairing and at every
// start, and it must never be something a cashier is held up by: it begins
// after the window is up and takes whatever time the connection gives it. The
// screens work throughout — an image that has not arrived yet is fetched the
// ordinary way and cached on its way past.
//
// ⚠️ **The widths are the ones the screens ask for**, not "all of them". The
// till's grid requests `?w=300`; warming a size nothing renders would download
// the whole menu twice and fill a monoblock's disk with pictures no screen will
// ever ask for.
func (a *App) warmImages() {
	if a.cache == nil || !a.cfg.paired() {
		return
	}
	go func() {
		// A quiet first minute: the window is opening, the menu is loading and
		// the agent is polling. A hundred downloads racing that is the one
		// thing somebody would notice.
		select {
		case <-a.ctx.Done():
			return
		case <-time.After(45 * time.Second):
		}
		paths, err := a.imagePaths()
		if err != nil {
			log.Printf("rasmlar ro'yxati olinmadi: %v", err)
			return
		}
		got := 0
		for _, p := range paths {
			for _, q := range []string{"w=300", ""} {
				if a.ctx.Err() != nil {
					return
				}
				if a.fetchInto(p, q) {
					got++
				}
			}
		}
		log.Printf("rasm keshi: %d ta fayl yuklandi (%d ta rasm)", got, len(paths))
	}()
}

// imagePaths asks the restaurant's own server what pictures its menu uses.
//
// ⚠️ **The menu endpoint rather than a new one.** It is already the list of
// everything the till draws, it is already reachable with the device token, and
// a purpose-built "list your images" route would be a second thing to keep in
// step with the menu.
func (a *App) imagePaths() ([]string, error) {
	body, err := a.getJSON("/api/v1/menu")
	if err != nil {
		return nil, err
	}
	var groups []struct {
		Items []struct {
			ImageURL string `json:"imageUrl"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &groups); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	out := []string{}
	for _, g := range groups {
		for _, it := range g.Items {
			p := uploadPath(it.ImageURL)
			if p == "" || seen[p] {
				continue
			}
			seen[p] = true
			out = append(out, p)
		}
	}
	// The restaurant's own logo and banners come from the profile, and they are
	// the first thing on the screen — the picture whose absence is most
	// visible while it loads.
	if body, err := a.getJSON("/api/v1/restaurant"); err == nil {
		var rest struct {
			LogoURL  string `json:"logoUrl"`
			CoverURL string `json:"coverUrl"`
			Banners  []struct {
				ImageURL string `json:"imageUrl"`
			} `json:"banners"`
		}
		if json.Unmarshal(body, &rest) == nil {
			for _, u := range append(
				[]string{rest.LogoURL, rest.CoverURL},
				bannerURLs(rest.Banners)...,
			) {
				if p := uploadPath(u); p != "" && !seen[p] {
					seen[p] = true
					out = append(out, p)
				}
			}
		}
	}
	return out, nil
}

func bannerURLs(in []struct {
	ImageURL string `json:"imageUrl"`
}) []string {
	out := make([]string, 0, len(in))
	for _, b := range in {
		out = append(out, b.ImageURL)
	}
	return out
}

// uploadPath reduces whatever the API returned to a path this proxy serves.
//
// ⚠️ **An absolute URL to somebody else's host is left out**, not rewritten: a
// restaurant that pasted a photograph from its own website is naming a server
// this cache has no business mirroring, and the screens load it directly.
func uploadPath(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "/uploads/") {
		return s
	}
	if i := strings.Index(s, "/uploads/"); i >= 0 && strings.HasPrefix(s, "http") {
		// Our own server, written absolutely. The path is what the proxy knows.
		return s[i:]
	}
	return ""
}

// fetchInto downloads one picture and stores it. Reports whether it arrived.
func (a *App) fetchInto(reqPath, query string) bool {
	if _, err := os.Stat(a.cache.file(reqPath, query)); err == nil {
		return false // already here
	}
	origin := a.serverOrigin()
	if origin == nil {
		return false
	}
	u := origin.String() + reqPath
	if query != "" {
		u += "?" + query
	}
	req, err := http.NewRequestWithContext(a.ctx, http.MethodGet, u, nil)
	if err != nil {
		return false
	}
	res, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return false
	}
	// ⚠️ Capped. A restaurant that uploaded a 40 MB photograph should not be
	// able to fill a monoblock's disk from a background loop nobody is watching.
	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return false
	}
	a.cache.store(reqPath, query, body)
	return true
}

// getJSON reads one endpoint from the restaurant's server with the device token.
func (a *App) getJSON(p string) ([]byte, error) {
	origin := a.serverOrigin()
	if origin == nil {
		return nil, os.ErrNotExist
	}
	req, err := http.NewRequestWithContext(a.ctx, http.MethodGet, origin.String()+p, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.cfg.Token)
	res, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	return io.ReadAll(io.LimitReader(res.Body, 8<<20))
}
