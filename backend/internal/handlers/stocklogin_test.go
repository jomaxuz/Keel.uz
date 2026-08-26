package handlers

import (
	"strings"
	"testing"
)

// ⚠️ **Deny-by-default, and this test is the reason it can stay that way.**
//
// The alternative design — let the storekeeper's token into the admin group and
// gate the sections it must not see — means every route added afterwards is
// visible to a storekeeper until somebody remembers otherwise. A list of what
// is *allowed* fails in the other direction: the cost of forgetting is a page
// that says forbidden, not a customer list on a stranger's screen.
func TestAStorekeeperReachesOnlyTheStore(t *testing.T) {
	for _, p := range []string{
		"/admin/ingredients", "/admin/warehouses", "/admin/purchases",
		"/admin/stocktake", "/admin/stock/balances", "/admin/reports/stock",
	} {
		if !stockAllowed(p) {
			t.Fatalf("%s is part of the store and was refused", p)
		}
	}
	for _, p := range []string{
		// Money and identity — the reasons this gate exists.
		"/admin/settings", "/admin/payments", "/admin/admins", "/admin/logs",
		"/admin/users", "/admin/campaigns", "/admin/rfm", "/admin/orders",
		"/admin/reports/sales", "/admin/reports/loss", "/admin/alerts/settings",
		"/admin/restaurant", "/admin/export", "/admin/cash",
	} {
		if stockAllowed(p) {
			t.Fatalf("%s is not the store and was allowed", p)
		}
	}
}

// ⚠️ **A prefix is a trap when two screens share a word.** `/admin/stock/`
// covers balances and movement; `/admin/stop-list` is a different screen with a
// similar name and a completely different audience.
func TestASimilarNameIsNotTheSameScreen(t *testing.T) {
	if stockAllowed("/admin/stop-list") {
		t.Fatal("the stop list was let in by the stock prefix")
	}
	if !stockAllowed("/admin/stock/balances") {
		t.Fatal("the store's own balances were refused")
	}
}

// ⚠️ **The scope is pinned to the storekeeper's own branch, and refused if it
// cannot be.**
//
// `clampToAdmin` returns the scope *unclamped* when it cannot find an admin
// account — harmless while every token in that group belonged to one, and not
// harmless the moment a staff account could reach it. A stock token with
// `?branchId=` would otherwise have read any store in the company.
func TestAStorekeeperCannotWidenTheirScope(t *testing.T) {
	src := readLossSource(t, "scope.go")
	i := strings.Index(src, "if claims.Role == RoleStock {")
	if i < 0 {
		t.Fatal("a stock token is no longer pinned to its own branch")
	}
	body := src[i : i+700]
	if !strings.Contains(body, "s.BranchID = s2.BranchID") {
		t.Fatal("the branch is not being taken from the employee")
	}
	// ⚠️ An error rather than a wider scope: failing open here is the whole
	// class of bug this guards against.
	if !strings.Contains(body, `errors.New("forbidden")`) {
		t.Fatal("a stock token with no branch falls through unclamped")
	}
}

// ⚠️ **The same refusal either way.** A staff account without the permission
// gets "invalid credentials" rather than "you lack permission": the login form
// is reachable by anybody, and telling a stranger which usernames exist and
// what they are missing is how a password gets guessed at leisure.
func TestARefusedStaffLoginSaysNothingExtra(t *testing.T) {
	src := readLossSource(t, "stocklogin.go")
	if strings.Contains(src, "http.StatusForbidden, \"ruxsat\"") {
		t.Fatal("the login form now tells a stranger which accounts exist")
	}
	if !strings.Contains(src, "if !s.Can(models.PermStock) {\n\t\treturn false") {
		t.Fatal("a staff account without the stock permission can sign in")
	}
}

// ⚠️ Tried only after the admin lookup fails, so an owner who happens to share
// a username with an employee still signs in as the owner.
func TestTheAdminAccountIsTriedFirst(t *testing.T) {
	src := readLossSource(t, "admin.go")
	admin := strings.Index(src, "h.Store.Admins.FindOne(r.Context(), bson.M{\"username\"")
	staff := strings.Index(src, "h.stockLogin(w, r, req)")
	if admin < 0 || staff < 0 || admin > staff {
		t.Fatal("a staff account can now shadow an admin one")
	}
}
