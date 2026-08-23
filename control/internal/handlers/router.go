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

		// What build a monoblock should be running, and the installer itself.
		// Public on purpose — see tillrelease.go.
		r.Get("/till/release", h.TillRelease)
		r.Get("/till/download", h.TillDownload)

		r.Get("/resolve", h.Resolve)
		r.Get("/tls-ask", h.TLSAsk)

		// A tenant server connecting its owner's own domain. Not part of the
		// dashboard session — the caller is a container, and it authenticates
		// with a per-tenant token it was created with. The domain is only
		// accepted once it already resolves here, which is the ownership
		// proof; see domainlink.go.
		r.Post("/domain", h.LinkDomain)
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", h.Login)

		r.Group(func(r chi.Router) {
			r.Use(mw.RequireAuth(cfg.JWTSecret))
			r.Get("/me", h.Me)

			// Console accounts and what each of them may see. Owner only — an
			// account that can grant itself more is not a boundary. See staff.go.
			r.Get("/staff", h.need("staff", h.ListStaff))
			r.Post("/staff", h.need("staff", h.CreateStaff))
			r.Put("/staff/{id}", h.need("staff", h.UpdateStaff))
			// Who did what. Owner only, for the same reason.
			r.Get("/console-log", h.need("log", h.ListConsoleLog))

			// An agent's day: where they plan to go, and what happened. Everybody
			// has these; only the roles that see every customer see everyone's.
			r.Get("/visits", h.ListVisits)
			r.Post("/visits", h.CreateVisit)
			r.Put("/visits/{id}", h.UpdateVisit)
			r.Delete("/visits/{id}", h.DeleteVisit)
			r.Get("/stats", h.need("stats", h.Stats))
			// "Does the collector even work?" — one press, and the answer is
			// the run's own report rather than another empty chart.
			r.Post("/stats/collect", h.need("stats", h.Collect))
			// The server's own vital signs. Disk is the one that decides when
			// the next customer stops being sellable.
			r.Get("/system", h.need("provision", h.System))
			// Frees what /system reports as reclaimable — orphaned images, stopped
			// containers, build cache. Never volumes: see PruneDocker.
			r.Post("/system/prune", h.need("provision", h.PruneDocker))
			r.Get("/tenants", h.ListTenants)
			r.Post("/tenants", h.CreateTenant)
			r.Get("/tenants/{id}", h.GetTenant)
			r.Put("/tenants/{id}", h.need("provision", h.UpdateTenant))
			r.Post("/tenants/{id}/provision", h.need("provision", h.ProvisionTenant))
			// ⚠️ Not behind "provision" like its neighbours — owner only, checked
			// inside the handler where the reason can be stated. This is the one
			// route that destroys something no other route can restore.
			r.Post("/tenants/{id}/purge", h.PurgeTenant)
			// One customer's live numbers, read from their own database.
			// Separate from GetTenant because it can be slow or fail, and the
			// card must render either way — see tenantlive.go.
			r.Get("/tenants/{id}/live", h.need("stats", h.TenantLive))

			// The page-layout constructor. The design is drawn here and written
			// into the tenant's own database — see handlers/design.go.
			r.Get("/tenants/{id}/design", h.need("provision", h.GetTenantDesign))
			r.Put("/tenants/{id}/design", h.need("provision", h.PutTenantDesign))
			r.Post("/tenants/{id}/design/publish", h.need("provision", h.PublishTenantDesign))
			r.Delete("/tenants/{id}/design", h.need("provision", h.RevertTenantDesign))
			// A short-lived link that shows the **unpublished** draft on the real
			// site, which is what the editor's iframe loads. See design.go for why
			// it is a token in the tenant's own database rather than a flag.
			r.Post("/tenants/{id}/design/preview", h.need("provision", h.PreviewTenantDesign))
			// One drawing reused across customers: the commercial point of the
			// tool. Applying one **copies** it, so editing a template later never
			// redraws a live site.
			// What a section can be asked. The console's settings panel is drawn
			// from this rather than written per type — see designtemplates.go.
			r.Get("/design-schema", h.need("provision", h.DesignSchema))
			r.Get("/design-templates", h.need("provision", h.ListDesignTemplates))
			r.Post("/design-templates", h.need("provision", h.CreateDesignTemplate))
			r.Delete("/design-templates/{id}", h.need("provision", h.DeleteDesignTemplate))

			// The customer's "download everything" button: off by default,
			// opened here for a written reason and a fixed number of days.
			// Written straight into their own database — see export.go for why
			// there is no path back the other way.
			r.Get("/tenants/{id}/export", h.need("provision", h.GetTenantExport))
			r.Put("/tenants/{id}/export", h.need("provision", h.PutTenantExport))

			// Rolling update. GET also answers the question a deploy never
			// does: how many customers are still running old code, counted
			// from live container image ids rather than from what we believe
			// we deployed.
			r.Get("/rollout", h.need("provision", h.GetRollout))
			r.Post("/rollout", h.need("provision", h.StartRollout))

			// The invoice ledger: what each customer was billed and what was
			// actually collected. Cash until the MChJ exists — see
			// invoices.go, where that is a first-class fact rather than a
			// workaround.
			r.Get("/invoices", h.need("billing", h.ListInvoices))
			r.Post("/tenants/{id}/invoices", h.need("billing", h.IssueInvoice))
			r.Post("/invoices/{id}/pay", h.need("billing", h.PayInvoice))
			r.Post("/invoices/{id}/void", h.need("billing", h.VoidInvoice))
		})
	})
	return r
}
