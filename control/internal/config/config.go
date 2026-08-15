package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"keel-control/internal/models"

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
	// The platform's volume ladder, used by every tenant that has not
	// negotiated its own. "upTo:price" bands, comma separated; the last band
	// uses upTo 0 for "no limit". Empty switches tiers off and bills flat.
	//
	// Read as "whichever band is cheapest" — a band may be bought early at its
	// full order count, see models.PriceForOrders. The published ladder is set
	// against the competitor's, band for band: theirs prices the whole volume
	// too, so band rates are directly comparable and ours are 20% under.
	PriceTiers []models.PriceTier
	// The least a customer is billed for a period they actually used, in so'm.
	//
	// The other end of the ladder: tiers keep the largest customer's invoice
	// from inviting a negotiation, this keeps the smallest one from costing
	// more in support than it pays. **Defaults to 0 — off** on purpose: a floor
	// changes what real customers owe, and that must be somebody's decision,
	// never the side effect of deploying a release.
	MinMonthly int
	// What removing the Keel badge costs per month, in so'm. ⚠️ A setting rather than a
	// constant: it is a price, and every price in this platform is negotiable with the
	// customer in front of you. Zero switches the charge off entirely.
	WatermarkPrice int
	// How long a new tenant evaluates before the clock matters.
	TrialDays int

	// ---- Provisioning ----
	//
	// Empty DockerSocket or CaddyAdmin switches the matching half off: the
	// control plane still records tenants, it just does not start them. That is
	// the mode it runs in on a laptop, and the mode it must not crash in.
	DockerSocket  string
	TenantImage   string
	DockerNetwork string
	// What a tenant container should use to reach Mongo — a name on the shared
	// network, not this service's own URI.
	TenantMongoHost string
	UploadsRoot     string
	// Which filesystem the console reports free space for. Defaults to the
	// uploads root because that is the one that fills: it is on the host disk
	// and it only ever grows. A path the container cannot see reports its own
	// overlay instead, which is why it is named in the answer.
	DiskPath string
	// Where the nightly backup writes, mounted read-only so the console can
	// answer "when did this last work" from the manifest on disk.
	//
	// Read live, never stored: a flag saying "backups are on" is true from the
	// moment it is written and tells you nothing about last night. The failure
	// this catches is the one that matters — cron removed, disk full, mongo
	// container renamed — and every one of those leaves the flag untouched
	// while the copies quietly stop.
	BackupPath string
	// Move every stale tenant onto the current image shortly after this
	// process starts. On by default: a deploy recreates this container and
	// nothing else knows a deploy happened, so left to a human the rollout
	// simply never runs. Safe on a plain reboot — "stale" is decided by image
	// id, and after a reboot nothing is.
	RolloutOnBoot bool

	CaddyAdmin   string
	CaddyEmail   string
	FrontendHost string
	// This service, as the edge can reach it.
	ControlHost  string
	MainUpstream string

	// Passed through to every tenant container unchanged.
	SMSProvider   string
	SMSFrom       string
	EskizEmail    string
	EskizPassword string
	MapAPIKey     string
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
		DefaultPricePerOrder: atoi(get("PRICE_PER_ORDER", "800"), 800),
		PriceTiers:           parseTiers(get("PRICE_TIERS", "3000:800,15000:560,50000:400,0:300")),
		MinMonthly:           atoi(get("MIN_MONTHLY", "0"), 0),
		WatermarkPrice:       atoi(get("WATERMARK_PRICE", "3000000"), 3_000_000),
		TrialDays:            atoi(get("TRIAL_DAYS", "14"), 14),

		DockerSocket:    get("DOCKER_SOCKET", ""),
		TenantImage:     get("TENANT_IMAGE", "keel-tenant:latest"),
		DockerNetwork:   get("DOCKER_NETWORK", "keel"),
		TenantMongoHost: get("TENANT_MONGO_HOST", "mongodb://mongo:27017"),
		UploadsRoot:     get("UPLOADS_ROOT", "/srv/keel/tenants"),
		DiskPath:        get("DISK_PATH", "/"),
		BackupPath:      get("BACKUP_PATH", "/srv/keel/backups"),
		RolloutOnBoot:   get("ROLLOUT_ON_BOOT", "1") != "0",

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

// parseTiers reads "3000:800,15000:560,50000:400,0:300".
//
// A malformed entry is skipped rather than defaulted: a typo that silently
// became a price would bill real customers. A list that ends up empty falls
// back to the flat rate, which is the old behaviour and safe.
func parseTiers(raw string) []models.PriceTier {
	out := []models.PriceTier{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		upTo, price, ok := strings.Cut(part, ":")
		if !ok {
			log.Printf("config: PRICE_TIERS bandi tushunilmadi: %q", part)
			continue
		}
		u, err1 := strconv.Atoi(strings.TrimSpace(upTo))
		p, err2 := strconv.Atoi(strings.TrimSpace(price))
		if err1 != nil || err2 != nil || p < 0 || u < 0 {
			log.Printf("config: PRICE_TIERS bandi tushunilmadi: %q", part)
			continue
		}
		out = append(out, models.PriceTier{UpTo: u, Price: p})
	}
	return out
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
