package handlers

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"time"

	"keel-control/internal/httpx"
	"keel-control/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The public status page's data.
//
// What it must not be is a page that says "all systems operational" because
// nothing has told it otherwise. A status page whose green is the absence of
// news is worse than none: it is read as evidence during the one hour it is
// wrong, and it costs the trust it was built to earn.
//
// So the number shown is measured, not assumed. Every minute the control plane
// checks itself — its own database, and whether the tenant containers it
// believes should be running actually are — and folds the result into an hourly
// bucket. Uptime is those buckets divided, over whatever window the page asks
// for.
//
// **Hourly buckets, not per-minute rows.** A minute of history per sample is
// half a million documents a year to render a strip nobody zooms into; an hour
// is 8,760, and "which hour was it down" is the resolution anybody actually
// asks about. The bucket is a single `$inc`, so a sample costs one round trip
// and never grows the collection.

// statusSample is one minute's verdict, folded into its hour.
type statusBucket struct {
	// "2006-01-02T15" — the hour, local, and the document's own id.
	Hour string `bson:"_id" json:"hour"`
	// Checks taken, and how many passed.
	Checks int `bson:"checks" json:"checks"`
	OK     int `bson:"ok" json:"ok"`
	// Slowest database round trip seen in the hour, milliseconds. A platform
	// that is "up" at four seconds a request is down to everybody using it.
	MaxMs int64 `bson:"maxMs" json:"maxMs"`
	// The last failure's wording, so an hour that dipped can explain itself.
	Note string `bson:"note,omitempty" json:"note,omitempty"`
}

// SampleStatus takes one measurement and folds it into the current hour.
//
// Deliberately cheap and deliberately honest about what it can see: the
// control plane's own database, and the tenant containers. It cannot check
// Caddy from inside — a probe that has to reach us to report us unreachable
// reports nothing at all — and it does not pretend to.
func (h *Handler) SampleStatus(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	started := time.Now()
	ok, note := true, ""

	// The database every screen depends on.
	if err := h.Store.DB.Client().Ping(ctx, nil); err != nil {
		ok, note = false, "database: "+err.Error()
	}
	ms := time.Since(started).Milliseconds()

	// The customers we believe should be serving. A container that should be
	// running and is not is an outage for that restaurant, whatever the
	// database says.
	//
	// ⚠️ **Down in two samples in a row, not in one.** A deploy or a rollout
	// recreates a container in seconds, and a sample that lands inside those
	// seconds used to count the whole minute as an outage — which is how the
	// 90-day strip filled with red days that were 1,439 minutes of 1,440 fine.
	// A container that stays down is counted from its second minute on; one
	// that was only being replaced is never counted. The database check above
	// is left immediate: a control plane that cannot reach its own database is
	// down for everybody, deploy or not.
	if ok && h.Docker != nil {
		if down := h.confirmDown(h.downTenantSlugs(ctx)); down > 0 {
			ok = false
			note = "tenant containers down: " + itoa(down)
		}
	}

	hour := time.Now().Format("2006-01-02T15")
	inc := bson.M{"checks": 1}
	if ok {
		inc["ok"] = 1
	}
	update := bson.M{
		"$inc": inc,
		"$max": bson.M{"maxMs": ms},
	}
	if note != "" {
		update["$set"] = bson.M{"note": clamp(note, 200)}
	}
	if _, err := h.Store.Status.UpdateOne(ctx,
		bson.M{"_id": hour}, update, options.Update().SetUpsert(true),
	); err != nil {
		// Losing a sample is not worth a log line every minute; losing every
		// sample shows up on the page as a gap, which is the honest outcome.
		_ = err
	}
}

// downTenants counts customers whose container is not running when it should
// be. A stopped customer is not an outage — it was stopped on purpose.
func (h *Handler) downTenants(ctx context.Context) int {
	return len(h.downTenantSlugs(ctx))
}

// downTenantSlugs names them, so a sample can be compared with the one before.
func (h *Handler) downTenantSlugs(ctx context.Context) []string {
	cur, err := h.Store.Tenants.Find(ctx, bson.M{
		"status": bson.M{"$in": []string{models.StatusActive, models.StatusTrial}},
	})
	if err != nil {
		return nil
	}
	var tenants []models.Tenant
	if err := cur.All(ctx, &tenants); err != nil {
		return nil
	}
	var down []string
	for _, t := range tenants {
		st, err := h.Docker.Status(ctx, t.Slug)
		if err != nil || st.Status != "running" {
			down = append(down, t.Slug)
		}
	}
	return down
}

