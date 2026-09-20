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
	// The platform's own Claude key. ⚠️ Held here and never copied into a
	// tenant container: one secret in one place is one thing to rotate.
	AnthropicKey string
	// Google's key, and the model each engine runs.
	//
	// ⚠️ **Two engines because one can be unreachable for reasons that have
	// nothing to do with the code.** Anthropic billing needs a card that works
	// internationally, and the first thing this platform's own key answered was
	// "credit balance too low". A restaurant's morning briefing should not
	// depend on which payment rails were available to us that month.
	GeminiKey string
	// Which is tried first. "gemini" puts Google in front; anything else keeps
	// Claude there. ⚠️ Not a hard choice — whichever is second still answers
	// when the first cannot, so a key that stops working costs a few seconds
	// rather than a morning.
	AIProvider string
	// Empty means each engine's own default. Here so a model can be changed
	// without a release.
	AIModel string
	// ⚠️ **Empty means every free Gemini model, tried in order** — not one
	// default. The free tier meters each model separately, so a spent
	// allowance on the newest one leaves five more full allowances on the same
	// key, and walking them is the difference between a briefing and no
	// briefing. Naming models here (comma-separated) pins the platform to
	// exactly those, which is what a paid key wants.
	GeminiModel string
	// The first dashboard account, created on boot when no user exists.
	AdminUsername string
	AdminPassword string

	CORSOrigins []string
	// Subdomain every new tenant gets for free: <slug>.<BaseDomain>.
	BaseDomain string
	// The IndexNow ownership key. Empty means the push endpoint refuses rather
	// than sending a submission that will be rejected — see handlers/seo.go.
	IndexNowKey string
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
	DockerSocket string
	TenantImage  string
	// ---- Building restaurants' Android apps ----
	//
	// ⚠️ **Empty AppBuildImage switches the whole feature off**, the way an
	// empty DockerSocket switches provisioning off: on a laptop there is no
	// Android toolchain and the console must still open. The button says so
	// rather than failing.
	AppBuildImage string
	// The repository checkout the build script and the app sources live in,
	// bound into the build container as `/opt/keel`.
	AppBuildRoot string
	// A named Docker volume for Gradle's cache. ⚠️ Without it every build
	// downloads the Gradle distribution and every dependency again — ten
	// minutes and several hundred megabytes, each time.
	AppBuildCache string
	// Keel's own keys, shared by every restaurant's app. ⚠️ The Firebase *app*
	// id is not here: it is per applicationId, and the pipeline gets it when it
	// registers one.
	AppMapsKey         string
	AppFirebaseProject string
	AppFirebaseAPIKey  string
	AppFirebaseSender  string
	DockerNetwork      string
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
	// Where the Windows till's release manifest and installers sit, so the
	// console can read what the tills are being offered. The same directory
	// handlers.TillRelease serves from — read from one place so a panel
	// reporting "0.2.0" and a till downloading 0.1.9 cannot both be right.
	TillReleaseDir string

	// ---- Raising the platform's version from the console ----
	//
	// ⚠️ **Actions, not contents.** The token this holds needs one permission
	// on one repository: Actions, read and write. That is enough to start a
	// workflow that is already on `main`, and not enough to put anything new
	// there — so a console somebody gets into can re-run a deployment, which is
	// a nuisance, rather than deploy code of their own, which is the platform.
	//
	// Empty switches the buttons off, the same way an absent DOCKER_SOCKET
	// switches provisioning off: the panel still reports every version it can
	// read, and says plainly that raising it is not wired up here. A laptop
	// must not be one misconfigured field away from releasing.
	// ---- The Meta app every restaurant connects through ----
	//
	// ⚠️ **One app for the platform, and its secret never leaves this
	// process.** The same rule the AI key follows: N restaurant containers
	// would be N copies of a credential to rotate, and a tenant container is
	// the customer's side of the wire. The tenant sends us the code Meta gave
	// it and gets back a token for its own ad account — see adsmeta.go.
	MetaAppID     string
	MetaAppSecret string
	// The Login for Business configuration id. ⚠️ It replaces `scope`
	// entirely: which permissions and which asset types are asked for is
	// decided in the Meta app dashboard, not in our query string.
	MetaConfigID string

	GitHubToken string
	// "owner/repo".
	GitHubRepo string
	// The workflow file that performs a release. A name rather than an id
	// because the id changes when the file is recreated, and the failure that
	// causes is a 404 from a button that used to work.
	ReleaseWorkflow string
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
	// The Firebase service account every tenant sends native notifications as.
	//
	// ⚠️ **One project for the whole platform, not one per restaurant.** The
	// phone applications are ours — four package names in one Firebase project —
	// so the credential that delivers to them is ours too. A restaurant has no
	// Firebase account and should never be asked for one.
	//
	// ⚠️ **Passed as the JSON itself.** `push.Configure` takes either the
	// document or a path, and a path would have to exist inside every tenant
	// container — which means a mount per tenant, created at provision time, for
	// a value that is identical everywhere. The cost is that the key is visible
	// in `docker inspect` on the host; the host is where the key already lives,
	// and where Eskiz's password already is.
	FCMCredentials string
}

