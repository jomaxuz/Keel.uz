package handlers

import (
	"testing"

	"restaurant-backend/internal/models"
)

// The half of the table that is a promise rather than a price.
//
// Each of these is either the law, or the thing that makes a till a till, or
// the customer's own data. Gating any of them does not produce a cheaper plan —
// it produces a plan that cannot legally be used, cannot count its drawer, or
// holds a restaurant's data hostage. Pinned here because the tempting edit is
// always to add one more line to moduleRoutes.
func TestNeverGated(t *testing.T) {
	never := []string{
		// The law.
		"/admin/fiscal", "/admin/settings/fiscal",
		// A till that cannot count itself is not a till.
		"/admin/cash", "/admin/cash/shift", "/admin/checks",
		// Attribution. Selling this leaves the cheap plans on one shared PIN,
		// and then every void carries the same name.
		"/admin/staff", "/admin/staff/pin", "/admin/admins", "/admin/logs",
		// The counter's own work.
		"/admin/orders", "/admin/menu", "/admin/categories",
		"/admin/stop-list", "/admin/printers",
		// Theirs, not ours.
		"/admin/export", "/admin/users", "/admin/restaurant",
		// ⚠️ Sold to everybody, and this is the entry that stops the trap
		// coming back: gating these means a restaurant that buys a till
		// *loses* the analytics and the call centre it already had.
		"/admin/reports/abc-xyz", "/admin/reports/finance",
		"/admin/campaigns", "/admin/calls", "/admin/segments", "/admin/rfm",
	}
	for _, path := range never {
		if mod := moduleFor(path); mod != "" {
			t.Errorf("%s is gated behind %q — see the comment in modulegate.go: "+
				"this one is not for sale", path, mod)
		}
	}
}

// The prefix walk has to be longest-first, and this is the case that proves it:
// the supplier report reads purchase invoices, so it belongs to stock, but it
// sits inside the reports prefix. A shortest-first walk hands it to the wrong
// module, and the symptom is a customer who bought the stock module and cannot
// open the one report it is for.
func TestLongestPrefixWins(t *testing.T) {
	cases := map[string]string{
		"/admin/reports/suppliers":  models.ModStock,
		"/admin/reports/stock":      models.ModStock,
		// ⚠️ Analysis and the customer base are NOT gated — see the note in
		// modulegate.go. Pinned as empty strings rather than left out, because
		// the tempting edit is to put them back.
		"/admin/reports/abc-xyz":    "",
		"/admin/reports/finance":    "",
		"/admin/campaigns/1/send":   "",
		"/admin/calls":              "",
		"/admin/ingredients":        models.ModStock,
		"/admin/ingredients/abc123": models.ModStock,
		"/staff/stocktake/sheet":    models.ModStock,
		"/admin/pos/mapping":        models.ModPOSIntegration,
		// Not sold separately, and inside no gated prefix.
		"/admin/orders/1/status": "",
	}
	for path, want := range cases {
		if got := moduleFor(path); got != want {
			t.Errorf("moduleFor(%q) = %q, want %q", path, got, want)
		}
	}
}

// A subscription that is absent or switched off closes the modules it sells.
//
// ⚠️ **This assertion was the opposite way round once**, and the flip is a
// pricing decision rather than a bug fix: the old rule handed the stock module
// — 290 000 a month on the price list — to every restaurant on the per-order
// website plan, because none of them has a subscription document. "No document"
// means nobody bought a counter, and the stockroom is sold with one.
//
// Sealed here because the two readings are one `!` apart, and reverting it
// would look like a tidy-up rather than like giving a paid module away.
func TestNoSubscriptionClosesSoldModules(t *testing.T) {
	var none *models.Subscription
	off := &models.Subscription{Enabled: false, Modules: []string{models.ModStock}}
	for _, s := range []*models.Subscription{none, off} {
		if s.Has(models.ModStock) {
			t.Error("a nil or disabled subscription must not grant a module")
		}
		// ⚠️ The switched-off case matters as much as the missing one: a
		// customer whose plan was turned off still carries the module list it
		// used to have, and reading that list without checking `Enabled` would
		// leave a cancelled subscription fully entitled.
		if s.Has(models.ModPOSIntegration) {
			t.Error("a nil or disabled subscription granted the till integration")
		}
	}
	on := &models.Subscription{Enabled: true, Modules: []string{models.ModPOSIntegration}}
	if !on.Has(models.ModPOSIntegration) || on.Has(models.ModStock) {
		t.Error("an enabled subscription must grant exactly what it lists")
	}
}

// ⚠️ **What the gate must never reach**, and this is the guard that makes
// closing-by-default safe rather than reckless.
//
// A restaurant with no plan now loses the stock section. It must not lose the
// ability to take money, close its day, or leave: a fiscal receipt is the law,
// a till that cannot count its drawer is not a till, and the data belongs to
// the customer. If somebody ever adds one of these prefixes to moduleRoutes,
// this fails before it reaches a restaurant.
func TestGateNeverTouchesTheUnsellable(t *testing.T) {
	for _, path := range []string{
		"/staff/checks/000000000000000000000000/close",
		"/staff/checks/000000000000000000000000/fiscal",
		"/admin/cash/shift",
		"/admin/roles",
		"/admin/staff/000000000000000000000000/pin",
		"/admin/stop-list",
		"/admin/branches/000000000000000000000000/sold-out",
		"/staff/stop-list",
		"/admin/export",
		"/admin/reports/sales",
		"/admin/campaigns",
		"/admin/calls",
	} {
		if mod := moduleFor(path); mod != "" {
			t.Errorf("%s is gated behind %q — it must be in the price for everybody", path, mod)
		}
	}
}
