package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

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

// AdminDomainConnect asks the platform to start serving the owner's own
// domain — the last step of the guide, which used to be a phone call to us.
//
// This server cannot do the work itself: the hostname has to reach the edge
// and a certificate has to be requested for it, and both live in the control
// plane. So this is a relay with one honest job — carry the owner's request,
// and carry the answer back in words they can act on.
//
// **We check DNS here, and the control plane checks it again.** Not
// belt-and-braces: the check here exists so the owner gets a straight answer
// without waiting on a round trip, while the one over there is the actual
// authorisation. This server is the customer's side of the wire, and a
// customer's server must not be able to claim a domain by asserting that it
// verified one.
//
// On a standalone install — one restaurant on its own VPS, no Keel — there is
// no control plane and this says so rather than failing obscurely: that owner
// points DNS at their own server and nothing else is needed.
func (h *Handler) AdminDomainConnect(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	if h.Cfg.ControlURL == "" || h.Cfg.ControlToken == "" {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"ok": false, "unsupported": true,
			"error": "bu server mustaqil o'rnatilgan — domen shu serverning o'zida sozlanadi",
		})
		return
	}
	var req struct {
		Domain string `json:"domain"`
		Remove bool   `json:"remove"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	domain := normalizeHost(req.Domain)
	if domain == "" {
		httpx.Error(w, http.StatusBadRequest, "domen noto'g'ri")
		return
	}

	// The convenience check. Refusing here saves a round trip and, more
	// usefully, lets the answer name the addresses that were found — which is
	// the difference between "not propagated yet" and "typed the wrong thing".
	if !req.Remove {
		expected := h.serverAddresses()
		found, _ := net.LookupHost(domain)
		if len(expected) > 0 && !anyMatch(found, expected) {
			httpx.JSON(w, http.StatusOK, map[string]any{
				"ok": false, "domain": domain, "found": found, "expected": expected,
				"error": "domen hali bu serverga yo'naltirilmagan — DNS yozuvini tekshiring",
			})
			return
		}
	}

	res, err := h.callControl(r.Context(), map[string]any{
		"domain": domain, "remove": req.Remove,
	})
	if err != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"ok": false, "domain": domain, "error": err.Error(),
		})
		return
	}
	h.logAction(r, ActSettingsUpdate, "settings", "domain", "Domen", domain)
	httpx.JSON(w, http.StatusOK, res)
}

// callControl posts to the control plane with this tenant's own token.
func (h *Handler) callControl(ctx context.Context, body map[string]any) (map[string]any, error) {
	raw, _ := json.Marshal(body)
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		h.Cfg.ControlURL+"/internal/domain", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.Cfg.ControlToken)
	req.Header.Set("X-Keel-Tenant", h.Cfg.TenantSlug)

	res, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return nil, errors.New("platformaga ulanib bo'lmadi: " + err.Error())
	}
	defer res.Body.Close()
	out := map[string]any{}
	dec, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	_ = json.Unmarshal(dec, &out)
	if res.StatusCode >= 400 {
		msg, _ := out["error"].(string)
		if msg == "" {
			msg = strings.TrimSpace(string(dec))
		}
		// Returned as a value rather than a status: the caller shows it to a
		// restaurant owner, and "409" is not a sentence anybody can act on.
		return nil, errors.New(msg)
	}
	return out, nil
}

func anyMatch(found, expected []string) bool {
	for _, f := range found {
		for _, e := range expected {
			if f == e {
				return true
			}
		}
	}
	return false
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
