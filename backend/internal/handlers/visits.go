package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// How many people came to the site, and how often.
//
// The restaurant's dashboard could say how many orders arrived and nothing at
// all about how many people looked — which is the difference between "nobody
// wants this" and "nobody can find it", and those need opposite responses.
//
// Deliberately the smallest thing that answers it:
//
//   - **One row per visitor per day** (`(date, vid)` unique), upserted. Unique
//     visitors is then a count of rows and page views a sum of a counter, from
//     one collection and one write per page.
//   - **The id is the browser's, generated in it, and never leaves it in the
//     clear** — it is hashed with the day before storing, so the row cannot be
//     used to follow somebody across days even by us. Counting visitors and
//     tracking people are different jobs and only the first one is wanted.
//   - **Crawlers are excluded for free**: this is called from JavaScript, and
//     Googlebot's page render does not run it. No user-agent list to maintain.
//   - **Rows expire.** A TTL index drops them after 100 days; the daily totals
//     that matter have long since been rolled up by the platform aggregate.

const visitTTLDays = 100

type visitRequest struct {
	// A random id the browser made and keeps. Never read back, never sent
	// anywhere else, and hashed with the day before it is stored.
	VID string `json:"vid"`
	// Which page, so an owner can see whether people leave at the menu.
	Path string `json:"path"`
}

// TrackVisit records one page view. Public and unauthenticated by nature.
//
// Answers 204 whatever happens: this is a beacon fired from a page the guest
// is already reading, and an error here must never be something they see or
// something the page waits on.
func (h *Handler) TrackVisit(w http.ResponseWriter, r *http.Request) {
	var req visitRequest
	_ = httpx.Decode(r, &req)

	vid := strings.TrimSpace(req.VID)
	if len(vid) > 100 {
		vid = vid[:100]
	}
	if vid == "" {
		// No id from the browser: fall back to something stable enough for one
		// day and useless after it — the address and the agent, hashed with
		// the date like everything else here.
		vid = clientIP(r) + "|" + r.UserAgent()
	}

	now := time.Now()
	day := now.Format("2006-01-02")
	// Hashed **with the day**, so the same browser is a different row
	// tomorrow. That is the property that makes this a counter rather than a
	// trail, and it is cheap to keep.
	sum := sha256.Sum256([]byte(day + "|" + vid))
	key := hex.EncodeToString(sum[:16])

	path := strings.TrimSpace(req.Path)
	if path == "" || !strings.HasPrefix(path, "/") {
		path = "/"
	}
	if len(path) > 200 {
		path = path[:200]
	}

	_, _ = h.Store.Visits.UpdateOne(r.Context(),
		bson.M{"date": day, "vid": key},
		bson.M{
			"$inc":         bson.M{"views": 1},
			"$setOnInsert": bson.M{"date": day, "vid": key, "firstAt": now, "firstPath": path},
			"$set":         bson.M{"lastAt": now},
		},
		options.Update().SetUpsert(true),
	)
	w.WriteHeader(http.StatusNoContent)
}

// clientIP reads the address Caddy forwarded, falling back to the socket.
func clientIP(r *http.Request) string {
	if f := r.Header.Get("X-Forwarded-For"); f != "" {
		if i := strings.IndexByte(f, ','); i > 0 {
			return strings.TrimSpace(f[:i])
		}
		return strings.TrimSpace(f)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
