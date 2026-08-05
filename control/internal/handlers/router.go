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
		r.Get("/resolve", h.Resolve)
		r.Get("/tls-ask", h.TLSAsk)
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", h.Login)

		r.Group(func(r chi.Router) {
			r.Use(mw.RequireAuth(cfg.JWTSecret))
			r.Get("/me", h.Me)
			r.Get("/stats", h.Stats)
			r.Get("/tenants", h.ListTenants)
			r.Post("/tenants", h.CreateTenant)
			r.Get("/tenants/{id}", h.GetTenant)
			r.Put("/tenants/{id}", h.UpdateTenant)
			r.Post("/tenants/{id}/provision", h.ProvisionTenant)
		})
	})
	return r
}
