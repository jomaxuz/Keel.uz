package handlers

// ---- keel.uz for AI assistants: llms.txt, llms-full.txt, and telling ----
//
// The site serves `/llms.txt` (an index) and `/llms-full.txt` (every page's
// words) in three languages, generated per request from the same sources the
// pages render (keel-site/src/lib/llms.ts). So the files themselves are never
// stale. What this file adds is the other half: **noticing that they changed,
// and saying so to the engines that listen.**
//
// ⚠️ **There is no "ping" for language models, and the screen says so.**
// ChatGPT, Claude and Perplexity crawl on their own schedule; none of them
// takes a pushed URL. What does exist is IndexNow — and the engines behind it
// are the ones AI answers are built on here: Bing feeds ChatGPT search and
// Copilot, Yandex feeds Alice. So "ping" means: submit the changed pages and
// the two files to IndexNow, the same way the SEO screen's button does.
//
// ⚠️ **Change is measured per page, not per file.** The full file is split on
// its `URL:` lines (the marker lib/llms.ts writes under every heading) and each
// page is hashed on its own. A new blog post pings that post and the two files;
// it does not re-submit ninety help articles that did not move — engines
// throttle hosts that shout, and the next real change would wait behind it.
//
// ⚠️ **The first look is a baseline, not a change.** With nothing stored, every
// page would count as "new" and the first run after a deploy would push the
// whole site. The button on the screen is there for that, deliberately.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"keel-control/internal/httpx"
)

// llmsLangs are the languages the site serves the files in, with the path
// prefix each lives under (Uzbek is the unprefixed base).
var llmsLangs = []struct{ Lang, Prefix string }{
	{"uz", ""}, {"ru", "/ru"}, {"en", "/en"},
}

// llmsURLMarker is the line lib/llms.ts writes under every page heading.
const llmsURLMarker = "URL: "

// llmsPage is one page's fingerprint. ⚠️ A list, not a map keyed by URL: Mongo
// reads a dot in a `$set` key as a path, and every URL here has one.
type llmsPage struct {
	URL  string `bson:"url" json:"url"`
	Hash string `bson:"hash" json:"-"`
}

type llmsFile struct {
	Lang  string `bson:"lang" json:"lang"`
	Index string `bson:"index" json:"index"`
	Full  string `bson:"full" json:"full"`
	Bytes int    `bson:"bytes" json:"bytes"`
	Pages int    `bson:"pages" json:"pages"`
}

type llmsPing struct {
	At      time.Time `bson:"at" json:"at"`
	Count   int       `bson:"count" json:"count"`
	Status  int       `bson:"status" json:"status"`
	Message string    `bson:"message,omitempty" json:"message,omitempty"`
	// Whether the watcher sent it (true) or somebody pressed the button.
	Auto bool `bson:"auto" json:"auto"`
}

// llmsState is the one document the watcher keeps, `seo_state/llms`.
type llmsState struct {
	Pages     []llmsPage `bson:"pages" json:"-"`
	Files     []llmsFile `bson:"files" json:"files"`
	CheckedAt time.Time  `bson:"checkedAt" json:"checkedAt"`
	// When the text last actually changed — not when it was last looked at.
	// ⚠️ Two different facts: a watcher that ran a minute ago on a file
	// unchanged for a month must not make the screen say "updated just now".
	ChangedAt *time.Time `bson:"changedAt,omitempty" json:"changedAt,omitempty"`
	// The pages that moved in that change, so the screen can name them.
	Changed []string `bson:"changed,omitempty" json:"changed,omitempty"`
	// Changes seen while there was no IndexNow key: said on the screen, since
	// they were noticed and nobody was told.
	Unsent   int       `bson:"unsent,omitempty" json:"unsent,omitempty"`
	LastPing *llmsPing `bson:"lastPing,omitempty" json:"lastPing,omitempty"`
	Error    string    `bson:"error,omitempty" json:"error,omitempty"`
}

// llmsURLs are the six addresses of the two files, every language.
func (h *Handler) llmsURLs() []string {
	out := make([]string, 0, len(llmsLangs)*2)
	for _, l := range llmsLangs {
		out = append(out, h.origin()+l.Prefix+"/llms.txt", h.origin()+l.Prefix+"/llms-full.txt")
	}
	return out
}

