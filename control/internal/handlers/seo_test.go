package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"keel-control/internal/config"
)

const sampleSitemap = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"
        xmlns:xhtml="http://www.w3.org/1999/xhtml">
  <url>
    <loc>%ORIGIN%</loc>
    <xhtml:link rel="alternate" hreflang="uz" href="%ORIGIN%"/>
    <xhtml:link rel="alternate" hreflang="ru" href="%ORIGIN%/ru"/>
    <xhtml:link rel="alternate" hreflang="en" href="%ORIGIN%/en"/>
  </url>
  <url>
    <loc>%ORIGIN%/help/tech-cards</loc>
    <xhtml:link rel="alternate" hreflang="uz" href="%ORIGIN%/help/tech-cards"/>
    <xhtml:link rel="alternate" hreflang="ru" href="%ORIGIN%/ru/help/tech-cards"/>
    <xhtml:link rel="alternate" hreflang="en" href="%ORIGIN%/en/help/tech-cards"/>
  </url>
  <url>
    <loc>https://somebody-else.example/page</loc>
  </url>
</urlset>`

func seoHandler(t *testing.T, body string) (*Handler, func()) {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sitemap.xml" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(strings.ReplaceAll(body, "%ORIGIN%", "http://"+srv.Listener.Addr().String())))
	}))
	host := srv.Listener.Addr().String()
	return &Handler{Cfg: &config.Config{BaseDomain: host}}, srv.Close
}

// ⚠️ **The alternates are the point.** A sitemap entry names one canonical URL
// and lists its `hreflang` siblings separately; reading only `<loc>` would push
// the Uzbek page and leave the Russian and English ones to be discovered by
// crawling, which is the exact wait this endpoint exists to skip. Two thirds of
// the pages would silently never be submitted.
func TestSitemapReadingPicksUpEveryLanguage(t *testing.T) {
	h, done := seoHandler(t, sampleSitemap)
	defer done()

	urls, err := h.sitemapURLs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Landing ×3 and the article ×3. The `<loc>` of each repeats its own `uz`
	// alternate, so the deduplication has to hold as well.
	if len(urls) != 6 {
		t.Fatalf("6 ta manzil kutilgan, %d keldi: %v", len(urls), urls)
	}
	for _, want := range []string{"/ru", "/en", "/ru/help/tech-cards", "/en/help/tech-cards"} {
		found := false
		for _, u := range urls {
			if strings.HasSuffix(u, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("%q topilmadi: %v", want, urls)
		}
	}
}

// ⚠️ IndexNow refuses an entire submission when one URL belongs to another
// host, so a single stray absolute link in the sitemap would take every other
// page down with it — and the failure would be one 422 for the whole run.
func TestForeignURLsAreDropped(t *testing.T) {
	h, done := seoHandler(t, sampleSitemap)
	defer done()

	urls, _ := h.sitemapURLs(context.Background())
	for _, u := range urls {
		if strings.Contains(u, "somebody-else.example") {
			t.Fatalf("begona manzil ro'yxatga tushdi: %s", u)
		}
	}
}

// ⚠️ "Could not read the sitemap" and "the sitemap is empty" look identical on
// a screen and mean opposite things — one is a broken deploy, the other is a
// broken generator.
func TestAMissingSitemapIsAnErrorNotAnEmptyList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	h := &Handler{Cfg: &config.Config{BaseDomain: srv.Listener.Addr().String()}}

	if _, err := h.sitemapURLs(context.Background()); err == nil {
		t.Fatal("404 bo'lgan sitemap xato bermadi")
	}
}

// ⚠️ https everywhere except a local host, and the condition is the hostname
// rather than a flag: a flag can be set on the real server by accident, and
// IndexNow refuses plain http — as it should.
func TestOriginIsHttpsExceptLocally(t *testing.T) {
	if got := (&Handler{Cfg: &config.Config{BaseDomain: "keel.uz"}}).origin(); got != "https://keel.uz" {
		t.Fatalf("prod uchun: %s", got)
	}
	if got := (&Handler{Cfg: &config.Config{BaseDomain: "localhost:3100"}}).origin(); got != "http://localhost:3100" {
		t.Fatalf("lokal uchun: %s", got)
	}
}
