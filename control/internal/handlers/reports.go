package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"keel-control/internal/httpx"
	"keel-control/internal/models"
)

// Crash reports: what broke, arriving before the owner writes to us.
//
// # Why this exists at all
//
// The support queue is the other end of the same problem, and it depends on a
// restaurant noticing, deciding it is worth reporting, and describing it. Every
// one of those loses faults: the ones that happen to one cashier on one tablet,
// the ones people work around, and the ones nobody can put into words. This
// path has none of those steps in it.
//
// # The shape of the intake, and what it refuses
//
// One request may carry a batch — a browser that crashed six times while
// offline sends them together when it comes back — and each report is folded
// into a group by fingerprint. Nothing here trusts the caller:
//
//   - **The restaurant comes from the link token**, never from the body.
//   - **Counts are capped per tenant per day** (maxReportsPerDay). A loop in
//     one panel must not be able to fill the platform's database.
//   - **Text is truncated** before it is stored, not after.
//
// ⚠️ **Being over the cap does not stop the count.** When a tenant has filed
// its day's worth, further reports still increment the group they belong to and
// stop creating new groups and new samples. Dropping the increment as well
// would make a storm look like it had ended — which is the one reading of this
// screen that must never be available.

const (
	// How many *distinct* faults one restaurant may open in a day. A restaurant
	// with thirty new bugs in one day has one bug.
	maxReportsPerDay = 40
	// Ceilings on the text. Generous enough for a real stack, small enough that
	// a hostile or broken client cannot post a book.
	maxReportMessage = 400
	maxReportStack   = 4000
	maxReportContext = 300
	// A batch bigger than this is a client that has lost track of itself.
	maxReportBatch = 20
)

// reportIn is one occurrence as an app sends it.
//
// ⚠️ Deliberately small, and deliberately without a body, headers, or anything
// resembling a request payload. What is being collected is *our* fault, not the
// restaurant's data — and a field that could carry a guest's phone number will
// eventually carry one.
type reportIn struct {
	App      string `json:"app"`
	Version  string `json:"version"`
	Platform string `json:"platform"`
	Message  string `json:"message"`
	Stack    string `json:"stack"`
	// Route, screen, or endpoint.
	Where   string `json:"where"`
	Context string `json:"context"`
	Branch  string `json:"branch"`
	Role    string `json:"role"`
	// An opaque per-install id, so "one tablet" can be told from "everybody".
	// ⚠️ Not a user, not a device fingerprint: whatever the app already has to
	// call itself, hashed here before it is stored.
	Session string `json:"session"`
	At      string `json:"at"`
}

