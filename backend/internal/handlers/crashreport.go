package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"restaurant-backend/internal/httpx"
)

// Crash reports on their way to the platform.
//
// # Why this endpoint is unauthenticated
//
// ⚠️ **A report about a broken session cannot be required to have a working
// one.** The most valuable reports here are exactly the ones where something
// went wrong badly enough that the app is not in a normal state: a token that
// failed to refresh, a screen that crashed before login, a page that threw
// while rendering. Putting this behind auth would collect everything except the
// class it exists for.
//
// So it is open, and made safe by being *small and boring* instead:
//
//   - Nothing is read from the body except text, and none of it is trusted.
//   - A per-IP gate in front of the route (middleware.NewRateLimit), the same
//     one that stands in front of login and the SMS sender.
//   - Batches capped, bodies capped, and the platform caps again on arrival.
//
// The worst a hostile caller achieves is filling one restaurant's daily report
// quota with noise, which is visible on the very screen it lands on.
//
// # Why it goes through here rather than straight to the platform
//
// ⚠️ The report arrives at the console **identified**, because it is forwarded
// with the per-tenant credential this server already holds. An app posting
// directly would have to carry a platform credential — in a browser, on a
// courier's phone, inside a Windows installer — and whatever it claimed about
// which restaurant it was would have to be believed.
//
// ⚠️ **The consequence is stated rather than hidden: if this container is down,
// nothing is reported.** That failure is the one the console reads live from
// Docker (`attention: "down"`), and it is a different mechanism on purpose — a
// pipe cannot carry news of its own absence.

const (
	maxReportBody    = 1 << 20
	maxReportInBatch = 20
	// The forward runs on its own clock. ⚠️ Short: nothing downstream of this
	// is worth making an app that has already crashed wait for.
	reportForwardTimeout = 10 * time.Second
)

// PostReport takes what an app says went wrong and forwards it to the platform.
//
// ⚠️ **It answers 202 whatever happens next.** The caller is an app in the
// middle of failing; an error from the endpoint that collects errors is a
// second failure to handle, in the code least likely to be tested. Nothing here
// can make the app's situation worse, and nothing here reports its own trouble
// back into itself.
func (h *Handler) PostReport(w http.ResponseWriter, r *http.Request) {
	// Accepted before anything else, so even the refusals are quiet.
	defer httpx.JSON(w, http.StatusAccepted, map[string]any{"ok": true})

	var body struct {
		Reports []map[string]any `json:"reports"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, maxReportBody)).
		Decode(&body) != nil || len(body.Reports) == 0 {
		return
	}
	if len(body.Reports) > maxReportInBatch {
		body.Reports = body.Reports[:maxReportInBatch]
	}

	// ⚠️ **The version is the app's to state, and nothing is stamped here.**
	// This server has no version constant of its own — only the control plane
	// does (control/internal/handlers/version.go) — and filling the field with
	// the platform's would name the wrong binary on every report: the question
	// asked of these is "did the deploy fix it", and the deploy in question is
	// the app's. An empty version is a gap somebody can see; a confident wrong
	// one is not.

	// ⚠️ **Detached from the request.** The app is not waiting for this and must
	// not be: a slow platform would turn one crashed screen into one crashed
	// screen that also hangs. `context.WithoutCancel` is the point — the
	// request's context dies the moment the response above is written.
	ctx, cancel := context.WithTimeout(
		context.WithoutCancel(r.Context()), reportForwardTimeout)
	go func() {
		defer cancel()
		_, _ = h.callControlPath(ctx, "/internal/report",
			map[string]any{"reports": body.Reports})
	}()
}
