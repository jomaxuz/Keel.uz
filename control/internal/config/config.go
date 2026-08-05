package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config is the control plane's runtime settings.
type Config struct {
	Port string
	// The control plane's own database — never a tenant's.
	MongoURI string
	MongoDB  string
	// Tenant databases live on the same server in the first deployment, so
	// this defaults to MongoURI. It is separate because the day Mongo moves to
	// its own machine, only this changes.
	TenantMongoURI string

	JWTSecret string
	// The first dashboard account, created on boot when no user exists.
	AdminUsername string
	AdminPassword string

	CORSOrigins []string
	// Subdomain every new tenant gets for free: <slug>.<BaseDomain>.
	BaseDomain string
	// So'm per order, for tenants created without an explicit price.
	DefaultPricePerOrder int
	// How long a new tenant evaluates before the clock matters.
	TrialDays int

	// ---- Provisioning ----
	//
	// Empty DockerSocket or CaddyAdmin switches the matching half off: the
	// control plane still records tenants, it just does not start them. That is
	// the mode it runs in on a laptop, and the mode it must not crash in.
	DockerSocket string
	TenantImage  string
	DockerNetwork string
	// What a tenant container should use to reach Mongo — a name on the shared
	// network, not this service's own URI.
	TenantMongoHost string
	UploadsRoot     string

	CaddyAdmin   string
	CaddyEmail   string
	FrontendHost string
	// This service, as the edge can reach it.
	ControlHost  string
	MainUpstream string

	// Passed through to every tenant container unchanged.
	SMSProvider     string
	SMSFrom         string
	EskizEmail      string
	EskizPassword   string
	MapAPIKey       string
}

func Load() *Config {
	_ = godotenv.Load()
	uri := get("MONGO_URI", "mongodb://localhost:27017")
	return &Config{
		Port:                 get("PORT", "9000"),
		MongoURI:             uri,
		MongoDB:              get("MONGO_DB", "keel_control"),
		TenantMongoURI:       get("TENANT_MONGO_URI", uri),
		JWTSecret:            get("JWT_SECRET", "change-me"),
		AdminUsername:        get("ADMIN_USERNAME", "admin"),
		AdminPassword:        get("ADMIN_PASSWORD", "admin123"),
		CORSOrigins:          splitCSV(get("CORS_ORIGINS", "http://localhost:3100")),
		BaseDomain:           get("BASE_DOMAIN", "keel.uz"),
		DefaultPricePerOrder: atoi(get("PRICE_PER_ORDER", "1000"), 1000),
		TrialDays:            atoi(get("TRIAL_DAYS", "14"), 14),

		DockerSocket:    get("DOCKER_SOCKET", ""),
		TenantImage:     get("TENANT_IMAGE", "keel-tenant:latest"),
		DockerNetwork:   get("DOCKER_NETWORK", "keel"),
		TenantMongoHost: get("TENANT_MONGO_HOST", "mongodb://mongo:27017"),
		UploadsRoot:     get("UPLOADS_ROOT", "/srv/keel/tenants"),

		CaddyAdmin:   get("CADDY_ADMIN", ""),
		CaddyEmail:   get("CADDY_EMAIL", ""),
		FrontendHost: get("FRONTEND_HOST", "keel-frontend:3000"),
		ControlHost:  get("CONTROL_HOST", "keel-control:9000"),
		MainUpstream: get("MAIN_UPSTREAM", "keel-site:3100"),

		SMSProvider:   get("SMS_PROVIDER", "demo"),
		SMSFrom:       get("SMS_FROM", "4546"),
		EskizEmail:    get("ESKIZ_EMAIL", ""),
		EskizPassword: get("ESKIZ_PASSWORD", ""),
		MapAPIKey:     get("MAP_API_KEY", ""),
	}
}

func get(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func atoi(s string, def int) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return def
		}
		n = n*10 + int(r-'0')
	}
	if n == 0 {
		return def
	}
	return n
}

func splitCSV(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