// Report is a tenant server forwarding what its apps reported.
func (h *Handler) Report(w http.ResponseWriter, r *http.Request) {
	t, err := h.tenantFromLink(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body struct {
		Reports []reportIn `json:"reports"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	if len(body.Reports) > maxReportBatch {
		body.Reports = body.Reports[:maxReportBatch]
	}

	now := time.Now()
	day := now.Format("2006-01-02")
	// Counted once for the whole batch rather than per report: the question the
	// cap answers is "has this restaurant already filled the screen today", and
	// asking it twenty times costs twenty round trips to answer it the same way.
	opened, _ := h.Store.Reports.CountDocuments(r.Context(), bson.M{
		"slug": t.Slug, "todayOn": day,
	})
	room := opened < maxReportsPerDay

	stored := 0
	for _, in := range body.Reports {
		if h.storeReport(r.Context(), t, in, now, day, room) {
			stored++
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "stored": stored})
}

func (h *Handler) storeReport(
	ctx context.Context, t *models.Tenant, in reportIn,
	now time.Time, day string, room bool,
) bool {
	msg := clip(strings.TrimSpace(in.Message), maxReportMessage)
	if msg == "" {
		return false
	}
	app := clip(strings.TrimSpace(in.App), 40)
	if app == "" {
		app = "unknown"
	}
	key := Fingerprint(msg, clip(strings.TrimSpace(in.Where), 200))

	at := now
	if in.At != "" {
		if parsed, err := time.Parse(time.RFC3339, in.At); err == nil {
			// ⚠️ Clamped to now. A tablet with a wrong clock is ordinary — the
			// till has one — and a report dated next March sits at the top of a
			// list sorted by recency forever.
			if parsed.Before(now) && parsed.After(now.Add(-30*24*time.Hour)) {
				at = parsed
			}
		}
	}

	filter := bson.M{"slug": t.Slug, "app": app, "key": key}
	set := bson.M{
		"restaurant":     t.Name,
		"message":        msg,
		"lastAt":         at,
		"latestVersion":  clip(strings.TrimSpace(in.Version), 40),
		"latestPlatform": clip(strings.TrimSpace(in.Platform), 80),
	}
	if w := clip(strings.TrimSpace(in.Where), 200); w != "" {
		set["where"] = w
	}
	inc := bson.M{"count": 1}
	update := bson.M{"$set": set, "$inc": inc}

	if !room {
		// Over the cap: the group still counts, and nothing new is created.
		// ⚠️ No upsert — that is the half that has to stop.
		res, err := h.Store.Reports.UpdateOne(ctx, filter, update)
		return err == nil && res.MatchedCount > 0
	}

	// A new day for this group resets the daily counter; the same day adds to
	// it. Two statements because Mongo cannot do "increment or reset" in one,
	// and the reset has to lose to a concurrent increment rather than the other
	// way round — an undercount for one second beats a count that restarts.
	var existing models.ErrorGroup
	found := h.Store.Reports.FindOne(ctx, filter).Decode(&existing) == nil
	if found && existing.TodayOn == day {
		inc["today"] = 1
	} else {
		set["today"] = 1
		set["todayOn"] = day
	}

	sample := models.ErrorSample{
		At:       at,
		Stack:    clip(strings.TrimSpace(in.Stack), maxReportStack),
		Context:  clip(strings.TrimSpace(in.Context), maxReportContext),
		Version:  clip(strings.TrimSpace(in.Version), 40),
		Platform: clip(strings.TrimSpace(in.Platform), 80),
		Branch:   clip(strings.TrimSpace(in.Branch), 80),
		Role:     clip(strings.TrimSpace(in.Role), 40),
	}
	update["$push"] = bson.M{"samples": bson.M{
		"$each": []models.ErrorSample{sample},
		// ⚠️ Negative slice keeps the **newest**. Positive keeps the oldest,
		// which is the one mistake here that looks identical in testing — the
		// samples are there, they are simply from the first hour of a fault
		// that has since changed shape.
		"$slice": -models.MaxReportSamples,
	}}
	update["$setOnInsert"] = bson.M{
		"slug":         t.Slug,
		"key":          key,
		"app":          app,
		"firstAt":      at,
		"firstVersion": clip(strings.TrimSpace(in.Version), 40),
		"platform":     clip(strings.TrimSpace(in.Platform), 80),
		"resolved":     false,
	}
	if s := strings.TrimSpace(in.Session); s != "" {
		// Hashed before storage: whatever the app calls itself is its business,
		// and all this screen needs is whether two reports came from one place.
		update["$addToSet"] = bson.M{"sessions": shortHash(s)}
	}

	_, err := h.Store.Reports.UpdateOne(ctx, filter, update,
		options.Update().SetUpsert(true))
	if err != nil {
		return false
	}
	h.countReportUsers(ctx, filter)
	return true
}

// countReportUsers keeps the "how many installs" number in step with the set it
// is counted from.
//
// ⚠️ A second write rather than arithmetic on the first: `$addToSet` does not
// say whether it added anything, so incrementing beside it would count every
// occurrence as a new install — and "forty tablets" instead of "one tablet,
// forty times" is the difference between a chain-wide outage and a broken
// browser cache.
func (h *Handler) countReportUsers(ctx context.Context, filter bson.M) {
	var g models.ErrorGroup
	if h.Store.Reports.FindOne(ctx, filter).Decode(&g) != nil {
		return
	}
	if g.Users == len(g.Sessions) {
		return
	}
	_, _ = h.Store.Reports.UpdateOne(ctx, filter,
		bson.M{"$set": bson.M{"users": len(g.Sessions)}})
}

// ---- What "the same fault" means ----

// Numbers, ids and quoted values inside a message are the parts that differ
// between two occurrences of one bug: "order 6f3a not found" and "order 91bc
// not found" are one fault, and left alone they are two rows and then a
// thousand.
var (
	reportHex   = regexp.MustCompile(`\b[0-9a-fA-F]{8,}\b`)
	reportNum   = regexp.MustCompile(`\b\d+\b`)
	reportQuote = regexp.MustCompile(`"[^"]*"|'[^']*'`)
	reportSpace = regexp.MustCompile(`\s+`)
)

// Fingerprint decides which reports are the same fault.
//
// ⚠️ **Deliberately blunt, and erring towards over-grouping.** Two distinct
// bugs merged into one row are found the moment somebody opens it and reads two
// different stacks; one bug split across four hundred rows is never found at
// all, because nothing about it looks important. The cost is asymmetric, so the
// rule is too.
func Fingerprint(message, where string) string {
	norm := strings.ToLower(message)
	norm = reportQuote.ReplaceAllString(norm, "?")
	norm = reportHex.ReplaceAllString(norm, "?")
	norm = reportNum.ReplaceAllString(norm, "?")
	norm = reportSpace.ReplaceAllString(norm, " ")
	norm = strings.TrimSpace(norm)

	// The route joins the fingerprint, because the same message from two
	// screens is usually two bugs — but only its shape: `/admin/users/6f3a`
	// and `/admin/users/91bc` are one screen.
	route := reportHex.ReplaceAllString(strings.ToLower(where), "?")
	route = reportNum.ReplaceAllString(route, "?")

	sum := sha256.Sum256([]byte(norm + "\n" + route))
	return hex.EncodeToString(sum[:8])
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:6])
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// Cut on a rune boundary: a stack sliced through a UTF-8 sequence renders
	// as a replacement character in the console and reads as corruption.
	for n > 0 && !utf8Start(s[n]) {
		n--
	}
	return s[:n] + "…"
}

