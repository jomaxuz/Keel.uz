package handlers

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"keel-control/internal/config"
	"keel-control/internal/httpx"
	"keel-control/internal/middleware"
	"keel-control/internal/models"
	"keel-control/internal/provision"
	"keel-control/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	Store *repository.Store
	Cfg   *config.Config
	// nil when this deployment cannot start containers — a laptop, or a server
	// where the socket was deliberately not mounted. Everything else still
	// works; tenants are recorded and started by hand.
	Docker *provision.Client
}

func New(s *repository.Store, cfg *config.Config) *Handler {
	h := &Handler{Store: s, Cfg: cfg}
	if cfg.DockerSocket != "" {
		h.Docker = provision.New(provision.Config{
			Socket:      cfg.DockerSocket,
			Image:       cfg.TenantImage,
			Network:     cfg.DockerNetwork,
			MongoURI:    cfg.TenantMongoHost,
			UploadsRoot: cfg.UploadsRoot,
			CommonEnv: map[string]string{
				"SMS_PROVIDER":            cfg.SMSProvider,
				"SMS_FROM":                cfg.SMSFrom,
				"ESKIZ_EMAIL":             cfg.EskizEmail,
				"ESKIZ_PASSWORD":          cfg.EskizPassword,
				"NEXT_PUBLIC_MAP_API_KEY": cfg.MapAPIKey,
			},
		})
	}
	return h
}

// ---- Auth ----

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var u models.User
	err := h.Store.Users.FindOne(r.Context(),
		bson.M{"username": strings.ToLower(strings.TrimSpace(req.Username))}).Decode(&u)
	// One message for both a wrong name and a wrong password: telling an
	// attacker which half they got right halves the work.
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		httpx.Error(w, http.StatusUnauthorized, "login yoki parol noto'g'ri")
		return
	}
	token, err := middleware.Sign(h.Cfg.JWTSecret, u.ID.Hex(), u.Username)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	now := time.Now()
	_, _ = h.Store.Users.UpdateByID(r.Context(), u.ID, bson.M{"$set": bson.M{"lastLoginAt": now}})
	httpx.JSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user":  map[string]string{"username": u.Username, "name": u.Name},
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	c := middleware.From(r.Context())
	if c == nil {
		httpx.Error(w, http.StatusUnauthorized, "avtorizatsiya kerak")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"username": c.Username})
}

// ---- Internal, for Caddy and the shared frontend ----

// Resolve answers "which tenant is this hostname". Called by the shared
// Next.js process on server-side render, so it is deliberately tiny and
// unauthenticated — it reveals nothing a visitor cannot see by loading the
// page.
func (h *Handler) Resolve(w http.ResponseWriter, r *http.Request) {
	t, err := h.byHost(r.Context(), r.URL.Query().Get("host"))
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "domen topilmadi")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"slug":   t.Slug,
		"status": t.Status,
		// The footer line, decided here rather than in the tenant's own
		// settings: it is what the customer pays to remove.
		"hideWatermark": t.HideWatermark,
	})
}

// TLSAsk is Caddy's on-demand TLS gate.
//
// Without it, anybody who points a domain at this IP makes us request a
// certificate for it, and Let's Encrypt's rate limit is spent on strangers.
// 200 means "ours"; anything else means Caddy does not ask.
func (h *Handler) TLSAsk(w http.ResponseWriter, r *http.Request) {
	if _, err := h.byHost(r.Context(), r.URL.Query().Get("domain")); err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) byHost(ctx context.Context, host string) (*models.Tenant, error) {
	host = strings.ToLower(strings.TrimSpace(host))
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	if host == "" {
		return nil, errNotFound
	}
	var t models.Tenant
	if err := h.Store.Tenants.FindOne(ctx, bson.M{"domains": host}).Decode(&t); err != nil {
		return nil, err
	}
	return &t, nil
}

var errNotFound = &notFound{}

type notFound struct{}

func (*notFound) Error() string { return "topilmadi" }

// slugRe is what may become a database name and a container name.
var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,30}[a-z0-9]$`)
