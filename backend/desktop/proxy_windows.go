//go:build windows

package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// The window serves the till's own files and forwards everything else to the
// restaurant's server.
//
// ⚠️ **This is what lets the shared screens run here unchanged.** lib/api.ts
// resolves to "/api/v1" in a browser and the comment above it explains why:
// single origin, no CORS, no mixed content. Inside a Wails window that path
// would resolve against the asset server and reach nothing — and pointing the
// screens at https://<slug>.keel.uz directly would fail on CORS instead, since
// the window's origin is not a domain any tenant lists. Neither is fixable in
// the shared code without giving it a second networking model to carry.
//
// A proxy keeps the model the screens already assume: one origin, everything
// under it. The address is per-restaurant, so this is decided at request time
// from the pairing, not at build time.
//
// ⚠️ **No credential is injected here.** The till alternates between the device
// token and the unlocked person's token per call (lib/api.ts, tillAuth), which
// is what makes a void carry the name of whoever is standing there. A proxy
// that stamped one Authorization header on everything would quietly undo the
// PIN.
func (a *App) proxy(next http.Handler) http.Handler {
	rp := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			origin := a.serverOrigin()
			if origin == nil {
				return
			}
			pr.Out.URL.Scheme = origin.Scheme
			pr.Out.URL.Host = origin.Host
			pr.Out.Host = origin.Host
			// ⚠️ **A name, not a credential**, which is what the note above
			// forbids. The server records it against this branch's register row
			// so the panel can say which computer each one is; it authorises
			// nothing and is never read as permission. Stamped here because the
			// screens are shared with the browser till, and a browser cannot
			// know the name of the machine it is running on.
			setTillHost(pr.Out.Header)
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !forwarded(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		// ⚠️ **Pictures come off the disk when they are there.** Every image the
		// till draws passes through here, and on a restaurant's connection the
		// menu grid filled in tile by tile every time it was opened. An uploaded
		// file has a random name, so it can never change — see
		// imagecache_windows.go.
		if isImage(r.URL.Path) && a.cache.serve(w, r) {
			return
		}
		if a.serverOrigin() == nil {
			// Not paired yet. ⚠️ 503 rather than a silent pass to the asset
			// server, which would answer the till's API calls with index.html
			// and produce a JSON parse error somewhere far away from the cause.
			http.Error(w, "till is not paired", http.StatusServiceUnavailable)
			return
		}
		if isImage(r.URL.Path) {
			// ⚠️ Recorded on the way past, so a picture nobody warmed — a dish
			// added this afternoon — is still only fetched once.
			cw := &cachingWriter{ResponseWriter: w}
			rp.ServeHTTP(cw, r)
			// ⚠️ **Written once the response is complete, not per chunk.** The
			// first version stored on every Write, which rewrote the whole file
			// for each packet of a photograph — and `store` renames a finished
			// file into place, so every one of those was a full write and
			// rename of a file that was still arriving.
			if cw.keep {
				a.cache.store(r.URL.Path, r.URL.RawQuery, cw.buf)
			}
			return
		}
		rp.ServeHTTP(w, r)
	})
}

// isImage reports whether a forwarded path is worth keeping on disk.
//
// ⚠️ **By extension, not by the response's content type.** The decision has to
// be made before the request goes out — that is what lets a cached file answer
// without a round trip at all — and `/uploads/` carries nothing but files a
// restaurant uploaded.
func isImage(p string) bool {
	if !strings.HasPrefix(p, "/uploads/") {
		return false
	}
	return typeOf(p) != ""
}

// cachingWriter copies a successful image response to disk as it is written.
//
// ⚠️ **Only 200s, and only the body.** A 404 stored on disk would be served for
// a week after somebody fixed the upload behind it, and a redirect kept as an
// image would be a broken picture with no way to explain itself.
type cachingWriter struct {
	http.ResponseWriter
	buf   []byte
	keep  bool
	wrote bool
}

func (w *cachingWriter) WriteHeader(code int) {
	w.wrote = true
	w.keep = code == http.StatusOK
	w.ResponseWriter.WriteHeader(code)
}

func (w *cachingWriter) Write(b []byte) (int, error) {
	// A handler that never calls WriteHeader has implicitly sent 200.
	if !w.wrote {
		w.wrote, w.keep = true, true
	}
	n, err := w.ResponseWriter.Write(b)
	// ⚠️ Only what actually reached the webview is kept. Buffering what we
	// *meant* to send would store a whole picture for a connection that dropped
	// halfway, and it would be served from disk as a broken image for as long as
	// the cache lives.
	if err != nil {
		w.keep = false
		return n, err
	}
	if w.keep && len(w.buf)+n <= 8<<20 {
		w.buf = append(w.buf, b[:n]...)
	} else if w.keep {
		// Too big to keep. Not an error — it is still served, just not stored.
		w.keep = false
	}
	return n, err
}

// forwarded reports whether a path belongs to the restaurant's server.
//
// ⚠️ An allowlist of two prefixes, not "everything the asset server does not
// have". A catch-all would turn every mistyped asset path into a request to the
// restaurant's server, and the 404 would come back looking like a backend
// fault.
func forwarded(path string) bool {
	return strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/uploads/")
}

// serverOrigin is the scheme and host this till is paired with, or nil.
func (a *App) serverOrigin() *url.URL {
	if a.cfg.Server == "" {
		return nil
	}
	u, err := url.Parse(a.cfg.Server)
	if err != nil || u.Host == "" {
		return nil
	}
	// cfg.Server carries the API path ("https://osh.keel.uz/api/v1"); the proxy
	// needs the origin, because the request already has its own path.
	return &url.URL{Scheme: u.Scheme, Host: u.Host}
}
