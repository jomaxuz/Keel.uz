package handlers

// ---- Panel roles that only reach part of the panel ----
//
// Two of them now, and they arrived from opposite directions:
//
//   - **stock** — a `staff` account signed in through the panel's form, because
//     the counting and the recipes they are responsible for live here (see
//     stocklogin.go).
//   - **operator** — a real `admin_user`, hired to answer the phone. Orders,
//     reservations and the call desk; nothing else.
//
// ⚠️ **Deny-by-default, and it is the whole design.** The alternative — let the
// token into the admin group and gate the sections it must not see — means every
// route added afterwards is reachable by these roles until somebody remembers
// otherwise. Here a path that is not on a list is refused, so the failure mode
// of a future edit is an operator seeing too little, which is reported the same
// day and breaks nothing.
//
// ⚠️ **The method matters, not only the path.** The orders board offers a
// courier dropdown, so an operator has to be able to *read* `/admin/couriers` —
// and the same prefix carries the POST and DELETE that create and remove courier
// accounts. A prefix list with no verb on it hands out the second with the
// first, and nothing on any screen would ever show it.

import (
	"net/http"
	"strings"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
)

// APIBase is where the whole API is mounted. The router takes it from here.
//
// ⚠️ **Because the gate below compares paths, and `r.URL.Path` keeps the mount
// prefix.** chi's `Route` does not rewrite the request URL, so the path this
// middleware sees is `/api/v1/admin/orders` and every rule in the lists below
// is written as `/admin/...`. Nothing matched — which for a deny-by-default
// gate means the storekeeper's token was refused **the entire panel**,
// including `/admin/me`. The panel reads a failure there as an expired token,
// so a storekeeper signed in correctly and was returned to the login form,
// forever, with no error anywhere. It failed closed, which is the only reason
// this was a lockout and not a leak.
const APIBase = "/api/v1"

// panelPath is the path as the allow-lists are written: without the mount.
func panelPath(p string) string {
	if rest, ok := strings.CutPrefix(p, APIBase); ok && rest != "" {
		return rest
	}
	return p
}

// RoleOperator is a panel account that only works the phone.
//
// ⚠️ **A role on the account rather than a flag on a manager.** Every
// `requireOwner` and every owner-only group in the router refuses it without
// being edited — the gate is the router's existing shape rather than a list
// somebody has to keep up to date.
const RoleOperator = "operator"

// pathRule is one line of an allow-list: a path prefix, and which methods of it
// are allowed. No methods means every method.
//
// ⚠️ **`exact` exists because `/admin/me` is a prefix of `/admin/menu`.** The
// operator may do anything under their own profile and may only read the menu —
// and as a prefix the first rule swallowed the second, so a PUT to
// `/admin/menu/{id}` was allowed by the line about the person's own name. A
// test caught it; nothing on any screen would have. Same family as the
// `/admin/stock` versus `/admin/stop-list` trap next door.
type pathRule struct {
	prefix  string
	methods []string
	exact   bool
}

func (p pathRule) allows(method, path string) bool {
	if p.exact {
		if path != p.prefix {
			return false
		}
	} else if !strings.HasPrefix(path, p.prefix) {
		return false
	}
	if len(p.methods) == 0 {
		return true
	}
	for _, m := range p.methods {
		if m == method {
			return true
		}
	}
	return false
}

var read = []string{http.MethodGet}

// operatorPaths is everything a call-centre operator's token may reach.
//
// ⚠️ Each line was chosen by opening the three screens and reading what they
// call, not by pattern. The screens are the orders board, the reservations list
// and the call desk; the rest of this list is what those three need in order to
// draw themselves — a courier's name, a branch's name, the menu the phone order
// is built from.
var operatorPaths = []pathRule{
	// The three sections themselves. Full access: taking an order over the
	// phone, moving it through the kitchen, cancelling it with a reason and
	// calling an external delivery service are the job.
	{prefix: "/admin/orders"},
	{prefix: "/admin/reservations"},
	{prefix: "/admin/calls"},
	// Who is calling, and what they ordered last time. The reason a call desk
	// is worth opening at all.
	{prefix: "/admin/lookup", methods: read},
	// ⚠️ Read-only, and this is the line that method rules exist for: the
	// courier dropdown on the orders board is a GET, and the same prefix is
	// how courier accounts are created and deleted.
	{prefix: "/admin/couriers", methods: read},
	// A phone order is built from the menu.
	{prefix: "/admin/menu", methods: read},
	{prefix: "/admin/categories", methods: read},
	// The external delivery services the orders board can hand a delivery to.
	{prefix: "/admin/delivery-providers", methods: read},
	// Which branch an order belongs to, and what it is called.
	{prefix: "/admin/branches", methods: read},
	{prefix: "/admin/brands", methods: read},
	// The panel's own chrome: who am I, what did this restaurant buy, and the
	// bell that says a new order arrived.
	// ⚠️ Their own profile, and **not** the menu: as a plain prefix this line
	// also matches `/admin/menu`. See pathRule.exact.
	{prefix: "/admin/me", exact: true},
	{prefix: "/admin/me/"},
	{prefix: "/admin/subscription", methods: read},
	{prefix: "/admin/alerts", methods: read},
	// ⚠️ **Their own password, and it is not optional.** A new panel account is
	// created with `mustChangePassword`, and the panel refuses to show anything
	// else until it is changed — without this the operator would be locked on
	// the one screen they are sent to.
	{prefix: "/admin/credentials", methods: []string{http.MethodPut}},
}

func operatorAllowed(method, path string) bool {
	for _, rule := range operatorPaths {
		if rule.allows(method, path) {
			return true
		}
	}
	return false
}

// PanelGate refuses a limited role anything outside its own list.
//
// ⚠️ **Applied to the whole admin group**, so it is one place rather than a
// check per handler — and a handler added tomorrow is covered by having been
// added, not by somebody remembering.
func (h *Handler) PanelGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := middleware.ClaimsFrom(r.Context())
		if claims != nil {
			path := panelPath(r.URL.Path)
			switch claims.Role {
			case RoleStock:
				if !stockAllowed(path) {
					httpx.Error(w, http.StatusForbidden, "forbidden")
					return
				}
			case RoleOperator:
				if !operatorAllowed(r.Method, path) {
					httpx.Error(w, http.StatusForbidden, "forbidden")
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
