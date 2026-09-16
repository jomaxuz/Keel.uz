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

		// The blog keel.uz renders. Public because it is a marketing page —
		// the same reasoning the partner logos above are public under.
		r.Get("/blog", h.BlogList)
		r.Get("/blog/{slug}", h.BlogRead)
		r.Get("/blog/image/{id}", h.BlogImage)
		// One reading, told by the reader's browser — see blog.go for why it
		// is not part of the read itself.
		r.Post("/blog/{slug}/view", h.BlogCountView)

		r.Get("/resolve", h.Resolve)
		r.Get("/tls-ask", h.TLSAsk)

		// A tenant server connecting its owner's own domain. Not part of the
		// dashboard session — the caller is a container, and it authenticates
		// with a per-tenant token it was created with. The domain is only
		// accepted once it already resolves here, which is the ownership
		// proof; see domainlink.go.
		r.Post("/domain", h.LinkDomain)

		// The morning briefing: a tenant sends the figures it computed, we hold
		// the key and send back sentences. Same credential as the domain link.
		// Support: a restaurant's question, and the request its panel holds
		// open waiting for an answer. Same per-tenant credential as the
		// briefing — the customer's browser never reaches this service.
		r.Post("/support/ask", h.SupportAsk)
		r.Get("/support/threads", h.SupportThreads)
		r.Get("/support/thread", h.SupportRead)
		r.Get("/support/wait", h.SupportWait)
		// The assistant's first answer, written only from help articles the
		// panel sent with the question — see supportai.go.
		r.Post("/support/assist", h.SupportAssist)

		// What broke, forwarded by the tenant's own server. ⚠️ The restaurant is
		// read from this credential and never from the body — the list is
		// evidence, and a slug an app could set is a list any browser could
		// write into under somebody else's name.
		r.Post("/report", h.Report)

		r.Post("/insight", h.Briefing)
		// The owner's own question, answered from figures their server
		// computed. ⚠️ Same credential and same daily allowance as the
		// briefing: a restaurant buys "the assistant", not two of them. See
		// advisor.go.
		r.Post("/advise", h.Advise)
		// What to advertise, from the same figures and under the same
		// credential. ⚠️ Its **own** daily allowance, unlike the advisor's:
		// this is a separately bought add-on, and sharing the assistant's pot
		// would make a restaurant's campaign questions eat its briefings. See
		// adsadvice.go.
		r.Post("/ads-advice", h.AdsAdvice)
		r.Post("/campaign-text", h.CampaignText)
		// Reading a menu off a page the owner pasted. ⚠️ The fallback only —
		// the tenant parses schema.org data itself first, which is exact and
		// free; this is for the pages that publish none.
		r.Post("/menu-extract", h.MenuExtract)
		r.Post("/ai-quota", h.AIQuota)
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
			r.Get("/staff/{id}", h.need("staff", h.GetStaff))
			r.Put("/staff/{id}", h.need("staff", h.UpdateStaff))
			// ⚠️ Gone for good, unlike switching an account off — see DeleteStaff.
			r.Delete("/staff/{id}", h.need("staff", h.DeleteStaff))
			// Who did what. Owner only, for the same reason.
			r.Get("/console-log", h.need("log", h.ListConsoleLog))

			// ⚠️ **Owner and support only.** This used to be open to every role
			// ("whoever is at a desk answers"); with a dedicated support role a
			// sales account answering a technical question is a promise the
			// platform then has to keep. See models.CanSupport.
			// Crash reports. ⚠️ Beside support rather than under stats: this is
			// the same queue read from the other end — what broke, arriving
			// before somebody writes in to say so.
			r.Get("/reports", h.need("support", h.ConsoleReports))
			r.Get("/reports/{id}", h.need("support", h.ConsoleReport))
			r.Post("/reports/{id}/resolve", h.need("support", h.ConsoleReportResolve))

			r.Get("/support", h.need("support", h.ConsoleSupportList))
			r.Get("/support/{id}", h.need("support", h.ConsoleSupportThread))
			r.Post("/support/{id}/reply", h.need("support", h.ConsoleSupportReply))

			// An agent's day: where they plan to go, and what happened. Everybody
			// has these; only the roles that see every customer see everyone's.
			// Who sends us customers from outside, and what we owe them.
			// ⚠️ Owner only ("partners"): this is money and contracts, and neither
			// a salesperson nor the platform admin negotiates them. See
			// handlers/referrals.go.
			// Search engines. ⚠️ Behind "provision" rather than "billing": it
			// touches the platform's own presence, not anybody's money, and it
			// is the same hands that run domains and deploys.
			// ⚠️ **Owner and admin**: the blog is published under our name, and
			// the hands that run domains and deploys are the hands that run it.
			// Search settings (below) are owner only.
			r.Get("/blog", h.need("blog", h.ConsoleBlogList))
			r.Post("/blog", h.need("blog", h.ConsoleBlogSave))
			r.Delete("/blog/{id}", h.need("blog", h.ConsoleBlogDelete))
			r.Post("/blog/upload", h.need("blog", h.ConsoleBlogUpload))

			r.Get("/seo", h.need("seo", h.SeoStatus))
			r.Post("/seo/indexnow", h.need("seo", h.SeoPing))

			r.Get("/referrers", h.need("partners", h.ListReferrers))
			r.Get("/referrers/{id}", h.need("partners", h.GetReferrer))
			r.Post("/referrers", h.need("partners", h.CreateReferrer))
			r.Put("/referrers/{id}", h.need("partners", h.UpdateReferrer))

			// ⚠️ Behind "tenants": a support-only account has no customers and no
			// visits, and an ungated route is the one it would reach anyway.
			r.Get("/visits", h.need("tenants", h.ListVisits))
			r.Post("/visits", h.need("tenants", h.CreateVisit))
			r.Put("/visits/{id}", h.need("tenants", h.UpdateVisit))
			r.Delete("/visits/{id}", h.need("tenants", h.DeleteVisit))
			r.Get("/stats", h.need("stats", h.Stats))
			// The whole platform over a window somebody chooses, as opposed to
			// /stats, which answers the billing month. Same permission: an
			// agent sees their own customers and no platform figures at all.
			r.Get("/overview", h.need("stats", h.Overview))
			// The same platform, split by what our customers actually are —
			// a restaurant and a shop averaged together describe neither.
			r.Get("/business", h.need("stats", h.BusinessBreakdown))
			// "Does the collector even work?" — one press, and the answer is
			// the run's own report rather than another empty chart.
			r.Post("/stats/collect", h.need("stats", h.Collect))
			// The server's own vital signs. Disk is the one that decides when
			// the next customer stops being sellable.
			r.Get("/system", h.need("provision", h.System))
			// Frees what /system reports as reclaimable — orphaned images, stopped
			// containers, build cache. Never volumes: see PruneDocker.
			r.Post("/system/prune", h.need("provision", h.PruneDocker))
			r.Get("/tenants", h.need("tenants", h.ListTenants))
			// What the assistant has cost, per tenant. Behind the console login
			// with everything else here — it is our spending, not a customer's.
			r.Get("/ai-usage", h.need("tenants", h.AIUsage))
			r.Post("/tenants", h.need("tenants", h.CreateTenant))
			r.Get("/tenants/{id}", h.need("tenants", h.GetTenant))
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
			// ⚠️ Both behind "provision", including the read: what a customer
			// pays for is a commercial fact, and an agent who can see the rung
			// can quote against it.
			r.Get("/tenants/{id}/till", h.need("provision", h.GetTenantTill))
			r.Put("/tenants/{id}/till", h.need("provision", h.PutTenantTill))

			// ---- The restaurant's own Android app ----
			//
			// ⚠️ **Behind `provision`, because it is the same kind of act**:
			// it spends the machine's memory for ten minutes and it mints a
			// signing key that can never be replaced. Reading the history is
			// open to anybody who can see the tenant.
			r.Get("/tenants/{id}/app-builds", h.need("tenants", h.AppBuilds))
			r.Post("/tenants/{id}/app-build", h.need("provision", h.StartAppBuild))
			r.Put("/tenants/{id}/android-app", h.need("provision", h.SetAndroidAppID))
			// ⚠️ **Its own path, not under the tenant.** The download deletes
			// the artifact, so it is addressed by the build it consumes rather
			// than by the customer it belongs to — a URL that named the tenant
			// would invite "give me the latest one", which is a file that may
			// already be gone.
			r.Get("/app-builds/{buildId}/download", h.need("provision", h.DownloadAppBuild))
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
