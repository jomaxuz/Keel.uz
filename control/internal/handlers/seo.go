package handlers

// ---- Telling search engines a page exists ----
//
// ⚠️ **Google cannot be pushed to, and this file does not pretend otherwise.**
// The obvious feature request is a button that submits every page to Google
// Search Console. There is no public API for it: the Indexing API is documented
// for `JobPosting` and `BroadcastEvent` only, and the sitemap ping endpoint
// (`google.com/ping?sitemap=`) was withdrawn in 2023. What works for Google is
// the sitemap, submitted once in Search Console by hand, which Google then
// re-reads on its own — and `sitemap.xml` already lists every page in all three
// languages. A button that quietly did nothing would be worse than no button:
// somebody would press it and stop wondering why nothing was indexed.
//
// ⚠️ **IndexNow is real and covers the engine that matters most here.** Bing,
// Yandex, Seznam and Naver accept a pushed URL list and fetch within minutes.
// Yandex alone justifies the endpoint: it is a large share of search in
// Uzbekistan, and it is the one this product's customers use.
//
// ⚠️ **The URL list comes from the sitemap, not from a list in this file.**
// The sitemap is already the canonical answer to "what pages exist" — it is
// generated from the same article data the site renders, in all three
// languages. A second list here would be correct on the day it was written and
// wrong after the next article.

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"keel-control/internal/httpx"
)

// indexNowEndpoint accepts for every participating engine at once — submitting
// to each separately is explicitly discouraged by the protocol.
const indexNowEndpoint = "https://api.indexnow.org/indexnow"

// keyLocation is where the site serves the ownership key. Fixed rather than
// `/<key>.txt` so rotating the key is one environment variable.
const keyPath = "/indexnow.txt"

type sitemapDoc struct {
	URLs []struct {
		Loc   string `xml:"loc"`
		Links []struct {
			Href string `xml:"href,attr"`
		} `xml:"link"`
	} `xml:"url"`
}

// seoState is the one document remembering the last push.
type seoState struct {
	LastPingAt  time.Time `bson:"lastPingAt" json:"lastPingAt"`
	LastCount   int       `bson:"lastCount" json:"lastCount"`
	LastStatus  int       `bson:"lastStatus" json:"lastStatus"`
	LastMessage string    `bson:"lastMessage,omitempty" json:"lastMessage,omitempty"`
}

// SeoStatus answers what the screen needs to decide whether to press anything.
func (h *Handler) SeoStatus(w http.ResponseWriter, r *http.Request) {
	urls, err := h.sitemapURLs(r.Context())
	out := map[string]any{
		"origin":      h.origin(),
		"sitemap":     h.origin() + "/sitemap.xml",
		"keyLocation": h.origin() + keyPath,
		"hasKey":      h.Cfg.IndexNowKey != "",
		"urls":        len(urls),
	}
	if err != nil {
		// ⚠️ Reported rather than swallowed: "0 pages" and "could not read the
		// sitemap" look identical on a screen and mean opposite things.
		out["error"] = err.Error()
	}
	var st seoState
	if e := h.Store.DB.Collection("seo_state").
		FindOne(r.Context(), bson.M{"_id": "indexnow"}).Decode(&st); e == nil {
		out["last"] = st
	}
	out["llms"] = h.llmsStatus(r.Context())
	httpx.JSON(w, http.StatusOK, out)
}

