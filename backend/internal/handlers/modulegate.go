package handlers

// Which admin screens belong to which module.
//
// ⚠️ **One table, matched on the request path, rather than a call at the top of
// forty handlers.** The forty-call version is the one that leaks: somebody adds
// a count, an export or a second list endpoint beside an existing one and
// forgets the line — and nothing fails, the screen simply works for a customer
// who did not buy it. The same argument as `tenantScope` in the console and
// `clampToAdmin` here: make the boundary a filter every request crosses, not a
// check each author has to remember.
//
// ⚠️ **Prefix-matched, longest first.** `/admin/reports/suppliers` is stock,
// not reports — it reads purchase invoices — and a shortest-first walk would
// hand it to whoever bought the reports module.

import (
	"net/http"
	"sort"
	"strings"

	"restaurant-backend/internal/models"
)

// gatedPrefix is one entry in the table.
type gatedPrefix struct {
	prefix string
	module string
}

// moduleRoutes is the whole map from URL to module.
//
// ⚠️ **What is deliberately absent is the more important half of this file.**
// Nothing here gates a fiscal receipt, an X/Z report, a cash shift, a role, a
// PIN, a void permission, the receipt printer, the cash drawer, the manual stop
// list or the data export. Those are not features to sell:
//
//   - the fiscal receipt is the law, and a plan that withholds it is a plan
//     that cannot legally be used;
//   - a till that cannot count its own drawer is not a till;
//   - selling *roles and PINs* would leave the cheap plans on one shared code,
//     which means every void carries the same name and attribution is gone —
//     we would be selling the customer's own theft risk back to them, and the
//     first theft would sound like our fault;
//   - the data is the restaurant's and must be able to leave with them.
//
// ⚠️ And nothing is gated by **counts of dishes or staff**, which is the
// tempting axis and the wrong one: it pushes a restaurant into merging dishes
// and sharing one waiter login, and both corrupt the data every report in this
// system is built on. Registers, branches and modules are clean axes because
// none of them can be gamed without giving something up.
var moduleRoutes = []gatedPrefix{
	// ---- Stock, cost price and everything downstream of a recipe ----
	{"/admin/ingredients", models.ModStock},
	{"/admin/warehouses", models.ModStock},
	{"/admin/purchases", models.ModStock},
	// Petty cash for the buying, so it lives and dies with the module the
	// buying is part of.
	{"/admin/advances", models.ModStock},
	{"/admin/suppliers", models.ModStock},
	{"/admin/writeoffs", models.ModStock},
	{"/admin/transfers", models.ModStock},
	{"/admin/stocktake", models.ModStock},
	{"/admin/stock/", models.ModStock},
	{"/admin/reports/stock", models.ModStock},
	{"/admin/reports/suppliers", models.ModStock},
	// ⚠️ The phone stocktake too. It is the same shelf being counted, reached
	// from a different screen — gating only the panel would leave the whole
	// module open to anybody who opened the staff app.
	{"/staff/warehouses", models.ModStock},
	{"/staff/stocktake", models.ModStock},
	// The buyer's phone reads and writes the same shelves as the panel's store,
	// so it crosses the same gate: a restaurant that has not bought the stock
	// module must not reach it through a different door.
	{"/staff/buy", models.ModStock},
	{"/staff/buy/orders", models.ModStock},

	// ---- The televisions on the wall ----
	//
	// ⚠️ **The panel only, and the screens themselves deliberately not.** This
	// gates adding, renaming and unpairing a television — the door. A set
	// already hanging in a dining room keeps playing whatever happens to the
	// subscription document, for the same reason the register cap is checked
	// when a link is issued and never again: the alternative is a room going
	// dark on a Friday evening because of a billing lookup.
	{"/admin/tv", models.ModTV},

	// ⚠️ **Somebody else's till is deliberately NOT here any more.**
	//
	// It was, behind Pro. Follow one customer through and the defect is the one
	// this file already describes two paragraphs down, wearing different
	// clothes: a restaurant paying per order for the website has no
	// subscription document, so it could connect its iiko — and the day it
	// bought a Start till, a document appeared with an empty module list and
	// the integration it had been running for months **stopped**. Paying us
	// more took away a thing that was working.
	//
	// And on its own terms it was the wrong axis. This is not a counter
	// feature: for a restaurant that never buys a Keel till, pushing its online
	// orders into the register it already runs is the entire reason the website
	// is worth having. Selling that back as an upgrade prices the product's own
	// value out of the plan most likely to need it.
	//
	// The module id stays in the model and on the tenants that hold it —
	// removing a stored string is how a paid module silently returns to
	// everybody or vanishes from everybody — but nothing gates on it.
}

