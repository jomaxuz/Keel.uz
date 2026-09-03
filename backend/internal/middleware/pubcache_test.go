package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// The cache exists to keep Mongo out of the repeated question. If it stops
// doing that nothing breaks visibly — the site is simply as slow as it was —
// so the count is asserted rather than the body.
func TestSecondRequestNeverReachesTheHandler(t *testing.T) {
	var hits int32
	h := wrap(&hits, time.Minute, "ok")

	for i := 0; i < 3; i++ {
		res := do(h, http.MethodGet, "/menu", nil)
		if body := res.Body.String(); body != "ok" {
			t.Fatalf("request %d answered %q", i, body)
		}
	}
	if hits != 1 {
		t.Fatalf("handler ran %d times, want 1", hits)
	}
}

// A cached answer has to say how long it may be held; that header is what takes
// the repeat visits off the network entirely.
func TestCachedAnswerCarriesCacheControl(t *testing.T) {
	var hits int32
	h := wrap(&hits, 30*time.Second, "ok")

	for _, label := range []string{"miss", "hit"} {
		res := do(h, http.MethodGet, "/menu", nil)
		cc := res.Header().Get("Cache-Control")
		if !strings.Contains(cc, "max-age=30") || !strings.HasPrefix(cc, "public") {
			t.Fatalf("%s: cache-control is %q", label, cc)
		}
	}
}

// ⚠️ **A cache that has quietly stopped caching looks exactly like one that
// works** — the site is only as slow as it used to be, and nobody raises that.
// This header is what the next load test reads to tell the two apart, so it is
// asserted rather than assumed.
func TestTheAnswerSaysWhereItCameFrom(t *testing.T) {
	var hits int32
	h := wrap(&hits, time.Minute, "ok")

	if got := do(h, http.MethodGet, "/menu", nil).Header().Get(CacheHeader); got != "MISS" {
		t.Fatalf("first request reported %q, want MISS", got)
	}
	if got := do(h, http.MethodGet, "/menu", nil).Header().Get(CacheHeader); got != "HIT" {
		t.Fatalf("second request reported %q, want HIT", got)
	}
	// An answer that was never eligible must claim neither.
	res := do(h, http.MethodGet, "/menu", map[string]string{"Authorization": "Bearer x"})
	if got := res.Header().Get(CacheHeader); got != "" {
		t.Fatalf("an uncached answer reported %q", got)
	}
}

// ⚠️ The failure this pins down is silent and wrong-looking rather than broken:
// a Russian guest served the sentence the previous Uzbek one was handed.
func TestLanguageIsPartOfTheKey(t *testing.T) {
	var hits int32
	h := wrap(&hits, time.Minute, "")

	uz := do(h, http.MethodGet, "/menu", map[string]string{"Accept-Language": "uz"})
	ru := do(h, http.MethodGet, "/menu", map[string]string{"Accept-Language": "ru-RU,ru"})
	if uz.Body.String() == ru.Body.String() {
		t.Fatalf("both languages got %q", uz.Body.String())
	}
	if hits != 2 {
		t.Fatalf("handler ran %d times, want one per language", hits)
	}
}

// ⚠️ The panel and the till read some of these same routes, and they are the
// only callers for whom thirty seconds of staleness reads as "my save failed".
func TestARequestWithATokenIsAnsweredFresh(t *testing.T) {
	var hits int32
	h := wrap(&hits, time.Minute, "ok")

	auth := map[string]string{"Authorization": "Bearer x"}
	do(h, http.MethodGet, "/restaurant", auth)
	do(h, http.MethodGet, "/restaurant", auth)
	if hits != 2 {
		t.Fatalf("handler ran %d times; an authenticated read must not be cached", hits)
	}
}

// The settings page edits the company document and reloads it the moment it has
// saved. `?raw=1` is how it asks, and it must never be answered from a copy.
func TestRawIsNeverCached(t *testing.T) {
	var hits int32
	h := wrap(&hits, time.Minute, "ok")

	do(h, http.MethodGet, "/restaurant?raw=1", nil)
	do(h, http.MethodGet, "/restaurant?raw=1", nil)
	if hits != 2 {
		t.Fatalf("handler ran %d times; ?raw=1 must not be cached", hits)
	}
}

