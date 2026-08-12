package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// A small in-memory rate limiter for the handful of endpoints that are
// expensive to answer and reachable without a token: signing in, and asking for
// an SMS code.
//
// ⚠️ **Why in-memory, and why that is enough here.** One tenant is one
// container (see CLAUDE.md — the Go backend is per-restaurant; only the
// frontend and Mongo are shared), so there is exactly one process to keep the
// counter in, and a limit that resets if that process restarts is no weakness:
// a restart is not something an attacker can trigger by knocking on the login.
// A shared store (Redis) would be a new dependency, a new failure mode and a new
// thing to secure, bought to defend a door that a map already bolts.
//
// It defends two distinct harms, which is why it lives in front of these routes
// rather than inside them:
//
//   - **Money.** The SMS request endpoint spends a real message per call. Its
//     per-number cooldown stops one phone being flooded, but nothing stopped a
//     script walking a thousand numbers and billing the restaurant for a
//     thousand texts. The gate is per IP, so the walk is what it catches.
//   - **CPU.** A login verifies a bcrypt hash, which is ~100 ms of one core by
//     design — and a tenant container is capped at one core (CpuShares). A few
//     parallel guesses per second would spend the whole backend on refusing
//     them. Turning the guesses away before bcrypt runs is the point.
//
// Not a replacement for the per-account protections underneath (the SMS
// cooldown, the code-attempt cap): those defend one victim, this defends the
// server. Both are wanted.

// rateLimiter is a fixed-window counter keyed by client IP.
type rateLimiter struct {
	mu     sync.Mutex
	hits   map[string]*window
	limit  int
	per    time.Duration
	lastGC time.Time
}

type window struct {
	count int
	reset time.Time
}

// NewRateLimit builds middleware allowing `limit` requests per `per` from one
// IP. Over that, it answers 429 without ever reaching the handler.
func NewRateLimit(limit int, per time.Duration) func(http.Handler) http.Handler {
	rl := &rateLimiter{
		hits:   make(map[string]*window),
		limit:  limit,
		per:    per,
		lastGC: time.Now(),
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rl.allow(clientIP(r)) {
				next.ServeHTTP(w, r)
				return
			}
			// Plain text and a Retry-After, so a real user who hit the wall
			// (a shared office NAT, a burst of retries) is told to wait rather
			// than left guessing.
			w.Header().Set("Retry-After", "60")
			http.Error(w, "juda ko'p urinish — birozdan keyin qayta urining",
				http.StatusTooManyRequests)
		})
	}
}

func (rl *rateLimiter) allow(key string) bool {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Opportunistic sweep of expired windows, so the map cannot grow without
	// bound from one-off IPs. Cheap, and only every `per`.
	if now.Sub(rl.lastGC) > rl.per {
		for k, wnd := range rl.hits {
			if now.After(wnd.reset) {
				delete(rl.hits, k)
			}
		}
		rl.lastGC = now
	}

	wnd, ok := rl.hits[key]
	if !ok || now.After(wnd.reset) {
		rl.hits[key] = &window{count: 1, reset: now.Add(rl.per)}
		return true
	}
	if wnd.count >= rl.limit {
		return false
	}
	wnd.count++
	return true
}

// clientIP is the remote address stripped of its port. RealIP has already run
// (it trusts the proxy's X-Forwarded-For / X-Real-IP), so in production this is
// the actual client; direct hits fall back to the socket address.
func clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