// SeoPing pushes every page in the sitemap to the IndexNow engines.
func (h *Handler) SeoPing(w http.ResponseWriter, r *http.Request) {
	if h.Cfg.IndexNowKey == "" {
		httpx.Error(w, http.StatusBadRequest,
			"INDEXNOW_KEY sozlanmagan — kalit qo'yilmaguncha yuborish rad etiladi")
		return
	}
	urls, err := h.sitemapURLs(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "sitemap o'qilmadi: "+err.Error())
		return
	}
	if len(urls) == 0 {
		httpx.Error(w, http.StatusBadGateway, "sitemapda manzil topilmadi")
		return
	}

	// ⚠️ **The AI files ride along.** `/llms.txt` and `/llms-full.txt` are not
	// in the sitemap (they are not pages a person lands on), so without this
	// line the one push that says "look at everything" would skip exactly the
	// files the answer engines read. See llms.go.
	urls = append(urls, h.llmsURLs()...)

	host := h.indexNowHost()
	status, message, err := h.indexNowSubmit(r.Context(), urls)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "yuborilmadi: "+err.Error())
		return
	}
	st := seoState{
		LastPingAt:  time.Now(),
		LastCount:   len(urls),
		LastStatus:  status,
		LastMessage: message,
	}
	_, _ = h.Store.DB.Collection("seo_state").UpdateOne(r.Context(),
		bson.M{"_id": "indexnow"}, bson.M{"$set": st}, options.Update().SetUpsert(true))

	h.logConsole(r.Context(), h.actorOrNil(r), "seo.indexnow", host,
		fmt.Sprintf("%d ta manzil, javob %d", len(urls), status))

	// ⚠️ The engine's own status is passed through rather than translated into
	// ok/not-ok. 200 and 202 both mean accepted; 403 means the key file is not
	// readable; 422 means the URLs do not belong to the host. Each has a
	// different fix, and "yuborilmadi" tells nobody which.
	httpx.JSON(w, http.StatusOK, map[string]any{
		"urls":    len(urls),
		"status":  status,
		"message": st.LastMessage,
		"ok":      indexNowAccepted(status),
	})
}

// indexNowAccepted is the engine's own word for "taken": 200 and 202 both.
func indexNowAccepted(status int) bool {
	return status == http.StatusOK || status == http.StatusAccepted
}

// indexNowHost is the host IndexNow is told the URLs belong to.
func (h *Handler) indexNowHost() string {
	return strings.TrimPrefix(strings.TrimPrefix(h.origin(), "https://"), "http://")
}

// indexNowSubmit posts one URL list to the IndexNow engines and hands back
// their answer unchanged — see SeoPing for why the status is not flattened.
//
// ⚠️ **One function for the button and for the watcher.** The automatic ping
// (llms.go) and the manual one must send the same body with the same key; two
// copies of this request is how one of them ends up on the old key location.
func (h *Handler) indexNowSubmit(ctx context.Context, urls []string) (int, string, error) {
	body, _ := json.Marshal(map[string]any{
		"host":        h.indexNowHost(),
		"key":         h.Cfg.IndexNowKey,
		"keyLocation": h.origin() + keyPath,
		"urlList":     urls,
	})
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, indexNowEndpoint,
		strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer res.Body.Close()
	answer, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
	return res.StatusCode, strings.TrimSpace(string(answer)), nil
}

// sitemapURLs reads every address the site publishes, in every language.
//
// ⚠️ **The alternates are collected too.** A sitemap entry lists one canonical
// URL and its `hreflang` siblings as `<xhtml:link>`; taking only `<loc>` would
// push the Uzbek page and leave the Russian and English ones to be found by
// crawling — which is the whole thing being avoided.
func (h *Handler) sitemapURLs(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		h.origin()+"/sitemap.xml", nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sitemap %d", res.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	var doc sitemapDoc
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	out := make([]string, 0, len(doc.URLs)*3)
	add := func(u string) {
		u = strings.TrimSpace(u)
		// ⚠️ Only our own host. IndexNow refuses a whole submission when one
		// URL belongs elsewhere, so a stray absolute link in the sitemap would
		// take every other page down with it.
		if u == "" || seen[u] || !strings.HasPrefix(u, h.origin()) {
			return
		}
		seen[u] = true
		out = append(out, u)
	}
	for _, u := range doc.URLs {
		add(u.Loc)
		for _, l := range u.Links {
			add(l.Href)
		}
	}
	return out, nil
}

// origin is the public address of the marketing site.
//
// ⚠️ **http only for a local host, and only for that.** In production this is
// always https — IndexNow refuses a plain-http submission, and so it should.
// But without this branch the feature cannot be exercised at all before it is
// deployed, and a screen whose only test is production is a screen that ships
// broken. The condition is the hostname, not a flag: a flag can be set on the
// real server by accident.
func (h *Handler) origin() string {
	host := h.Cfg.BaseDomain
	if strings.HasPrefix(host, "localhost") || strings.HasPrefix(host, "127.0.0.1") {
		return "http://" + host
	}
	return "https://" + host
}
