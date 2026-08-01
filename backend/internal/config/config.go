package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	Port          string
	MongoURI      string
	MongoDB       string
	JWTSecret     string
	UploadDir     string
	PublicBaseURL string
	CORSOrigins   []string
	AdminUsername string
	AdminPassword string

	// SMS login (phone + one-time code). Default provider is "demo": nothing is
	// actually sent and the code is returned by the API.
	SMSProvider        string
	SMSFrom            string
	EskizEmail         string
	EskizPassword      string
	EskizBaseURL       string
	PlayMobileURL      string
	PlayMobileLogin    string
	PlayMobilePassword string
}

// Load reads configuration from the environment (and .env if present).
func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:          get("PORT", "8080"),
		MongoURI:      get("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:       get("MONGO_DB", "restaurant"),
		JWTSecret:     get("JWT_SECRET", "change-me"),
		UploadDir:     get("UPLOAD_DIR", "./uploads"),
		PublicBaseURL: get("PUBLIC_BASE_URL", "http://localhost:8080"),
		CORSOrigins:   splitCSV(get("CORS_ORIGINS", "http://localhost:3000")),
		AdminUsername: get("ADMIN_USERNAME", "admin"),
		AdminPassword: get("ADMIN_PASSWORD", "admin123"),

		SMSProvider:        get("SMS_PROVIDER", "demo"),
		SMSFrom:            get("SMS_FROM", "4546"),
		EskizEmail:         get("ESKIZ_EMAIL", ""),
		EskizPassword:      get("ESKIZ_PASSWORD", ""),
		EskizBaseURL:       get("ESKIZ_BASE_URL", "https://notify.eskiz.uz"),
		PlayMobileURL:      get("PLAYMOBILE_URL", ""),
		PlayMobileLogin:    get("PLAYMOBILE_LOGIN", ""),
		PlayMobilePassword: get("PLAYMOBILE_PASSWORD", ""),
	}
}

func get(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
