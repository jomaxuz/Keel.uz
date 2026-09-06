package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"time"

	"keel-control/internal/caddy"
	"keel-control/internal/httpx"
	"keel-control/internal/models"
	"keel-control/internal/provision"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Bringing a tenant up is two independent steps, and they fail independently:
//
//	1. the container — the customer's own server, with its own database
//	2. the edge — Caddy learning which hostnames belong to it
//
// Neither is allowed to fail the request that triggered it. A tenant recorded
// but not started is recoverable with one button; a create that 500s halfway
// leaves an operator guessing which half happened. This is the same rule the
// POS integration follows for orders, for the same reason.

// provisionEnabled reports whether this deployment can actually start things.
// On a laptop it cannot, and that must not be an error — the control plane is
// still useful as a record.
func (h *Handler) provisionEnabled() bool { return h.Docker != nil }

// Provision brings one tenant to the state its status implies.
//
// `rebuild` replaces a running container instead of leaving it alone. That is
// needed whenever something baked into its environment changed — in practice,
// its domains: `PUBLIC_BASE_URL` and `CORS_ORIGINS` are read from the primary
// domain at creation, so a customer who adds their own domain would otherwise
// get a site that answers on the new address while every absolute link it
// prints — payment callbacks, order-tracking URLs, QR codes — still names the
// old one. The container is replaced; the uploads volume and the database are
// not touched.
func (h *Handler) provisionTenant(ctx context.Context, t *models.Tenant, rebuild bool) error {
	if h.Docker == nil {
		return nil
	}
	if t.Offline() {
		return h.Docker.Stop(ctx, t.Slug)
	}
	primary := ""
	if len(t.Domains) > 0 {
		primary = t.Domains[0]
	}
	spec := provision.Spec{
		Slug:          t.Slug,
		DBName:        t.DBName(),
		JWTSecret:     t.JWTSecret,
		BusinessType:  t.BusinessType,
		BrandName:     t.Name,
		AdminUsername: t.AdminUsername,
		AdminPassword: t.AdminPassword,
		PrimaryDomain: primary,
		// Over the shared docker network by name, never over the public
		// internet: this call adds a hostname to the edge, and it has no
		// business being reachable from outside.
		// The base, not a path: the tenant appends whichever internal endpoint
		// it needs. Naming one here made two files guess at the other half,
		// and both guessed wrong.
		ControlURL:   "http://" + h.Cfg.ControlHost,
		ControlToken: LinkToken(h.Cfg.JWTSecret, t.Slug),
	}
	run := h.Docker.Ensure
	if rebuild {
		run = h.Docker.Recreate
	}
	if err := run(ctx, spec); err != nil {
		return err
	}
	// Started is not the same as working. A server that cannot reach its
	// database stays alive for its whole connection timeout, so only the
	// tenant's own health endpoint answers the question honestly.
	return h.Docker.WaitHealthy(ctx, t.Slug, 40*time.Second)
}

// syncEdge rewrites the whole Caddy config from the tenant list.
//
// Every tenant, every time — not a patch for the one that changed. A config
// derived from one source cannot drift from it, and drift here means a
// customer's domain serving somebody else's shop.
func (h *Handler) syncEdge(ctx context.Context) error {
	if h.Cfg.CaddyAdmin == "" {
		return nil
	}
	cur, err := h.Store.Tenants.Find(ctx, bson.M{})
	if err != nil {
		return err
	}
	var tenants []models.Tenant
	if err := cur.All(ctx, &tenants); err != nil {
		return err
	}
	sites := make([]caddy.Site, 0, len(tenants))
	for _, t := range tenants {
		sites = append(sites, caddy.Site{
			Slug:      t.Slug,
			Domains:   t.Domains,
			Suspended: t.Offline(),
		})
	}
	cfg := caddy.Render(sites, caddy.Options{
		Frontend:     h.Cfg.FrontendHost,
		Control:      h.Cfg.ControlHost,
		MainDomains:  []string{h.Cfg.BaseDomain, "www." + h.Cfg.BaseDomain},
		MainUpstream: h.Cfg.MainUpstream,
		Email:        h.Cfg.CaddyEmail,
	})
	return caddy.Apply(ctx, h.Cfg.CaddyAdmin, cfg)
}

// SyncEdge re-renders the whole edge configuration from the tenant list.
//
// Exported because it has to happen at boot as well as on every tenant change:
// Caddy re-reads its bootstrap file when it restarts, so after a reboot the
// edge knows about no customers at all until something pushes the real config.
// Nothing else would — `apply` only runs when a tenant is edited, and a server
// can go months without that.
func (h *Handler) SyncEdge(ctx context.Context) error { return h.syncEdge(ctx) }

