package handlers

import (
	"net/http"

	"keel-control/internal/config"
	mw "keel-control/internal/middleware"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func Router(h *Handler, cfg *config.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RealIP, chimw.Logger, chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Served to a suspended tenant's visitors by the edge.
	r.Get("/suspended", h.Suspended)
	r.Get("/suspended/*", h.Suspended)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	// Unauthenticated on purpose: Caddy calls tls-ask before any session
	// exists, and resolve tells a caller only what a visitor already sees.
	r.Route("/internal", func(r chi.Router) {
		// The reference logos keel.uz renders. Public: it is a marketing page,
		// and everything in it is already published on the customers' own
		// sites. Opt-in per tenant — see showcase.go.
		r.Get("/partners", h.Partners)

		// Measured uptime. Public on purpose: a status page behind a login is
		// a status page for the people who already know.
		r.Get("/status", h.StatusPage)

		r.Get("/resolve", h.Resolve)
		r.Get("/tls-ask", h.TLSAsk)

		// A tenant server connecting its owner's own domain. Not part of the
		// dashboard session — the caller is a container, and it authenticates
		// with a per-tenant token it was created with. The domain is only
		// accepted once it already resolves here, which is the ownership
		// proof; see domainlink.go.
		r.Post("/internal/domain", h.LinkDomain)
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", h.Login)

		r.Group(func(r chi.Router) {
			r.Use(mw.RequireAuth(cfg.JWTSecret))
			r.Get("/me", h.Me)
			r.Get("/stats", h.Stats)
			// "Does the collector even work?" — one press, and the answer is
			// the run's own report rather than another empty chart.
			r.Post("/stats/collect", h.Collect)
			r.Get("/tenants", h.ListTenants)
			r.Post("/tenants", h.CreateTenant)
			r.Get("/tenants/{id}", h.GetTenant)
			r.Put("/tenants/{id}", h.UpdateTenant)
			r.Post("/tenants/{id}/provision", h.ProvisionTenant)
			// One customer's live numbers, read from their own database.
			// Separate from GetTenant because it can be slow or fail, and the
			// card must render either way — see tenantlive.go.
			r.Get("/tenants/{id}/live", h.TenantLive)

			// Rolling update. GET also answers the question a deploy never
			// does: how many customers are still running old code, counted
			// from live container image ids rather than from what we believe
			// we deployed.
			r.Get("/rollout", h.GetRollout)
			r.Post("/rollout", h.StartRollout)

			// The invoice ledger: what each customer was billed and what was
			// actually collected. Cash until the MChJ exists — see
			// invoices.go, where that is a first-class fact rather than a
			// workaround.
			r.Get("/invoices", h.ListInvoices)
			r.Post("/tenants/{id}/invoices", h.IssueInvoice)
			r.Post("/invoices/{id}/pay", h.PayInvoice)
			r.Post("/invoices/{id}/void", h.VoidInvoice)
		})
	})
	return r
}
