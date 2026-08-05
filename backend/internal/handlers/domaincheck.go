package handlers

import (
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"restaurant-backend/internal/httpx"
)

// AdminDomainCheck tells the owner whether their own domain already points at
// this server.
//
// The DNS step is where bringing a domain goes wrong, and it goes wrong
// silently: the record is saved at the registrar, nothing appears to happen,
// and the owner has no way to tell "not propagated yet" from "typed the wrong
// thing". A button that answers that one question is the whole difference
// between a guide people follow and a guide they abandon.
//
// The expected address is not configured anywhere. It is whatever this site's
// **current** address resolves to — the tenant is already reachable there, so
// it is by definition the right answer, and there is no setting to keep in
// sync with reality.
func (h *Handler) AdminDomainCheck(w http.ResponseWriter, r *http.Request) {
	domain := normalizeHost(r.URL.Query().Get("domain"))
	if domain == "" {
		httpx.Error(w, http.StatusBadRequest, "domen kerak")
		return
	}

	expected := h.serverAddresses()
	found, err := net.LookupHost(domain)
	if err != nil {
		// Not an error of ours: a domain with no record yet is the normal state
		// halfway through the guide, and saying "no record found" is more use
		// than a 500.
		httpx.JSON(w, http.StatusOK, map[string]any{
			"domain": domain, "found": []string{}, "expected": expected, "ok": false,
		})
		return
	}
	found = uniqueSorted(found)

	// A match on any address is enough: a host may legitimately answer on both
	// IPv4 and IPv6, and the owner only has to get one of them right.
	ok := false
	for _, f := range found {
		for _, e := range expected {
			if f == e {
				ok = true
			}
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"domain": domain, "found": found, "expected": expected, "ok": ok,
	})
}

// serverAddresses resolves the address this site is already served on.
func (h *Handler) serverAddresses() []string {
	host := ""
	if u, err := url.Parse(h.Cfg.PublicBaseURL); err == nil {
		host = u.Hostname()
	}
	if host == "" || host == "localhost" {
		return []string{}
	}
	addrs, err := net.LookupHost(host)
	if err != nil {
		return []string{}
	}
	return uniqueSorted(addrs)
}

// normalizeHost strips what people paste: a scheme, a path, a port, a case.
func normalizeHost(d string) string {
	d = strings.ToLower(strings.TrimSpace(d))
	d = strings.TrimPrefix(strings.TrimPrefix(d, "https://"), "http://")
	if i := strings.IndexAny(d, "/:"); i >= 0 {
		d = d[:i]
	}
	if !strings.Contains(d, ".") {
		return ""
	}
	return d
}

func uniqueSorted(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}