// apply runs both halves and records the outcome on the tenant.
//
// The stored admin password is cleared once the container has started: the
// tenant server has seeded its owner by then and will never read it again, so
// keeping a plaintext password would be storing a secret for no purpose. If it
// ever needs resetting, that is `cmd/adminreset` on the tenant itself.
func (h *Handler) apply(ctx context.Context, t *models.Tenant, rebuild bool) {
	set := bson.M{"updatedAt": time.Now()}
	var failures []string

	// ⚠️ **Both halves run, always.** They used to be chained with `else if`, so
	// a container that would not come up meant the edge was never re-rendered —
	// and `syncEdge` rewrites the config for **every** tenant, so one sick
	// customer quietly froze domain changes across the whole platform.
	//
	// That is how a real domain went missing: the owner connected it, the
	// database took it, the response said `ok`, the console listed it, and Caddy
	// never heard of it. Nothing anywhere said otherwise — the only symptom was
	// a site that would not load over HTTPS, which reads as a DNS problem or a
	// slow certificate, i.e. as somebody else's fault.
	//
	// They are also genuinely independent: the edge config is derived from the
	// tenant list in Mongo, not from anything the container does.
	if err := h.provisionTenant(ctx, t, rebuild); err != nil {
		failures = append(failures, "konteyner: "+err.Error())
	}
	if err := h.syncEdge(ctx); err != nil {
		failures = append(failures, "chekka: "+err.Error())
	}
	// Labelled, because "which half failed" is the first question and the two
	// have entirely different fixes.
	failure := strings.Join(failures, "; ")

	if failure != "" {
		set["provisionStatus"] = "failed"
		set["provisionError"] = failure
		log.Printf("provision %s: %s", t.Slug, failure)
	} else if h.provisionEnabled() {
		now := time.Now()
		set["provisionStatus"] = "ready"
		set["provisionError"] = ""
		set["provisionedAt"] = now
		if !t.Offline() && t.AdminPassword != "" {
			set["adminPassword"] = ""
		}
	}
	if _, err := h.Store.Tenants.UpdateByID(ctx, t.ID, bson.M{"$set": set}); err != nil {
		log.Printf("provision %s: natijani yozib bo'lmadi: %v", t.Slug, err)
	}
}

// ProvisionTenant is the retry button.
func (h *Handler) ProvisionTenant(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var t models.Tenant
	if err := h.Store.Tenants.FindOne(r.Context(), bson.M{"_id": id}).Decode(&t); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	if !h.provisionEnabled() && h.Cfg.CaddyAdmin == "" {
		httpx.Error(w, http.StatusBadRequest,
			"bu serverda avtomatik ishga tushirish yoqilmagan (DOCKER_SOCKET va CADDY_ADMIN)")
		return
	}
	// The retry button rebuilds: it is pressed when something is wrong, and
	// "start the container that is already running" is rarely the fix.
	h.apply(r.Context(), &t, true)
	h.GetTenant(w, r)
}

// containerStates is every tenant's live container state, keyed by slug.
//
// Empty map when this deployment has no Docker socket, and that is the honest
// answer rather than an error: the control plane is still useful as a record on
// a laptop. Callers must read a missing entry as "unknown", never as "down" —
// the difference decides whether every row on a developer's machine lights up
// with a warning.
//
// A tenant that Docker *does* know nothing about is reported as `absent`, which
// is the one thing this exists to surface: a customer marked ready, with no
// container, and a domain answering 502.
func (h *Handler) containerStates(ctx context.Context) map[string]string {
	if h.Docker == nil {
		return map[string]string{}
	}
	states, err := h.Docker.States(ctx)
	if err != nil {
		// Docker itself is unreachable. Nothing is known about any container,
		// and saying "all down" would be a platform-wide false alarm.
		return map[string]string{}
	}
	out := make(map[string]string, len(states))
	for slug, st := range states {
		out[slug] = st.Status
	}
	return out
}

// stateOf reads one tenant out of that map. A slug Docker did not list has no
// container at all — the answer, not a gap.
func stateOf(states map[string]string, slug string) string {
	if len(states) == 0 {
		return ""
	}
	if s, ok := states[slug]; ok {
		return s
	}
	return "absent"
}

// containerStatus asks Docker, tolerating an answer of "no Docker here".
func (h *Handler) containerStatus(ctx context.Context, slug string) string {
	if h.Docker == nil {
		return ""
	}
	st, err := h.Docker.Status(ctx, slug)
	if err != nil {
		return "unknown"
	}
	return st.Status
}

// Suspended is what a customer behind on payment sees.
//
// A page rather than silence: an unreachable domain looks like an outage, and
// the restaurant will phone about the wrong thing.
func (h *Handler) Suspended(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`<!doctype html><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Vaqtincha to'xtatilgan</title>
<style>
 body{margin:0;min-height:100svh;display:grid;place-items:center;background:#05101a;color:#ecf1f5;
      font:16px/1.6 system-ui,sans-serif;padding:24px}
 .b{max-width:30rem;text-align:center}
 h1{font-size:1.5rem;margin:0 0 .75rem}
 p{margin:0;color:#8094a4}
 a{color:#f5a524}
</style>
<div class="b">
 <h1>Sayt vaqtincha to'xtatilgan</h1>
 <p>Ma'lumotlaringiz saqlanib turibdi. Qayta ishga tushirish uchun
 <a href="https://t.me/keeluz">Telegramda yozing</a>.</p>
</div>`))
}

// newSecret makes a per-tenant JWT secret.
func newSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Never seen in practice; a predictable secret is worse than a crash.
		panic("provision: tasodifiy sonlar mavjud emas: " + err.Error())
	}
	return hex.EncodeToString(b)
}
