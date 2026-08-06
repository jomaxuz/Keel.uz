package handlers

import (
	"net/http"
	"strings"
	"testing"

	"keel-control/internal/config"

	"github.com/go-chi/chi/v5"
)

// The paths other services call, pinned.
//
// Three call sites guessed at these and all three guessed wrong: keel.uz asked
// for `/api/v1/partners` and `/api/v1/status`, and a tenant asked for
// `/api/v1/internal/domain`, while the routes actually sat under `/internal`
// — one of them double-prefixed by a copy-paste.
//
// What made it expensive is that **every caller swallows a 404 on purpose**.
// The partner strip renders empty when there are no partners; the status page
// says "control plane unreachable" when it cannot reach the control plane.
// Both are correct behaviour for their real failure, and both are
// indistinguishable from a mistyped path. So the landing page shipped looking
// exactly like an outage, and nothing anywhere logged a thing.
//
// A test that only checked the handlers would have passed. This walks the
// router the server actually serves.

func routes(t *testing.T) map[string]bool {
	t.Helper()
	r := Router(&Handler{}, &config.Config{})
	found := map[string]bool{}
	err := chi.Walk(r.(chi.Routes),
		func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
			// chi reports the pattern with a trailing slash on sub-routers.
			found[method+" "+strings.TrimSuffix(route, "/")] = true
			return nil
		})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	return found
}

func TestPublicPathsOtherServicesCall(t *testing.T) {
	found := routes(t)

	// keel.uz renders these server-side. A 404 here is invisible: the strip
	// goes empty and the status page reports an outage.
	for _, want := range []string{
		"GET /internal/partners",
		"GET /internal/status",
		// Caddy's TLS gate and the tenant host resolver — the two that were
		// already here and set the convention.
		"GET /internal/resolve",
		"GET /internal/tls-ask",
		// A tenant connecting its owner's own domain. Was registered as
		// "/internal/internal/domain": the group already supplies the prefix.
		"POST /internal/domain",
	} {
		if !found[want] {
			t.Errorf("missing route %q — a caller elsewhere will get 404 and swallow it", want)
		}
	}

	// The prefix belongs to the group, not to the pattern. Written out because
	// the doubled path looked right in a diff.
	if found["POST /internal/internal/domain"] {
		t.Error("domain route is double-prefixed")
	}
}

// These need a dashboard session, so keel.uz must never reach for them
// server-to-server: it holds no token, and the 401 would look like an outage
// in exactly the same way.
func TestDashboardPathsStayUnderApiV1(t *testing.T) {
	found := routes(t)
	for _, want := range []string{
		"GET /api/v1/stats",
		"POST /api/v1/stats/collect",
		"GET /api/v1/tenants",
		"GET /api/v1/tenants/{id}/live",
		"GET /api/v1/rollout",
		"POST /api/v1/rollout",
		"GET /api/v1/invoices",
		"POST /api/v1/tenants/{id}/invoices",
		"POST /api/v1/invoices/{id}/pay",
		"POST /api/v1/invoices/{id}/void",
	} {
		if !found[want] {
			t.Errorf("missing route %q", want)
		}
	}
}
