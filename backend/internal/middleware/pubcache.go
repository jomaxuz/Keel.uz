package middleware

// ---- The public reads that never reach Mongo twice ----
//
// **Measured, not guessed.** The load test of 2026-09-03 put the server at
// ~350 req/s and named the bottleneck: mongod at 139% CPU — 1.4 of four cores —
// while the twelve Go backends together spent 10–20%. The application code is
// not the cost; asking the database the same question over and over is.
//
// And it is the same question. Five of the nine requests in a visitor's flow
// (`/restaurant`, `/menu`, `/categories`, `/promotions`, `/payment-methods`)
// answer with data an owner changes a few times a week, and every one of them
// went to Mongo, for every guest, every time.
//
// ⚠️ **In the backend rather than at the edge, because of who the key belongs
// to.** One Keel deployment serves twelve restaurants, so an edge cache has to
// carry the hostname in its key or it serves one restaurant's menu on another's
// domain — the worst bug this system could have (the note at the top of
// caddy/pagecache.conf is the long version). A tenant's backend is its own
// container: there is no second restaurant it *could* answer for, so the key
// cannot be wrong in that direction at all.
//
// ⚠️ **Thirty seconds, and the whole design leans on that being short.** No
// invalidation is wired to the admin writes on purpose: a cache that has to be
// told about every future write is a cache that eventually is not told, and the
// symptom — an owner editing a price and not seeing it — reads exactly like the
// save having failed. A TTL expires whether or not anybody remembered it.

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// CacheHeader says whether this answer came from the cache.
//
// ⚠️ **The only way to tell this is working from outside**, and it is kept in
// production for exactly the reason pagecache.conf keeps its own: a cache that
// has quietly stopped caching looks identical to one that is working — the site
// is simply as slow as it was before, which is nobody's alarm. It reveals
// nothing: HIT or MISS is a fact about a page every visitor can already fetch.
const CacheHeader = "X-Cache"

// PublicCache holds the last answer to each public read.
type PublicCache struct {
	mu      sync.Mutex
	entries map[string]*pubEntry
	// ⚠️ **A cap, because the key contains a query string a stranger writes.**
	// `?brand=aaa`, `?brand=aab`, … would otherwise grow this map until the
	// container is killed. Past the cap nothing new is stored and requests
	// simply go to Mongo as they did before this file existed — the old
	// behaviour is the overflow behaviour.
	max int
}

type pubEntry struct {
	// Closed once the response below is filled in. A second request for the
	// same cold key waits on this rather than running the handler as well —
	// the request coalescing that keeps a dinner-time burst from multiplying
	// into work (same reason as `proxy_cache_lock` in pagecache.conf).
	ready  chan struct{}
	exp    time.Time
	status int
	header http.Header
	body   []byte
	// False when the response turned out not to be cacheable. Waiters then run
	// the handler themselves rather than being served nothing.
	ok bool
}

// NewPublicCache builds a cache holding at most max distinct answers.
func NewPublicCache(max int) *PublicCache {
	if max <= 0 {
		max = 512
	}
	return &PublicCache{entries: map[string]*pubEntry{}, max: max}
}

// For wraps a read-only public handler.
//
// The TTL is per route because the answers do not age alike: a menu may be a
// minute stale without anybody noticing, a promotions list is what the guest is
// about to try to use.
func (c *PublicCache) For(ttl time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cacheable(r) {
				next.ServeHTTP(w, r)
				return
			}
			key := pubKey(r)
			if e, leader := c.claim(key, ttl); e != nil {
				if leader {
					// The leader always writes what it produced, stored or
					// not — re-running the handler for the request that just
					// ran it would double every uncacheable answer.
					rec := c.fill(e, key, ttl, next, r, langOf(r))
					replay(w, rec, ttl, e.ok)
					return
				}
				select {
				case <-e.ready:
				case <-r.Context().Done():
					// The guest navigated away while the leader was still
					// working. Nothing to write to.
					return
				}
				if e.ok {
					writeEntry(w, e, ttl)
					return
				}
				// The answer was not cacheable — an error, a redirect,
				// something carrying a cookie. Produce it for this request.
			}
			// Not cached: answer the old way, but still tell the browser and any
			// CDN how long it may hold this. ⚠️ The header goes on through
			// `ccWriter` rather than up front, because it must not be attached
			// to a response that turns out to be an error or to carry a cookie.
			next.ServeHTTP(&ccWriter{ResponseWriter: w, ttl: ttl}, r)
		})
	}
}

// claim returns the entry to use, and whether this request has to produce it.
func (c *PublicCache) claim(key string, ttl time.Duration) (*pubEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	if e, found := c.entries[key]; found {
		// A entry still being produced has a zero expiry: waiting for it is the
		// point, so it must not be read as expired.
		if !e.ok || now.Before(e.exp) {
			return e, false
		}
	}
	if len(c.entries) >= c.max {
		for k, e := range c.entries {
			if e.ok && now.After(e.exp) {
				delete(c.entries, k)
			}
		}
		if len(c.entries) >= c.max {
			return nil, false
		}
	}
	e := &pubEntry{ready: make(chan struct{})}
	c.entries[key] = e
	return e, true
}