// splitLLMsFull fingerprints every page in one full file.
//
// ⚠️ **A page runs from its `##` heading to the next page's heading** — found
// as the heading line just above each `URL:` line, not as "any line starting
// with ##". A blog post is free text, and an author's own `## Subheading` would
// otherwise end the post early: everything below it would stop being watched.
func splitLLMsFull(text string) []llmsPage {
	lines := strings.Split(text, "\n")
	type mark struct {
		url   string
		start int
	}
	var marks []mark
	for i, ln := range lines {
		if !strings.HasPrefix(ln, llmsURLMarker) {
			continue
		}
		start := i
		for j := i - 1; j >= 0; j-- {
			if strings.TrimSpace(lines[j]) == "" {
				continue
			}
			if strings.HasPrefix(lines[j], "## ") {
				start = j
			}
			break
		}
		marks = append(marks, mark{strings.TrimSpace(strings.TrimPrefix(ln, llmsURLMarker)), start})
	}
	out := make([]llmsPage, 0, len(marks))
	for k, m := range marks {
		end := len(lines)
		if k+1 < len(marks) {
			end = marks[k+1].start
		}
		sum := sha256.Sum256([]byte(strings.TrimSpace(strings.Join(lines[m.start:end], "\n"))))
		out = append(out, llmsPage{URL: m.url, Hash: hex.EncodeToString(sum[:])})
	}
	return out
}

// diffLLMs names the pages that are new, changed or gone.
//
// ⚠️ **Gone counts.** A deleted blog post is still in the engines' index until
// somebody tells them; submitting its URL makes them fetch it, meet the 404 and
// drop it — which is the protocol's own way of reporting a removal.
func diffLLMs(old, cur []llmsPage) []string {
	was := make(map[string]string, len(old))
	for _, p := range old {
		was[p.URL] = p.Hash
	}
	now := make(map[string]bool, len(cur))
	var out []string
	for _, p := range cur {
		now[p.URL] = true
		if was[p.URL] != p.Hash {
			out = append(out, p.URL)
		}
	}
	for _, p := range old {
		if !now[p.URL] {
			out = append(out, p.URL)
		}
	}
	sort.Strings(out)
	return out
}

// fetchText reads one file off the site.
func (h *Handler) fetchText(ctx context.Context, url string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: %d", url, res.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	return string(raw), err
}

// llmsMu keeps one check at a time: the ticker, a nudge after a blog save and
// the button can all land together, and two runs would each diff against the
// same stored state and ping the same pages twice.
var llmsMu sync.Mutex

// CheckLLMs reads the three full files, finds what changed since the last look
// and pings it. `force` sends the two files even when nothing changed — the
// button's meaning.
func (h *Handler) CheckLLMs(ctx context.Context, force, auto bool) (llmsState, error) {
	llmsMu.Lock()
	defer llmsMu.Unlock()

	coll := h.Store.DB.Collection("seo_state")
	var prev llmsState
	_ = coll.FindOne(ctx, bson.M{"_id": "llms"}).Decode(&prev)

	next := prev
	next.CheckedAt = time.Now()
	next.Error = ""

	var pages []llmsPage
	var files []llmsFile
	for _, l := range llmsLangs {
		full := h.origin() + l.Prefix + "/llms-full.txt"
		text, err := h.fetchText(ctx, full)
		if err != nil {
			// ⚠️ **Nothing is diffed against half a site.** One language failing
			// to load would read as every one of its pages having been
			// deleted, and the ping would tell the engines so.
			next.Error = err.Error()
			_, _ = coll.UpdateOne(ctx, bson.M{"_id": "llms"},
				bson.M{"$set": bson.M{"checkedAt": next.CheckedAt, "error": next.Error}},
				options.Update().SetUpsert(true))
			return next, err
		}
		p := splitLLMsFull(text)
		pages = append(pages, p...)
		files = append(files, llmsFile{
			Lang: l.Lang, Index: h.origin() + l.Prefix + "/llms.txt", Full: full,
			Bytes: len(text), Pages: len(p),
		})
	}
	next.Pages = pages
	next.Files = files

	baseline := len(prev.Pages) == 0
	changed := []string{}
	if !baseline {
		changed = diffLLMs(prev.Pages, pages)
	}
	if len(changed) > 0 {
		now := time.Now()
		next.ChangedAt = &now
		next.Changed = changed
		if len(next.Changed) > 50 {
			next.Changed = next.Changed[:50]
		}
	}
	if baseline && next.ChangedAt == nil {
		now := time.Now()
		next.ChangedAt = &now
	}

	if len(changed) > 0 || force {
		urls := append(append([]string{}, changed...), h.llmsURLs()...)
		if h.Cfg.IndexNowKey == "" {
			// ⚠️ Counted, not dropped silently: the screen says how many
			// changes nobody was told about, which is the argument for the key.
			next.Unsent += len(changed)
		} else {
			status, msg, err := h.indexNowSubmit(ctx, urls)
			ping := &llmsPing{At: time.Now(), Count: len(urls), Status: status, Message: msg, Auto: auto}
			if err != nil {
				ping.Message = err.Error()
			}
			next.LastPing = ping
			if err == nil && indexNowAccepted(status) {
				next.Unsent = 0
			}
		}
	}

	_, err := coll.UpdateOne(ctx, bson.M{"_id": "llms"}, bson.M{"$set": next},
		options.Update().SetUpsert(true))
	return next, err
}