// ⚠️ **Reports and the CRM are deliberately absent, and that is a pricing
// decision worth stating here.**
//
// They were in this table once, behind Standard and Pro. The trap only appears
// when you follow one customer through: a restaurant paying per order for the
// website has no subscription document, so the gate lets everything through and
// they use the analytics and the call centre for months. The day they buy a
// Start till a document appears with an empty module list — and buying from us
// *takes those screens away*. Paying more for less is the same defect as the
// old third-register cliff, wearing different clothes, and the customer would
// be right to read it as a bait and switch.
//
// So the axes are scale (registers, branches) and the modules that genuinely
// belong to the counter: stock and franchise. Analysis, the customer base and —
// since the note above — somebody else's till stay in the price for everybody,
// which is also what the public pricing page has always promised.

func init() {
	// Longest prefix first, so a more specific rule always wins over the
	// general one it sits inside.
	sort.SliceStable(moduleRoutes, func(i, j int) bool {
		return len(moduleRoutes[i].prefix) > len(moduleRoutes[j].prefix)
	})
}

// moduleFor names the module a path belongs to, or "" when it is not sold
// separately.
//
// ⚠️ **The mount prefix is stripped first, and forgetting that switched this
// whole file off.** The table is written as `/admin/…` while `r.URL.Path`
// carries `/api/v1/admin/…` — chi's `Route` does not rewrite the request URL.
// So nothing matched, `moduleFor` answered "" for every request, and the gate
// waved through every paid module on every install. It failed **open**, which
// is why it went unnoticed: the only symptom was a customer using a module
// nobody had sold them. The panel gate had the identical bug in the other
// direction, and it locked storekeepers out — see handlers/panelgate.go.
func moduleFor(path string) string {
	path = panelPath(path)
	for _, g := range moduleRoutes {
		if strings.HasPrefix(path, g.prefix) {
			return g.module
		}
	}
	return ""
}

// ModuleGate refuses a request for a module this restaurant has not bought.
//
// ⚠️ **Closed when nothing is sold, and this is the one place in this codebase
// where a missing document does NOT mean "carry on as before".**
//
// It used to be open. The argument was the one written down under "an empty
// mapProvider is 2GIS": a zero value must mean today's behaviour, or the day
// the feature ships every existing customer loses something. That argument is
// real and it is the reason this note is long — but it answers the wrong
// question here. `soldOut`, `hidePlan` and `mapProvider` default towards
// *working*; this field defaults towards *paying*. A missing subscription is
// not "we have not decided yet", it is "nobody has bought a counter", and the
// stockroom is sold with one.
//
// Read the other way round the old default was: every restaurant on the
// per-order website plan gets cost price, technical cards, stocktakes, supplier
// debt and the shopping list for nothing — which is the module the price list
// puts at 290 000 a month, given away to everybody who never asked for a till.
//
// ⚠️ **What this costs, stated plainly rather than discovered later.** On the
// day this shipped, every install without a subscription document lost the
// stock section. That is deliberate and it is the owner's decision; it is
// written here because it is exactly the kind of change that gets rediscovered
// as a bug report six months later by somebody reading the gate and finding no
// trace of the trade. The external till was in that list and has since been
// taken back out — see the note in moduleRoutes for why.
//
// ⚠️ **The narrow blast radius is what makes it safe**, and it is a property of
// moduleRoutes rather than of this function: nothing here touches a fiscal
// receipt, an X/Z report, a cash shift, a role, a PIN, the printer, the drawer,
// the manual stop list, the data export, the reports or the CRM. A restaurant
// that loses this gate's contents can still take money, close its day and get
// its data out.
func (h *Handler) ModuleGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mod := moduleFor(r.URL.Path)
		if mod == "" {
			next.ServeHTTP(w, r)
			return
		}
		// ⚠️ Straight to requireModule: `Subscription.Has` already answers
		// false for a nil or disabled document, so the decision lives in one
		// method rather than in a special case here that could drift from it.
		if !h.requireModule(w, r, mod) {
			return
		}
		next.ServeHTTP(w, r)
	})
}
