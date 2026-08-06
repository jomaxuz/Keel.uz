package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"keel-control/internal/httpx"
	"keel-control/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Connecting a restaurant's own domain, without us in the middle.
//
// Until now the last step was manual: the owner pointed DNS at us, and then
// somebody here typed the domain into the console. That is the step that does
// not scale and, worse, the one that makes an owner wait — usually overnight,
// for a change they already made.
//
// The hard part is not the plumbing, it is **why we are allowed to believe
// them**. A restaurant that can name any domain can make us ask Let's Encrypt
// for a certificate in somebody else's name. So:
//
//   - **DNS is the proof of ownership.** A domain is accepted only once it
//     already resolves to this server. Pointing a domain here is something
//     only the person holding the registrar account can do, which is exactly
//     the claim being made. Vercel and Netlify accept the same evidence.
//
//   - **The control plane verifies it itself.** The tenant checks too, so the
//     owner gets an answer without a round trip, but that check is a
//     convenience. The tenant server is the customer's side of the wire; a
//     compromised or merely buggy one must not be able to claim a domain by
//     saying it verified.
//
//   - **Our own names are never claimable.** `keel.uz` and anything under it
//     belong to the platform. Without this, a customer could point a CNAME at
//     us, "connect" admin.keel.uz, and be served for it.
//
// The token is HMAC(secret, slug) — computed, never stored, like the kiosk
// codes in the tenant app. A tenant container therefore gets its credential
// simply by being created, and nothing has to be migrated or kept in sync.

// linkToken is the credential a tenant uses to talk to the control plane.
func LinkToken(secret, slug string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte("domain-link:" + slug))
	return hex.EncodeToString(m.Sum(nil))
}

// tenantFromLink authenticates a call from a tenant server.
func (h *Handler) tenantFromLink(r *http.Request) (*models.Tenant, error) {
	slug := strings.TrimSpace(r.Header.Get("X-Keel-Tenant"))
	token := strings.TrimPrefix(strings.TrimSpace(r.Header.Get("Authorization")), "Bearer ")
	if slug == "" || token == "" {
		return nil, errUnauthorized
	}
	want := LinkToken(h.Cfg.JWTSecret, slug)
	// Constant time: this endpoint is reachable from every tenant container,
	// and a byte-wise comparison leaks how much of a guess was right.
	if subtle.ConstantTimeCompare([]byte(token), []byte(want)) != 1 {
		return nil, errUnauthorized
	}
	var t models.Tenant
	if err := h.Store.Tenants.FindOne(r.Context(), bson.M{"slug": slug}).Decode(&t); err != nil {
		return nil, errUnauthorized
	}
	return &t, nil
}

type linkError string

func (e linkError) Error() string { return string(e) }

const errUnauthorized = linkError("unauthorized")

// How many domains one customer may connect. Not a technical limit — it is the
// blast radius of a mistake, and no restaurant needs a sixth address.
const maxTenantDomains = 6

