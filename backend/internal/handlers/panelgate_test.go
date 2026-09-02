package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"restaurant-backend/internal/auth"
	"restaurant-backend/internal/middleware"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ⚠️ **Deny-by-default, and this test is why it can stay that way.**
//
// An operator is hired to answer the phone. The panel behind that phone holds
// the customer base, every price, the payroll and the takings — so the list is
// what they may reach, not what they may not: the cost of forgetting a line is
// a screen that says forbidden, and the cost of forgetting the other kind is a
// company's numbers on a temp's screen.
func TestAnOperatorReachesOnlyTheThreeSections(t *testing.T) {
	for _, p := range []string{
		"/admin/orders", "/admin/orders/abc/status", "/admin/orders/abc/courier",
		"/admin/reservations", "/admin/reservations/abc/status",
		"/admin/calls", "/admin/calls/live", "/admin/calls/stats",
	} {
		if !operatorAllowed(http.MethodPost, p) {
			t.Errorf("%s is the operator's own work and was refused", p)
		}
	}
	for _, p := range []string{
		// Money, people and identity — the reasons this gate exists.
		"/admin/settings", "/admin/payments", "/admin/admins", "/admin/logs",
		"/admin/users", "/admin/campaigns", "/admin/rfm", "/admin/cash",
		"/admin/reports/sales", "/admin/payroll", "/admin/staff",
		"/admin/restaurant", "/admin/export", "/admin/stock/balances",
		"/admin/ingredients", "/admin/promotions", "/admin/feedback",
	} {
		if operatorAllowed(http.MethodGet, p) {
			t.Errorf("%s is not the operator's and was allowed", p)
		}
	}
}

// ⚠️ **The verb is half the rule.** The orders board draws a courier dropdown,
// so an operator has to read `/admin/couriers` — and the same prefix is how
// courier accounts are created and deleted. A prefix list with no method on it
// hands out the second with the first, and no screen would ever show it.
func TestAnOperatorReadsCouriersAndCannotChangeThem(t *testing.T) {
	if !operatorAllowed(http.MethodGet, "/admin/couriers") {
		t.Fatal("the courier dropdown on the orders board cannot load")
	}
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		if operatorAllowed(m, "/admin/couriers/abc") {
			t.Fatalf("an operator may %s a courier account", m)
		}
	}
	// The same shape for the menu: a phone order is built from it, and nothing
	// about answering a phone changes a price.
	if !operatorAllowed(http.MethodGet, "/admin/menu") {
		t.Fatal("a phone order cannot be built without the menu")
	}
	if operatorAllowed(http.MethodPut, "/admin/menu/abc") {
		t.Fatal("an operator may edit a dish")
	}
}

// ⚠️ **`/admin/me` is a prefix of `/admin/menu`**, and that is how the rule
// about somebody's own profile — the one line with no method restriction —
// quietly handed out write access to the price list. Caught here first; there
// is no screen on which it would have looked wrong.
func TestOneRuleDoesNotSwallowAnother(t *testing.T) {
	if !operatorAllowed(http.MethodPut, "/admin/me/dashboard") {
		t.Fatal("an operator cannot arrange their own screen")
	}
	if operatorAllowed(http.MethodDelete, "/admin/menu/abc") {
		t.Fatal("the profile rule is still swallowing the menu")
	}
	// The storekeeper's list has the same trap in it, sealed next door
	// (/admin/stock versus /admin/stop-list) — stated here so the two lists are
	// read as one family.
	if stockAllowed("/admin/stop-list") {
		t.Fatal("the stop list was let in by the stock prefix")
	}
}

