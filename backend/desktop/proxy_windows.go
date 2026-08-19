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
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !forwarded(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if a.serverOrigin() == nil {
			// Not paired yet. ⚠️ 503 rather than a silent pass to the asset
			// server, which would answer the till's API calls with index.html
			// and produce a JSON parse error somewhere far away from the cause.
			http.Error(w, "till is not paired", http.StatusServiceUnavailable)
			return
		}
		rp.ServeHTTP(w, r)
	})
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