// fill runs the handler once and stores the result.
func (c *PublicCache) fill(
	e *pubEntry, key string, ttl time.Duration,
	next http.Handler, r *http.Request, lang string,
) *recorder {
	rec := &recorder{header: http.Header{}, status: http.StatusOK, lang: lang}
	// ⚠️ The channel is closed and the entry dropped even if the handler
	// panics: a leader that never publishes leaves every later request for that
	// key waiting on a channel nobody will close, which is a hung site rather
	// than a failed request. Recoverer above still turns the panic into a 500.
	defer func() {
		if !e.ok {
			c.mu.Lock()
			if c.entries[key] == e {
				delete(c.entries, key)
			}
			c.mu.Unlock()
		}
		close(e.ready)
	}()
	next.ServeHTTP(rec, r)
	if !storable(rec) {
		return rec
	}
	e.status, e.header, e.body = rec.status, rec.header.Clone(), rec.body
	e.exp = time.Now().Add(ttl)
	e.ok = true
	return rec
}

// replay sends the leader's own recorded response to its own client.
func replay(w http.ResponseWriter, rec *recorder, ttl time.Duration, cached bool) {
	for k, vs := range rec.header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	if cached {
		w.Header().Set(CacheHeader, "MISS")
		setCacheControl(w, ttl)
	}
	w.WriteHeader(rec.status)
	_, _ = w.Write(rec.body)
}

// ccWriter attaches the caching header to a response only once it is known to
// deserve one.
//
// ⚠️ **A `public` cache-control on the wrong response is worse than none.** The
// header is decided at `WriteHeader`, where the status and any `Set-Cookie` are
// finally known — set at the top of the handler instead, a 500 or a response
// carrying a session would be held by every browser and CDN in the chain for
// the whole window.
type ccWriter struct {
	http.ResponseWriter
	ttl   time.Duration
	wrote bool
}

func (w *ccWriter) WriteHeader(status int) {
	if !w.wrote {
		w.wrote = true
		if status == http.StatusOK && len(w.Header().Values("Set-Cookie")) == 0 {
			setCacheControl(w.ResponseWriter, w.ttl)
		}
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *ccWriter) Write(p []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

// Lang and Flush keep the wrapper transparent to everything that looks through
// it — the language the response is written in, and streaming.
func (w *ccWriter) Lang() string {
	if lw, ok := w.ResponseWriter.(interface{ Lang() string }); ok {
		return lw.Lang()
	}
	return ""
}

func (w *ccWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *ccWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func writeEntry(w http.ResponseWriter, e *pubEntry, ttl time.Duration) {
	for k, vs := range e.header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.Header().Set(CacheHeader, "HIT")
	setCacheControl(w, ttl)
	w.WriteHeader(e.status)
	_, _ = w.Write(e.body)
}

func setCacheControl(w http.ResponseWriter, ttl time.Duration) {
	secs := int(ttl.Seconds())
	if secs < 1 {
		secs = 1
	}
	// `public` so a CDN may hold it too, and `stale-while-revalidate` so the
	// refresh happens behind a guest who is already looking at the page.
	w.Header().Set("Cache-Control",
		"public, max-age="+itoa(secs)+", stale-while-revalidate="+itoa(secs*2))
}

// cacheable decides whether this *request* may be answered from the cache.
func cacheable(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	// ⚠️ **Anything with a token is answered fresh.** The panel, the till and
	// the staff apps read some of these same routes, and they are the callers
	// for whom thirty seconds of staleness reads as "my change did not save" —
	// the one symptom this cache must never produce. They are also a rounding
	// error in the traffic, so nothing is lost by excluding them.
	if r.Header.Get("Authorization") != "" {
		return false
	}
	q := r.URL.Query()
	// The settings page asks for the company document as stored, and it asks
	// again right after saving it.
	if q.Get("raw") == "1" {
		return false
	}
	// A design preview exists to show the change that was just made; see the
	// same rule, and the same reason, in caddy/pagecache.conf.
	if q.Get("preview") != "" {
		return false
	}
	return true
}

// storable decides whether this *response* may be kept.
func storable(rec *recorder) bool {
	if rec.status != http.StatusOK {
		return false
	}
	// ⚠️ Never store a response that sets a cookie — a cached `Set-Cookie`
	// hands one visitor's session to whoever asks next.
	if len(rec.header.Values("Set-Cookie")) > 0 {
		return false
	}
	// A megabyte is far above any of these answers (the largest measured was
	// 22 KB). It is here so one pathological document cannot fill the map.
	return len(rec.body) <= 1<<20
}

// pubKey is everything that can change the answer.
//
// ⚠️ **The language belongs in the key.** These responses carry sentences the
// server writes, and a cache keyed on the path alone would answer a Russian
// guest with whatever the previous Uzbek one was handed. The brand and the
// branch are already in the query string — they are parameters here, not
// cookies — which is why the key does not name them separately.
func pubKey(r *http.Request) string {
	var b strings.Builder
	b.WriteString(r.URL.Path)
	b.WriteByte('?')
	b.WriteString(r.URL.RawQuery)
	b.WriteByte('|')
	b.WriteString(langOf(r))
	return b.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// recorder captures a handler's response instead of sending it.
//
// ⚠️ **It answers `Lang()`, and that is not decoration.** The handler is given
// this writer in place of the real one, and the real one is the `langWriter`
// that `httpx.Error` and `httpx.LangOf` interrogate. A recorder without the
// method would type-assert to nothing and every sentence written through a
// cached route would silently be Uzbek — the exact bug the note at the top of
// lang.go describes, reintroduced from underneath.
type recorder struct {
	header http.Header
	status int
	body   []byte
	wrote  bool
	lang   string
}

func (r *recorder) Lang() string { return r.lang }

func (r *recorder) Header() http.Header { return r.header }

func (r *recorder) WriteHeader(status int) {
	if !r.wrote {
		r.status, r.wrote = status, true
	}
}

func (r *recorder) Write(p []byte) (int, error) {
	r.wrote = true
	r.body = append(r.body, p...)
	return len(p), nil
}