// ⚠️ **The first screen a new account sees.** Panel accounts are created with
// `mustChangePassword`, and the panel shows nothing else until it is changed —
// so refusing this would lock every operator out on the day they were hired,
// with the password form itself as the wall.
func TestANewOperatorCanChangeTheirPassword(t *testing.T) {
	if !operatorAllowed(http.MethodPut, "/admin/credentials") {
		t.Fatal("a new operator cannot leave the password screen")
	}
	if !operatorAllowed(http.MethodGet, "/admin/me") {
		t.Fatal("the panel cannot ask who is signed in")
	}
}

// ⚠️ **`/admin/me` is the panel's first call after every sign-in**, and it read
// only the `admin_user` collection — so a storekeeper's token, which names a
// `staff` record, got a 404. The panel reads any failure there as an expired
// token: it cleared the token and returned to the login form. Correct password,
// then the login screen again, with nothing logged anywhere.
func TestAStorekeeperStaysSignedIn(t *testing.T) {
	src := readLossSource(t, "admin.go")
	i := strings.Index(src, "func (h *Handler) Me(")
	if i < 0 {
		t.Fatal("Me is gone")
	}
	if !strings.Contains(src[i:i+800], "claims.Role == RoleStock") {
		t.Fatal("/admin/me looks a storekeeper up in the admin accounts again")
	}
	// And the two answers come from one function, so the login and the session
	// check cannot describe the same person differently.
	stock := readLossSource(t, "stocklogin.go")
	if strings.Count(stock, "stockUserView(") < 3 {
		t.Fatal("the login response and /admin/me no longer share one shape")
	}
}

// The gate as it actually runs: a signed token through the same middleware the
// router mounts.
//
// ⚠️ **The list is not the guarantee — the middleware is.** A correct list
// wired in the wrong order, or onto the wrong group, refuses nothing at all,
// and every test above would still pass.
func TestTheGateRefusesARealOperatorToken(t *testing.T) {
	const secret = "test-secret"
	token, err := auth.Generate(secret, primitive.NewObjectID().Hex(), RoleOperator)
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{}
	r := chi.NewRouter()
	// ⚠️ **Mounted exactly as the router mounts it.** An earlier version of this
	// test called `/admin/orders` directly and passed while the gate was
	// refusing every storekeeper in production: chi's Route does not rewrite
	// `r.URL.Path`, so the middleware sees `/api/v1/admin/orders` and the lists
	// are written without the mount. The prefix *is* the thing under test.
	r.Route(APIBase, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole(secret, "owner", "manager", RoleStock, RoleOperator))
			r.Use(h.PanelGate)
			r.Handle("/admin/*", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
		})
	})

	call := func(method, path string) int {
		req := httptest.NewRequest(method, APIBase+path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec.Code
	}

	if got := call(http.MethodGet, "/admin/orders"); got != http.StatusOK {
		t.Errorf("the orders board answered %d — an operator cannot work", got)
	}
	if got := call(http.MethodGet, "/admin/users"); got != http.StatusForbidden {
		t.Errorf("the customer base answered %d, want 403", got)
	}
	if got := call(http.MethodDelete, "/admin/couriers/abc"); got != http.StatusForbidden {
		t.Errorf("deleting a courier answered %d, want 403", got)
	}

	// The same, for the role this gate was built for first. ⚠️ A storekeeper
	// reaching nothing at all is what the mount prefix caused, and `/admin/me`
	// is where it showed: the panel reads a failure there as an expired token
	// and returns to the login form.
	stock, err := auth.Generate(secret, primitive.NewObjectID().Hex(), RoleStock)
	if err != nil {
		t.Fatal(err)
	}
	token = stock
	if got := call(http.MethodGet, "/admin/me"); got != http.StatusOK {
		t.Errorf("a storekeeper cannot ask who they are: %d", got)
	}
	if got := call(http.MethodGet, "/admin/ingredients"); got != http.StatusOK {
		t.Errorf("the store answered %d for a storekeeper", got)
	}
	if got := call(http.MethodGet, "/admin/users"); got != http.StatusForbidden {
		t.Errorf("the customer base answered %d for a storekeeper", got)
	}
}