// statusDown is which tenants the previous sample found down.
//
// ⚠️ **In memory, and that is enough.** One control plane samples, once a
// minute, from one goroutine; after a restart the first sample has nothing to
// compare with and counts nobody — which errs towards a missed minute, never
// towards an outage that did not happen.
var statusDown struct {
	sync.Mutex
	prev map[string]bool
}

func (h *Handler) confirmDown(now []string) int {
	statusDown.Lock()
	defer statusDown.Unlock()
	n, next := confirmedDown(statusDown.prev, now)
	statusDown.prev = next
	return n
}

// confirmedDown counts the tenants down now that were also down last time, and
// returns the set to remember for the next sample.
func confirmedDown(prev map[string]bool, now []string) (int, map[string]bool) {
	next := make(map[string]bool, len(now))
	n := 0
	for _, slug := range now {
		next[slug] = true
		if prev[slug] {
			n++
		}
	}
	return n, next
}

// StatusPage is what keel.uz/status renders. Public, and deliberately so: a
// status page behind a login is a status page for the people who already know.
func (h *Handler) StatusPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now()

	// 90 days of hours. At 24 buckets a day that is ~2,160 documents — small
	// enough to read whole, and the alternative (an aggregation per window)
	// would need three of them for one page.
	from := now.AddDate(0, 0, -90).Format("2006-01-02T15")
	cur, err := h.Store.Status.Find(ctx,
		bson.M{"_id": bson.M{"$gte": from}},
		options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var buckets []statusBucket
	if err := cur.All(ctx, &buckets); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	byHour := make(map[string]statusBucket, len(buckets))
	for _, b := range buckets {
		byHour[b.Hour] = b
	}

	// The last 48 hours, one bar each — the resolution that answers "is it
	// working right now, and was it an hour ago".
	hours := make([]hourPoint, 0, 48)
	for i := 47; i >= 0; i-- {
		t := now.Add(time.Duration(-i) * time.Hour)
		key := t.Format("2006-01-02T15")
		b, seen := byHour[key]
		hours = append(hours, hourPoint{
			Hour: key, Checks: b.Checks, OK: b.OK, MaxMs: b.MaxMs,
			Note: b.Note, Seen: seen,
		})
	}

	// 90 days, one bar each, for the long view.
	byDay := map[string]*dayPoint{}
	for _, b := range buckets {
		day := b.Hour[:10]
		d := byDay[day]
		if d == nil {
			d = &dayPoint{Day: day}
			byDay[day] = d
		}
		d.Checks += b.Checks
		d.OK += b.OK
	}
	days := make([]dayPoint, 0, 90)
	for i := 89; i >= 0; i-- {
		key := now.AddDate(0, 0, -i).Format("2006-01-02")
		if d, ok := byDay[key]; ok {
			days = append(days, *d)
		} else {
			days = append(days, dayPoint{Day: key})
		}
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Day < days[j].Day })

	checks, okCount := 0, 0
	for _, b := range buckets {
		checks += b.Checks
		okCount += b.OK
	}

	// "Up" means the most recent bucket that has any samples passed all of
	// them. Reading only the current hour would show a fresh unknown as an
	// outage for the first minute of every hour.
	up, since := currentState(hours)

	httpx.JSON(w, http.StatusOK, map[string]any{
		// ⚠️ On the status page rather than in a footer somewhere. This is the page
		// somebody opens when something looks wrong, and "which version is this?" is
		// the first question after "is it up?" — for us as much as for them.
		"version": Version,
		"stage":   Stage,
		"up":      up,
		// Null when there has never been a sample — a platform that has not
		// been measured, said plainly rather than painted green.
		"lastCheck": since,
		"uptime90d": ratio(okCount, checks),
		"hours":     hours,
		"days":      days,
		"now":       now,
	})
}

type hourPoint struct {
	Hour   string `json:"hour"`
	Checks int    `json:"checks"`
	OK     int    `json:"ok"`
	MaxMs  int64  `json:"maxMs"`
	Note   string `json:"note,omitempty"`
	// False when no sample exists for that hour — the platform was not running,
	// or was not yet measured. Drawn as a gap, never as a failure.
	Seen bool `json:"seen"`
}

type dayPoint struct {
	Day    string `json:"day"`
	Checks int    `json:"checks"`
	OK     int    `json:"ok"`
}

func currentState(hours []hourPoint) (bool, *string) {
	for i := len(hours) - 1; i >= 0; i-- {
		if hours[i].Checks == 0 {
			continue
		}
		h := hours[i].Hour
		return hours[i].OK == hours[i].Checks, &h
	}
	return false, nil
}

func ratio(ok, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(ok) / float64(total) * 100
}

func clamp(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
