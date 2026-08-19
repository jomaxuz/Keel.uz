package handlers

import (
	"net/http"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
)

// The dashboard an admin arranged for themselves.
//
// The front page grew to about twenty figures, and roughly none of them are
// useful to everybody: an owner opens it for takings and the average bill, a
// branch manager opens it to see what is unconfirmed and who is on shift. A
// screen where the number you came for is the fourteenth one down is a screen
// people stop reading, and then stop trusting — they check the orders list
// instead and form their own impression of the month.
//
// ⚠️ **The rules are all zero-value rules**, because every account that exists
// today has no preferences and must keep the screen it has:
//
//   - an empty `hidden` shows everything — today's dashboard, exactly;
//   - an empty `order` is the default order;
//   - a tile added in a later version appears for everyone, because the stored
//     list names what is **off**, never what is on.
//
// The last one is why this is a hide-list rather than a show-list, and it is
// the difference between an update that adds a figure and an update that
// quietly withholds it from every existing account forever.

// dashboardTileIDs is every tile the front page can draw, in its default order.
//
// ⚠️ **The server owns this list, not the browser.** It is what an incoming
// preference is validated against: without it, `hidden` is an unbounded array
// of arbitrary strings that a client can grow until the admin document stops
// fitting anywhere useful. The panel has the same ids — it has to, to draw
// them — but a copy the client can edit is not a check.
var dashboardTileIDs = []string{
	// Orders
	"orders.total", "orders.delivered", "orders.delivery", "orders.pickup",
	"orders.dineIn", "orders.cancelled",
	// Money
	"money.revenue", "money.pending", "money.debt", "money.avgOrder",
	"money.deliveryFee",
	"money.cash",
	// People
	"people.usersTotal", "people.usersNew", "people.usersActive",
	"people.couriers", "people.admins", "people.withAddress",
	// Menu
	"menu.dishes", "menu.available", "menu.categories",
}

// knownTile reports whether an id is one the panel can actually draw.
func knownTile(id string) bool {
	for _, known := range dashboardTileIDs {
		if known == id {
			return true
		}
	}
	return false
}

// dashboardRequest is what the panel sends when the admin rearranges the page.
type dashboardRequest struct {
	Hidden []string `json:"hidden"`
	Order  []string `json:"order"`
}

// AdminSetMyDashboard stores this admin's arrangement of the front page.
func (h *Handler) AdminSetMyDashboard(w http.ResponseWriter, r *http.Request) {
	admin, err := h.adminUser(r)
	if err != nil {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	var req dashboardRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	prefs := models.DashboardPrefs{
		Hidden: cleanTileIDs(req.Hidden),
		Order:  cleanTileIDs(req.Order),
	}
	if _, err := h.Store.Admins.UpdateByID(r.Context(), admin.ID,
		bson.M{"$set": bson.M{"dashboard": prefs}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Not written to the audit log. That log answers "who changed the
	// restaurant", and rearranging one's own screen changes nothing anybody
	// else can see — filling the log with it would bury the entries that do.
	httpx.JSON(w, http.StatusOK, prefs)
}

// cleanTileIDs narrows a submitted list to ids the panel knows, without
// duplicates and in the order given.
//
// Unknown ids are dropped rather than rejected with a 400: a panel one version
// behind the server will send an id that no longer exists, and refusing the
// whole request would mean an admin whose browser had not reloaded could not
// save their layout at all — with an error naming a tile they have never heard
// of.
func cleanTileIDs(in []string) []string {
	// Bounded by the tile list itself: this is stored on the account document,
	// and a list that can only contain known ids can only be as long as there
	// are tiles.
	out := make([]string, 0, len(dashboardTileIDs))
	seen := map[string]bool{}
	for _, id := range in {
		if !knownTile(id) || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// resolveDashboard is the tile order an admin should actually see.
//
// Kept on the server and covered by a test because the two rules are easy to
// get backwards and neither failure looks like a bug: a wrong order looks like
// a preference that did not save, and a wrong hide looks like a tile the
// version removed.
func resolveDashboard(prefs models.DashboardPrefs) []string {
	hidden := map[string]bool{}
	for _, id := range prefs.Hidden {
		hidden[id] = true
	}

	// Named first, in the order given; everything else keeps its default
	// position behind them. An admin who dragged two tiles to the top is not
	// also deciding the order of the eighteen they never touched.
	ordered := make([]string, 0, len(dashboardTileIDs))
	placed := map[string]bool{}
	for _, id := range prefs.Order {
		if knownTile(id) && !placed[id] {
			placed[id] = true
			ordered = append(ordered, id)
		}
	}
	for _, id := range dashboardTileIDs {
		if !placed[id] {
			ordered = append(ordered, id)
		}
	}

	out := make([]string, 0, len(ordered))
	for _, id := range ordered {
		if !hidden[id] {
			out = append(out, id)
		}
	}
	return out
}

// AdminDashboardTiles is the catalogue plus this admin's arrangement.
//
// Both in one response because the settings screen needs both to draw a single
// checkbox list, and two requests would let it render a half-built list in
// between — the state in which every tile looks switched off.
func (h *Handler) AdminDashboardTiles(w http.ResponseWriter, r *http.Request) {
	admin, err := h.adminUser(r)
	if err != nil {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		// ⚠️ Built into a fresh slice rather than handed the package variable:
		// a nil or shared slice going out through JSON is how a caller ends up
		// mutating the catalogue (§ "Nil slice null bo'lib chiqadi" is the
		// other half of the same habit).
		"all":     append([]string(nil), dashboardTileIDs...),
		"hidden":  nonNil(admin.Dashboard.Hidden),
		"order":   nonNil(admin.Dashboard.Order),
		"visible": resolveDashboard(admin.Dashboard),
	})
}

// nonNil turns a nil slice into an empty one.
//
// ⚠️ Go marshals a nil slice as `null`, not `[]`, and the panel does
// `hidden.includes(...)` on it. This is the trap that has already bitten twice
// in this codebase; the fix lives in a function so the next edit cannot walk
// around it.
func nonNil(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}