// ⚠️ A stored `Set-Cookie` hands one visitor's session to whoever asks next.
func TestAResponseThatSetsACookieIsNotStored(t *testing.T) {
	var hits int32
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		http.SetCookie(w, &http.Cookie{Name: "sid", Value: "secret"})
		_, _ = w.Write([]byte("ok"))
	})
	h := NewPublicCache(8).For(time.Minute)(inner)

	do(h, http.MethodGet, "/menu", nil)
	do(h, http.MethodGet, "/menu", nil)
	if hits != 2 {
		t.Fatalf("handler ran %d times; a response with Set-Cookie must not be stored", hits)
	}
}

// Errors are not answers. Caching a 500 would keep a momentary Mongo hiccup on
// the site for the rest of the window.
func TestOnlyOkIsStored(t *testing.T) {
	var hits int32
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("nope"))
	})
	h := NewPublicCache(8).For(time.Minute)(inner)

	for i := 0; i < 2; i++ {
		if res := do(h, http.MethodGet, "/menu", nil); res.Code != 500 {
			t.Fatalf("status %d", res.Code)
		}
	}
	if hits != 2 {
		t.Fatalf("handler ran %d times; a 500 must not be cached", hits)
	}
}

// ⚠️ **This is the part that isolates a burst**, and it is the reason the entry
// is created before the handler runs rather than after. A hundred simultaneous
// requests for a cold menu must become one query and ninety-nine waiters, not a
// hundred queries fighting for four cores.
func TestASimultaneousBurstIsOneQuery(t *testing.T) {
	var hits int32
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		time.Sleep(20 * time.Millisecond)
		_, _ = w.Write([]byte("ok"))
	})
	h := NewPublicCache(8).For(time.Minute)(inner)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if body := do(h, http.MethodGet, "/menu", nil).Body.String(); body != "ok" {
				t.Errorf("answered %q", body)
			}
		}()
	}
	wg.Wait()
	if hits != 1 {
		t.Fatalf("a cold burst ran the handler %d times, want 1", hits)
	}
}

// ⚠️ The key holds a query string a stranger writes. Past the cap the cache
// stops storing and the server answers exactly as it did before this existed —
// it must not grow until the container is killed.
func TestTheMapIsBounded(t *testing.T) {
	c := NewPublicCache(4)
	h := c.For(time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	for i := 0; i < 200; i++ {
		res := do(h, http.MethodGet, "/menu?brand="+itoa(i), nil)
		if res.Body.String() != "ok" {
			t.Fatalf("request %d answered %q", i, res.Body.String())
		}
	}
	c.mu.Lock()
	n := len(c.entries)
	c.mu.Unlock()
	if n > 4 {
		t.Fatalf("cache holds %d entries, cap is 4", n)
	}
}

// ⚠️ The handler is handed the recorder in place of the real writer, and the
// real one is what `httpx.LangOf` interrogates. Lose the method and every
// sentence on a cached route is silently Uzbek.
func TestTheRecorderStillKnowsTheLanguage(t *testing.T) {
	var got string
	h := NewPublicCache(8).For(time.Minute)(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if lw, ok := w.(interface{ Lang() string }); ok {
				got = lw.Lang()
			}
			_, _ = w.Write([]byte("ok"))
		}))

	do(h, http.MethodGet, "/menu", map[string]string{"Accept-Language": "ru"})
	if got != "ru" {
		t.Fatalf("handler saw language %q, want ru", got)
	}
}

// A stale entry is replaced rather than served for ever.
func TestTheEntryExpires(t *testing.T) {
	var hits int32
	h := wrap(&hits, 20*time.Millisecond, "ok")

	do(h, http.MethodGet, "/menu", nil)
	time.Sleep(40 * time.Millisecond)
	do(h, http.MethodGet, "/menu", nil)
	if hits != 2 {
		t.Fatalf("handler ran %d times; the entry should have expired", hits)
	}
}

// ---- helpers ----

// wrap builds a cached handler that counts its calls and, when body is empty,
// answers with the language it was asked in — which is how the key tests tell
// two answers apart.
func wrap(hits *int32, ttl time.Duration, body string) http.Handler {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		out := body
		if out == "" {
			out = langOf(r)
		}
		_, _ = w.Write([]byte(out))
	})
	return NewPublicCache(64).For(ttl)(inner)
}

func do(h http.Handler, method, target string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}