// llmsStatus is what the SEO screen draws. Read, never computed: opening the
// screen must not fetch three megabytes from the site.
func (h *Handler) llmsStatus(ctx context.Context) map[string]any {
	var st llmsState
	_ = h.Store.DB.Collection("seo_state").FindOne(ctx, bson.M{"_id": "llms"}).Decode(&st)
	files := st.Files
	if len(files) == 0 {
		// Never checked yet: still show where the files are.
		for _, l := range llmsLangs {
			files = append(files, llmsFile{
				Lang: l.Lang, Index: h.origin() + l.Prefix + "/llms.txt",
				Full: h.origin() + l.Prefix + "/llms-full.txt",
			})
		}
	}
	out := map[string]any{
		"files":     files,
		"checkedAt": nilIfZero(st.CheckedAt),
		"changedAt": st.ChangedAt,
		"changed":   nonNil(st.Changed),
		"unsent":    st.Unsent,
		"lastPing":  st.LastPing,
		"error":     st.Error,
		"everyMin":  int(llmsEvery / time.Minute),
	}
	return out
}

func nilIfZero(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// nonNil keeps a list a list on the wire — a nil slice is `null`, and the
// screen reads `.length` (CLAUDE.md §10, the trap that has bitten twice).
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// SeoLLMsPing is the button: look now, and send the two files whatever changed.
func (h *Handler) SeoLLMsPing(w http.ResponseWriter, r *http.Request) {
	if h.Cfg.IndexNowKey == "" {
		httpx.Error(w, http.StatusBadRequest,
			"INDEXNOW_KEY sozlanmagan — kalit qo'yilmaguncha yuborish rad etiladi")
		return
	}
	st, err := h.CheckLLMs(r.Context(), true, false)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "llms fayllari o'qilmadi: "+err.Error())
		return
	}
	detail := "llms: tekshirildi"
	if st.LastPing != nil {
		detail = fmt.Sprintf("llms: %d ta manzil, javob %d", st.LastPing.Count, st.LastPing.Status)
	}
	h.logConsole(r.Context(), h.actorOrNil(r), "seo.llms", h.indexNowHost(), detail)
	httpx.JSON(w, http.StatusOK, h.llmsStatus(r.Context()))
}

// llmsEvery is how often the watcher reads the files back.
//
// ⚠️ **It is what catches the pages nobody saves in the console.** A help
// article changes with a deploy of keel.uz, which this service is not told
// about; a quarter of an hour is soon enough for an engine and rare enough to
// be nothing to the site.
const llmsEvery = 15 * time.Minute

// llmsSettle is how long a blog save waits before the files are read back.
// ⚠️ Past the site's one-minute cache of the blog (lib/blog.ts): read sooner,
// the file still has the old post and the change is found a quarter of an hour
// late, by the ticker.
const llmsSettle = 90 * time.Second

var (
	llmsNudgeMu    sync.Mutex
	llmsNudgeTimer *time.Timer
)

// NudgeLLMs schedules a check shortly after something the files are built from
// changed. Several saves in a row collapse into one check after the last.
func (h *Handler) NudgeLLMs() {
	llmsNudgeMu.Lock()
	defer llmsNudgeMu.Unlock()
	if llmsNudgeTimer != nil {
		llmsNudgeTimer.Stop()
	}
	llmsNudgeTimer = time.AfterFunc(llmsSettle, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if _, err := h.CheckLLMs(ctx, false, true); err != nil {
			log.Printf("llms: %v", err)
		}
	})
}

// WatchLLMs is the watcher's loop, started with the server.
func (h *Handler) WatchLLMs(ctx context.Context) {
	// Not at once: at boot keel.uz may be restarting in the same deploy, and a
	// failed first read is noise in a log that is read for real failures.
	select {
	case <-ctx.Done():
		return
	case <-time.After(2 * time.Minute):
	}
	run := func() {
		c, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		if _, err := h.CheckLLMs(c, false, true); err != nil {
			log.Printf("llms: %v", err)
		}
	}
	run()
	t := time.NewTicker(llmsEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}