func utf8Start(b byte) bool { return b&0xC0 != 0x80 }

// ---- The console's side ----

// ConsoleReports is the list, newest fault first.
func (h *Handler) ConsoleReports(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := bson.M{}
	if slug := strings.TrimSpace(q.Get("slug")); slug != "" {
		filter["slug"] = slug
	}
	if app := strings.TrimSpace(q.Get("app")); app != "" {
		filter["app"] = app
	}
	// ⚠️ Open by default. A screen that opens showing everything ever fixed is
	// a screen where today's three faults are below four hundred old ones.
	switch q.Get("state") {
	case "resolved":
		filter["resolved"] = true
	case "all":
	default:
		filter["resolved"] = bson.M{"$ne": true}
	}
	if s := strings.TrimSpace(q.Get("q")); s != "" {
		rx := primitive.Regex{Pattern: regexp.QuoteMeta(s), Options: "i"}
		filter["$or"] = []bson.M{
			{"message": rx}, {"where": rx}, {"restaurant": rx},
		}
	}

	limit := int64(100)
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 && n <= 300 {
		limit = int64(n)
	}
	cur, err := h.Store.Reports.Find(r.Context(), filter,
		options.Find().SetSort(bson.M{"lastAt": -1}).SetLimit(limit).
			// The samples are the expensive half and the list does not draw
			// them. Fetching them here is the whole screen's payload spent on
			// text nobody reads until they click.
			SetProjection(bson.M{"samples": 0, "sessions": 0}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	groups := []models.ErrorGroup{}
	if err := cur.All(r.Context(), &groups); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"groups": groups})
}

// ConsoleReport is one fault with its samples.
func (h *Handler) ConsoleReport(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	var g models.ErrorGroup
	if err := h.Store.Reports.FindOne(r.Context(), bson.M{"_id": id}).Decode(&g); err != nil {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"group": g})
}

// ConsoleReportResolve marks a fault dealt with.
//
// ⚠️ **It does not stop collection, and it does not delete.** The group keeps
// counting, so a resolved fault that starts counting again shows up as exactly
// what it is — a fix that did not hold — instead of arriving a second time as a
// brand new bug with no history.
func (h *Handler) ConsoleReportResolve(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	var body struct {
		Resolved bool   `json:"resolved"`
		Note     string `json:"note"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body)

	var g models.ErrorGroup
	if h.Store.Reports.FindOne(r.Context(), bson.M{"_id": id}).Decode(&g) != nil {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}

	now := time.Now()
	set := bson.M{"resolved": body.Resolved, "note": clip(strings.TrimSpace(body.Note), 300)}
	unset := bson.M{}
	if body.Resolved {
		set["resolvedAt"] = now
		if u, err := h.actor(r); err == nil {
			set["resolvedBy"] = u.Name
		}
		// The count at the moment of the fix, so "has it come back" is
		// arithmetic rather than somebody's memory of a number.
		set["resolvedCount"] = g.Count
	} else {
		unset["resolvedAt"] = ""
		unset["resolvedBy"] = ""
		unset["resolvedCount"] = ""
	}
	update := bson.M{"$set": set}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	if _, err := h.Store.Reports.UpdateOne(r.Context(), bson.M{"_id": id}, update); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