func Load() *Config {
	_ = godotenv.Load()
	uri := get("MONGO_URI", "mongodb://localhost:27017")
	return &Config{
		Port:           get("PORT", "9000"),
		MongoURI:       uri,
		MongoDB:        get("MONGO_DB", "keel_control"),
		TenantMongoURI: get("TENANT_MONGO_URI", uri),
		JWTSecret:      get("JWT_SECRET", "change-me"),
		AnthropicKey:   get("ANTHROPIC_API_KEY", ""),
		GeminiKey:      get("GEMINI_API_KEY", ""),
		AIProvider:     strings.ToLower(strings.TrimSpace(get("AI_PROVIDER", ""))),
		AIModel:        get("AI_MODEL", ""),
		GeminiModel:    get("GEMINI_MODEL", ""),
		AdminUsername:  get("ADMIN_USERNAME", "admin"),
		AdminPassword:  get("ADMIN_PASSWORD", "admin123"),
		CORSOrigins:    splitCSV(get("CORS_ORIGINS", "http://localhost:3100")),
		BaseDomain:     get("BASE_DOMAIN", "keel.uz"),
		// ⚠️ No default. IndexNow refuses a submission whose key file does not
		// match, and a hardcoded fallback would be a key every install shares —
		// which is a key anybody can read off this repository and use to push
		// URLs on somebody else's domain.
		IndexNowKey:          get("INDEXNOW_KEY", ""),
		DefaultPricePerOrder: atoi(get("PRICE_PER_ORDER", "800"), 800),
		PriceTiers:           parseTiers(get("PRICE_TIERS", "3000:800,15000:560,50000:400,0:300")),
		MinMonthly:           atoi(get("MIN_MONTHLY", "0"), 0),
		WatermarkPrice:       atoi(get("WATERMARK_PRICE", "3000000"), 3_000_000),
		TrialDays:            atoi(get("TRIAL_DAYS", "14"), 14),

		DockerSocket:  get("DOCKER_SOCKET", ""),
		AppBuildImage: get("APP_BUILD_IMAGE", ""),
		AppBuildRoot:  get("APP_BUILD_ROOT", "/opt/keel"),
		AppBuildCache: get("APP_BUILD_CACHE", "keel-gradle-cache"),

		AppMapsKey:         get("APP_MAPS_KEY", ""),
		AppFirebaseProject: get("APP_FIREBASE_PROJECT", ""),
		AppFirebaseAPIKey:  get("APP_FIREBASE_API_KEY", ""),
		AppFirebaseSender:  get("APP_FIREBASE_SENDER", ""),
		TenantImage:        get("TENANT_IMAGE", "keel-tenant:latest"),
		DockerNetwork:      get("DOCKER_NETWORK", "keel"),
		TenantMongoHost:    get("TENANT_MONGO_HOST", "mongodb://mongo:27017"),
		UploadsRoot:        get("UPLOADS_ROOT", "/srv/keel/tenants"),
		DiskPath:           get("DISK_PATH", "/"),
		BackupPath:         get("BACKUP_PATH", "/srv/keel/backups"),
		RolloutOnBoot:      get("ROLLOUT_ON_BOOT", "1") != "0",
		TillReleaseDir:     get("TILL_RELEASE_DIR", ""),

		MetaAppID:     get("META_APP_ID", ""),
		MetaAppSecret: get("META_APP_SECRET", ""),
		MetaConfigID:  get("META_CONFIG_ID", ""),

		GitHubToken:     get("GITHUB_TOKEN", ""),
		GitHubRepo:      get("GITHUB_REPO", ""),
		ReleaseWorkflow: get("RELEASE_WORKFLOW", "release.yml"),

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
		// ⚠️ Accepts a path too, so a deployment that would rather mount the
		// file than put it in the environment can — `push.Configure` reads
		// either. The path then has to exist inside the tenant containers.
		FCMCredentials: fileOrValue(get("FCM_CREDENTIALS", "")),
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

// fileOrValue returns the contents of `v` when it names a readable file, and
// `v` itself otherwise.
//
// ⚠️ **Resolved here rather than passed through, because what is downstream is a
// container.** A path is the natural way to hand a service-account key to a
// process on a host — but this value is copied into every tenant's environment,
// and a path that resolves on the host resolves to nothing inside them. Reading
// it once, here, means an operator may write either and both work.
//
// ⚠️ A missing file is not an error and not a guess: the value is passed through
// unchanged, and `push.Configure` in the tenant reports what it could not parse.
// Refusing to start the control plane over a notification credential would take
// every restaurant's website down with it.
func fileOrValue(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || strings.HasPrefix(v, "{") {
		return v
	}
	b, err := os.ReadFile(v)
	if err != nil {
		return v
	}
	return string(b)
}
