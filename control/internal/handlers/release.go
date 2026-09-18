package handlers

// ---- What version everything is on, and raising it ----
//
// "Which version are you on?" is the first question after "is it up?", and
// until this panel existed the honest answer was that nobody could tell. Every
// part of Keel carries its own constant (see version.go for why), which is
// correct and means the parts can silently disagree — and the disagreement is
// never visible from inside any one of them.
//
// So the panel reports each part as **that part says it is**, not as the repo
// says it should be:
//
//   - the console, from the constant in this binary;
//   - the restaurants' servers, by asking each running container's own /health;
//   - the Windows till, from the manifest the tills are actually offered.
//
// ⚠️ **The point is the drift, not the number.** A deploy that builds an image
// and leaves the containers running is the trap this platform has already
// fallen into twice — commit right, containers healthy, checks green, old code
// serving. A row saying "12 ta v0.2.1 · 3 ta v0.2.0" is that failure, stated,
// on the screen somebody already looks at. A panel that printed one number
// taken from this binary would have shown v0.2.1 through the whole of it.
//
// ⚠️ **Raising it is a release, not a setting.** The version is a constant
// compiled into seven artefacts; nothing here can change the number the running
// process reports. What the button does is ask GitHub to run the release
// workflow, which writes the seven files, pushes, and deploys — and the number
// on this screen moves minutes later, when the container answering this request
// is a different build. The gap between "asked" and "arrived" is real, so it is
// shown rather than hidden behind a spinner that lies.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"keel-control/internal/httpx"
	"keel-control/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// How long a release may be in flight before the panel stops calling it
// running. A full deploy builds four images on one VPS; twenty minutes is a
// slow one and thirty is one that is not coming.
const releaseGiveUp = 30 * time.Minute

// How long a sweep of the tenants' versions is reused for.
//
// ⚠️ **Cached because it costs a container inspect and an HTTP call per
// customer.** The panel is on the overview, which is left open on a desk all
// day; without this, fifty restaurants would be probed twice a minute forever
// to answer a question whose answer changes about once a week.
const tenantVersionTTL = time.Minute

type tenantVersionCache struct {
	mu   sync.Mutex
	at   time.Time
	part versionPart
}

var tenantVersions tenantVersionCache

// versionPart is one row of the panel.
type versionPart struct {
	// Stable id; the console translates it. Never shown raw.
	ID string `json:"id"`
	// What this part reports. Empty means it could not be asked — which is a
	// different fact from "it is behind", and drawn differently.
	Version string `json:"version,omitempty"`
	// True when this part agrees with the console's own version.
	//
	// ⚠️ Decided here rather than in the browser, because the till's number is
	// bare ("0.2.0") and the others carry a "v" — a comparison written twice
	// is a comparison that is right once.
	Match bool `json:"match"`
	// A sentence for the operator: how many containers, which installer file,
	// or why the question could not be answered. Shown as-is.
	Note string `json:"note,omitempty"`
	// Set when this part is something we could not reach at all, so the row
	// reads as unknown rather than as wrong.
	Unknown bool `json:"unknown,omitempty"`
}

