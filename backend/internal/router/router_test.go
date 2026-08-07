package router

import (
	"net/http"
	"strings"
	"testing"

	"restaurant-backend/internal/config"
	"restaurant-backend/internal/handlers"

	"github.com/go-chi/chi/v5"
)

// The paths the site and the apps call, pinned.
//
// ⚠️ This exists because a handler can be written, compiled, reviewed and
// deployed **without ever being routed**. Go does not warn about a method
// nobody calls, so `TrackVisit` shipped complete and unreachable: the edit that
// was supposed to register it silently matched nothing, the build passed, and
// the only symptom was a 404 from a beacon that swallows its own errors by
// design.
//
// The same shape already cost this platform once on the control plane, where
// three callers guessed a prefix and all three were wrong. Both are the same
// lesson: the handler being right is not the same as the handler being wired,
// and only walking the real router can tell them apart.

func routes(t *testing.T) map[string]bool {
	t.Helper()
	r := New(&handlers.Handler{}, &config.Config{})
	found := map[string]bool{}
	err := chi.Walk(r.(chi.Routes),
		func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
			found[method+" "+strings.TrimSuffix(route, "/")] = true
			return nil
		})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	return found
}

// What the public site calls with no token at all. A 404 on any of these is
// invisible: the pages that use them all degrade quietly on purpose.
func TestPublicSitePaths(t *testing.T) {
	found := routes(t)
	for _, want := range []string{
		// The visitor beacon — the one that shipped unrouted.
		"POST /api/v1/visit",
		"GET /api/v1/restaurant",
		"GET /api/v1/menu",
		"GET /api/v1/categories",
		"POST /api/v1/orders",
		"POST /api/v1/delivery/quote",
		"GET /health",
	} {
		if !found[want] {
			t.Errorf("missing route %q", want)
		}
	}
}

// The beacon belongs under the API prefix the browser actually uses. Mounted
// at the root it answers on a path nothing calls — which is exactly what
// happened.
func TestVisitIsNotMountedAtTheRoot(t *testing.T) {
	if routes(t)["POST /visit"] {
		t.Error("/visit is mounted at the root; the site calls /api/v1/visit")
	}
}

// Every provider callback, pinned by its exact path.
//
// ⚠️ These are the only routes in the system whose URL is **typed into
// somebody else's cabinet**. A renamed or unregistered one cannot be found by
// testing our own site: nothing here calls them, the provider is the only
// caller, and it discovers the 404 with a guest's card already entered.
//
// ATMOS is the sharpest case. Its callback is not a notification but a
// permission — "the amount will only be deducted after receiving a successful
// status from the merchant" — so an unrouted endpoint does not lose a record,
// it declines every payment at the till while the site looks perfectly healthy.
func TestProviderCallbackPaths(t *testing.T) {
	found := routes(t)
	for _, want := range []string{
		"POST /api/v1/payments/payme",
		"POST /api/v1/payments/click/prepare",
		"POST /api/v1/payments/click/complete",
		"POST /api/v1/payments/uzum/check",
		"POST /api/v1/payments/uzum/create",
		"POST /api/v1/payments/uzum/confirm",
		"POST /api/v1/payments/uzum/reverse",
		"POST /api/v1/payments/uzum/status",
		"POST /api/v1/payments/atmos",
		// The link the guest is sent to pay with, and the list of methods the
		// checkout may offer. Both public by design.
		"GET /api/v1/orders/{number}/pay",
		"GET /api/v1/payment-methods",
	} {
		if !found[want] {
			t.Errorf("missing provider route %q", want)
		}
	}
}

// The reports, which are reached by a link rather than by the app's own code —
// so a renamed one fails as a download that does nothing, with no error
// anywhere and no failing request in the console.
func TestReportPaths(t *testing.T) {
	found := routes(t)
	for _, want := range []string{
		"GET /api/v1/admin/reports/abc-xyz",
		"GET /api/v1/admin/reports/finance",
		"GET /api/v1/admin/reports/cash",
		"GET /api/v1/admin/cash/shift",
		"POST /api/v1/admin/cash/shift/open",
		"POST /api/v1/admin/cash/shift/close",
		"POST /api/v1/admin/cash/entries",
	} {
		if !found[want] {
			t.Errorf("missing report route %q", want)
		}
	}
}