// LinkDomain is the tenant server asking us to serve its owner's own domain.
//
// Idempotent: an owner who presses the button twice, or whose first request
// timed out after it succeeded, gets the same answer rather than an error about
// a domain that is already theirs.
func (h *Handler) LinkDomain(w http.ResponseWriter, r *http.Request) {
	t, err := h.tenantFromLink(r)
	if err != nil {
		// Quietly, and without saying which half was wrong.
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		Domain string `json:"domain"`
		// Remove instead of add. Same authority: an owner who can connect a
		// domain can disconnect it.
		Remove bool `json:"remove"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	domain := normalizeDomain(req.Domain)
	if domain == "" {
		httpx.Error(w, http.StatusBadRequest, "domen noto'g'ri")
		return
	}

	if req.Remove {
		h.unlinkDomain(w, r, t, domain)
		return
	}

	// Ours, and never claimable. A CNAME at us plus a "connect" would
	// otherwise hand a customer the platform's own hostnames.
	if isPlatformDomain(domain, h.Cfg.BaseDomain) {
		httpx.Error(w, http.StatusBadRequest,
			"bu domen platformaniki — o'z domeningizni kiriting")
		return
	}
	if slices.Contains(t.Domains, domain) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"ok": true, "domain": domain, "domains": t.Domains,
			"note": "allaqachon ulangan",
		})
		return
	}
	if len(t.Domains) >= maxTenantDomains {
		httpx.Error(w, http.StatusBadRequest, "domenlar soni chegarasiga yetdi")
		return
	}

	// The ownership proof, checked here rather than taken on trust from the
	// tenant. Pointing a domain at this server is something only the registrar
	// account holder can do, and that is precisely the claim.
	expected := h.serverAddresses()
	if len(expected) == 0 {
		httpx.Error(w, http.StatusServiceUnavailable,
			"server manzilini aniqlab bo'lmadi — birozdan keyin urinib ko'ring")
		return
	}
	found, _ := net.LookupHost(domain)
	if !anyMatch(found, expected) {
		httpx.JSON(w, http.StatusConflict, map[string]any{
			"ok": false, "domain": domain, "found": found, "expected": expected,
			"error": "domen hali bu serverga yo'naltirilmagan",
		})
		return
	}

	res, err := h.Store.Tenants.UpdateOne(r.Context(),
		// Guarded by the length so two simultaneous requests cannot push a
		// customer past the cap between the check above and the write.
		bson.M{"_id": t.ID, "domains": bson.M{"$not": bson.M{"$size": maxTenantDomains}}},
		bson.M{
			"$addToSet": bson.M{"domains": domain},
			"$set":      bson.M{"updatedAt": time.Now()},
		})
	if mongo.IsDuplicateKeyError(err) {
		// The unique index on domains. Somebody else already serves this name,
		// and two tenants answering one hostname is the failure this whole
		// endpoint exists to avoid.
		httpx.Error(w, http.StatusConflict, "bu domen boshqa mijozda band")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res.MatchedCount == 0 {
		httpx.Error(w, http.StatusBadRequest, "domenlar soni chegarasiga yetdi")
		return
	}

	h.afterDomainChange(r, t.ID)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"ok": true, "domain": domain, "domains": append(t.Domains, domain),
	})
}

func (h *Handler) unlinkDomain(w http.ResponseWriter, r *http.Request,
	t *models.Tenant, domain string) {
	// The first domain is the tenant's own <slug>.keel.uz — the address that
	// works when everything else is misconfigured, and the one every absolute
	// link falls back to. Removing it would leave a customer with no way in.
	if len(t.Domains) > 0 && t.Domains[0] == domain {
		httpx.Error(w, http.StatusBadRequest,
			"asosiy manzilni o'chirib bo'lmaydi")
		return
	}
	if _, err := h.Store.Tenants.UpdateByID(r.Context(), t.ID, bson.M{
		"$pull": bson.M{"domains": domain},
		"$set":  bson.M{"updatedAt": time.Now()},
	}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.afterDomainChange(r, t.ID)
	rest := make([]string, 0, len(t.Domains))
	for _, d := range t.Domains {
		if d != domain {
			rest = append(rest, d)
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"ok": true, "domain": domain, "domains": rest, "removed": true,
	})
}

// afterDomainChange teaches the edge about the new name and rebuilds the
// container.
//
// The rebuild is not optional: PUBLIC_BASE_URL and CORS_ORIGINS are read from
// the primary domain when the container is created, so a tenant that gained a
// domain without one would answer on the new address while every absolute link
// it prints — payment callbacks, order tracking, QR codes — still named the old
// one. That bug shipped once already.
func (h *Handler) afterDomainChange(r *http.Request, id any) {
	var t models.Tenant
	if err := h.Store.Tenants.FindOne(r.Context(), bson.M{"_id": id}).Decode(&t); err != nil {
		return
	}
	h.apply(r.Context(), &t, true)
}

// isPlatformDomain reports whether a name belongs to us rather than to a
// customer — the base domain itself, or anything under it.
func isPlatformDomain(domain, base string) bool {
	base = strings.ToLower(strings.TrimSpace(base))
	if base == "" {
		return false
	}
	return domain == base || strings.HasSuffix(domain, "."+base)
}

func anyMatch(found, expected []string) bool {
	for _, f := range found {
		if slices.Contains(expected, f) {
			return true
		}
	}
	return false
}

// serverAddresses resolves the address this platform is served on.
//
// Taken from the base domain rather than from a configured IP: the platform is
// already reachable there, so it is by definition the right answer, and there
// is no setting that can drift away from reality.
func (h *Handler) serverAddresses() []string {
	host := strings.TrimSpace(h.Cfg.BaseDomain)
	if host == "" {
		if u, err := url.Parse(h.Cfg.MainUpstream); err == nil {
			host = u.Hostname()
		}
	}
	if host == "" || host == "localhost" {
		return nil
	}
	addrs, err := net.LookupHost(host)
	if err != nil {
		return nil
	}
	return addrs
}