// Version answers the console's version panel.
func (h *Handler) Version(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// ⚠️ **Two separate facts, because they need different screens.** "This
	// deployment cannot cut releases" is worth a sentence — somebody is looking
	// at a laptop, or at a server where the token was never set. "You may not
	// cut one" is worth nothing at all: an admin reading the version panel
	// should see the versions and no buttons, not an explanation of a power
	// they were never given. Folding the two into one flag would print the
	// GITHUB_TOKEN sentence at every account that is simply not the owner.
	mayRelease := false
	if u, err := h.actor(r); err == nil {
		mayRelease = u.Can(models.CanRelease)
	}
	res := map[string]any{
		"version": Version,
		"stage":   Stage,
		"next":    NextVersions(Version),
		"parts":   h.versionParts(ctx),
		// Whether this account may press the button at all.
		"mayRelease": mayRelease,
		// Whether the button is wired up on this deployment.
		"releaseWired": h.canRelease(),
	}
	if rel, ok := h.resolveRelease(ctx); ok {
		res["release"] = rel
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) canRelease() bool {
	return strings.TrimSpace(h.Cfg.GitHubToken) != "" &&
		strings.TrimSpace(h.Cfg.GitHubRepo) != ""
}

// versionParts asks everything that can be asked.
func (h *Handler) versionParts(ctx context.Context) []versionPart {
	return []versionPart{h.tenantVersionPart(ctx), h.tillVersionPart()}
}

// ---- The restaurants' servers ----

// tenantVersionPart asks every running tenant container what build it is.
//
// ⚠️ **Asked over HTTP, not read from the image tag.** The tag says what a
// container *would* be created from; this says what the process is serving.
// They differ for exactly as long as a rollout has not run, which is the window
// this row exists to make visible.
func (h *Handler) tenantVersionPart(ctx context.Context) versionPart {
	tenantVersions.mu.Lock()
	defer tenantVersions.mu.Unlock()
	if time.Since(tenantVersions.at) < tenantVersionTTL && tenantVersions.at.After(time.Time{}) {
		return tenantVersions.part
	}
	p := h.probeTenantVersions(ctx)
	tenantVersions.at, tenantVersions.part = time.Now(), p
	return p
}

func (h *Handler) probeTenantVersions(ctx context.Context) versionPart {
	part := versionPart{ID: "tenants"}
	if h.Docker == nil {
		part.Unknown = true
		part.Note = "bu serverda konteynerlar boshqarilmaydi"
		return part
	}
	states, err := h.Docker.States(ctx)
	if err != nil {
		part.Unknown = true
		part.Note = err.Error()
		return part
	}

	var slugs []string
	for slug, st := range states {
		// Only containers that are actually serving. A stopped one is a
		// suspended customer or a failure the overview already reports, and
		// counting it as "unknown version" would put a permanent warning on
		// this row for a reason that has nothing to do with versions.
		if st.Status == "running" {
			slugs = append(slugs, slug)
		}
	}
	if len(slugs) == 0 {
		part.Unknown = true
		part.Note = "ishlab turgan konteyner yo'q"
		return part
	}
	sort.Strings(slugs)

	// Bounded: one VPS, and this is a background curiosity next to the work the
	// same box is doing for paying customers.
	const workers = 8
	type result struct{ version string }
	out := make([]result, len(slugs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)
	for i, slug := range slugs {
		wg.Add(1)
		go func(i int, slug string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i] = result{version: h.tenantVersion(ctx, slug)}
		}(i, slug)
	}
	wg.Wait()

	counts := map[string]int{}
	unknown := 0
	for _, r := range out {
		if r.version == "" {
			unknown++
			continue
		}
		counts[r.version]++
	}

	// The majority is what the row shows; everything else is the note, because
	// the note is the whole message when there is more than one.
	best, bestN := "", 0
	for v, n := range counts {
		if n > bestN || (n == bestN && v > best) {
			best, bestN = v, n
		}
	}
	part.Version = best
	part.Match = best == Version
	part.Unknown = best == ""

	var bits []string
	for _, v := range sortedKeys(counts) {
		bits = append(bits, fmt.Sprintf("%d ta %s", counts[v], v))
	}
	if unknown > 0 {
		// ⚠️ Named as an older build rather than as an error. A tenant that
		// answers /health without a version field is running a backend from
		// before /health carried one — which is precisely "behind", and the
		// most behind of anything on this screen.
		bits = append(bits, fmt.Sprintf("%d ta eski build (versiyani aytmaydi)", unknown))
	}
	part.Note = strings.Join(bits, " · ")
	if unknown > 0 && best == "" {
		part.Match = false
	}
	return part
}

// tenantVersion asks one container. Empty on any failure: this is a panel, and
// a customer whose container is mid-restart must not turn it into an error.
func (h *Handler) tenantVersion(ctx context.Context, slug string) string {
	ip, err := h.Docker.IP(ctx, slug)
	if err != nil || ip == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+ip+":8080/health", nil)
	if err != nil {
		return ""
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode >= 400 {
		return ""
	}
	var body struct {
		Version string `json:"version"`
	}
	// A reader cap: /health is two fields, and this is an endpoint we would
	// still like to survive being wrong about.
	if err := json.NewDecoder(io.LimitReader(res.Body, 4<<10)).Decode(&body); err != nil {
		return ""
	}
	return strings.TrimSpace(body.Version)
}

// ---- The Windows till ----

// tillVersionPart reads the manifest the tills are being offered.
//
// ⚠️ **The manifest, not the constant in desktop/version.go.** What matters is
// what a monoblock on a counter is told to install; the source file says what
// the next build *will* claim, and the two are different for the whole of the
// window between raising the version and publishing the installer — which is
// the window where somebody asks why the tills have not updated.
func (h *Handler) tillVersionPart() versionPart {
	part := versionPart{ID: "till"}
	dir := h.tillReleaseDir()
	if dir == "" {
		part.Unknown = true
		part.Note = "reliz papkasi sozlanmagan (TILL_RELEASE_DIR)"
		return part
	}
	raw, err := os.ReadFile(filepath.Join(dir, "latest.json"))
	if err != nil {
		part.Unknown = true
		part.Note = "latest.json o'qilmadi"
		return part
	}
	var rel tillRelease
	if err := json.Unmarshal(raw, &rel); err != nil {
		part.Unknown = true
		part.Note = "latest.json buzuq"
		return part
	}
	part.Version = strings.TrimSpace(rel.Version)
	// ⚠️ Compared without the "v". The till's number is bare because Windows
	// wants it bare — see scripts/set-version.sh — and comparing the two shapes
	// directly would report every till as behind, forever, on a screen whose
	// entire job is to report being behind.
	part.Match = part.Version != "" && part.Version == strings.TrimPrefix(Version, "v")
	part.Note = installerName(rel.URL)
	return part
}

// installerName is the file a till would download, for the note.
func installerName(url string) string {
	if i := strings.LastIndex(url, "file="); i >= 0 {
		return url[i+len("file="):]
	}
	return url
}

// ---- Raising it ----

// BumpVersion asks GitHub to release the next version.
//
// ⚠️ **Owner only, and checked here rather than on the router**, so the reason
// can be written beside it: this replaces every container on the box, including
// the one serving every customer's orders. It is the most consequential button
// in the console and it is one press — which is the argument for the narrowest
// possible gate, not for a confirmation dialog nobody reads.
func (h *Handler) BumpVersion(w http.ResponseWriter, r *http.Request) {
	u, err := h.actor(r)
	if err != nil {
		fail(w, err)
		return
	}
	if !u.Can(models.CanRelease) {
		fail(w, errForbidden)
		return
	}
	if !h.canRelease() {
		httpx.Error(w, http.StatusBadRequest,
			"bu serverda reliz sozlanmagan (GITHUB_TOKEN / GITHUB_REPO yo'q)")
		return
	}

	var body struct {
		Part string `json:"part"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "so'rov o'qilmadi")
		return
	}
	next, ok := NextVersion(Version, strings.TrimSpace(body.Part))
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "qaysi qism ko'tarilishi noma'lum")
		return
	}

	// ⚠️ **One release at a time, and the guard is the stored row rather than a
	// mutex.** The process that would hold a mutex is the process the release
	// is about to kill; a second press two minutes later would meet a fresh
	// container with an empty lock and start a second release onto a tree the
	// first one has already moved.
	if cur, running := h.resolveRelease(r.Context()); running && cur.Status == "running" {
		httpx.Error(w, http.StatusConflict,
			fmt.Sprintf("%s relizi hali ketmoqda", cur.Version))
		return
	}

	rel := models.Release{
		Version:     next,
		From:        Version,
		Part:        strings.TrimSpace(body.Part),
		RequestedAt: time.Now(),
		RequestedBy: u.Username,
		Status:      "running",
		RunURL: fmt.Sprintf("https://github.com/%s/actions/workflows/%s",
			strings.Trim(h.Cfg.GitHubRepo, "/"), strings.TrimSpace(h.Cfg.ReleaseWorkflow)),
	}

	// ⚠️ **Written before the dispatch, and removed if the dispatch fails.**
	// The other order loses the record when GitHub accepts the call and this
	// process is replaced before it can write — which is not unlikely, because
	// being replaced is what it just asked for.
	if err := h.saveRelease(r.Context(), rel); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "reliz yozilmadi")
		return
	}
	if err := h.dispatchRelease(r.Context(), next); err != nil {
		_, _ = h.Store.Releases.DeleteOne(r.Context(), bson.M{"_id": models.ReleaseDocID})
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	h.logConsole(r.Context(), u, "release.bump", next, Version+" → "+next)
	httpx.JSON(w, http.StatusOK, map[string]any{"release": rel})
}

// dispatchRelease starts the workflow.
//
// ⚠️ **workflow_dispatch rather than a commit from here.** Writing the seven
// files would mean a token that can push to `main`, which is a token that can
// deploy anything — and it would put the version-writing logic in a second
// place beside scripts/set-version.sh, where the two would disagree on the
// first file that moves. This way the console knows only a version number, and
// everything about *how* a release is made stays in the repository, reviewed,
// with the test that checks it right beside it.
func (h *Handler) dispatchRelease(ctx context.Context, version string) error {
	repo := strings.Trim(strings.TrimSpace(h.Cfg.GitHubRepo), "/")
	file := strings.TrimSpace(h.Cfg.ReleaseWorkflow)
	payload, err := json.Marshal(map[string]any{
		"ref":    "main",
		"inputs": map[string]string{"version": version},
	})
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/actions/workflows/%s/dispatches", repo, file)
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(h.Cfg.GitHubToken))
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("GitHub javob bermadi: %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	// 204 is the documented success and it carries no body — which is why the
	// stored row links to the workflow's page rather than to this run.
	if res.StatusCode == http.StatusNoContent || res.StatusCode == http.StatusOK {
		return nil
	}
	// The message is GitHub's, verbatim. "Workflow does not have
	// workflow_dispatch trigger" and "Resource not accessible by integration"
	// are two completely different mistakes, and paraphrasing them into "reliz
	// boshlanmadi" throws away the only sentence that says which.
	var e struct {
		Message string `json:"message"`
	}
	_ = json.NewDecoder(io.LimitReader(res.Body, 8<<10)).Decode(&e)
	if strings.TrimSpace(e.Message) == "" {
		return fmt.Errorf("GitHub: %s", res.Status)
	}
	return fmt.Errorf("GitHub: %s", e.Message)
}

// ---- Is it there yet ----

// resolveRelease reads the stored release and decides what actually became of
// it, writing the verdict back when it has changed.
//
// ⚠️ **Decided on the read path, not by a goroutine.** The event this waits for
// is this process being replaced: any watcher started before the release is
// dead by the time the answer exists, and the process that knows the answer is
// the new one, on its first request. So the comparison happens wherever the
// question is asked.
//
// ⚠️ **"Done" means the running constant equals what was asked for** — never
// that the workflow reported success. A green workflow with an unreplaced
// container is the exact failure this platform has shipped before, and a panel
// that trusted CI would have agreed with it.
func (h *Handler) resolveRelease(ctx context.Context) (models.Release, bool) {
	var rel models.Release
	err := h.Store.Releases.FindOne(ctx, bson.M{"_id": models.ReleaseDocID}).Decode(&rel)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			// Nothing to show and nothing to claim: an unreadable row must not
			// become "no release is running", which is what would let a second
			// one start.
			return rel, false
		}
		return rel, false
	}
	if rel.Status != "running" {
		return rel, true
	}
	switch {
	case rel.Version == Version:
		rel.Status = "done"
		rel.FinishedAt = time.Now()
	case time.Since(rel.RequestedAt) > releaseGiveUp:
		rel.Status = "stale"
	default:
		return rel, true
	}
	_ = h.saveRelease(ctx, rel)
	return rel, true
}

func (h *Handler) saveRelease(ctx context.Context, rel models.Release) error {
	_, err := h.Store.Releases.UpdateOne(ctx,
		bson.M{"_id": models.ReleaseDocID},
		bson.M{"$set": rel},
		options.Update().SetUpsert(true))
	return err
}

// ---- small helpers ----

func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
