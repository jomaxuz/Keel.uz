package handlers

import (
	"testing"
	"time"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// The permission boundary is the reason the two roles exist, so it is sealed
// here rather than trusted to the screens: a waiter's tablet hiding the payment
// button is a courtesy, and the rule has to hold when the request is made
// anyway.
func TestTillDenial(t *testing.T) {
	waiter := models.Staff{IsActive: true, CanWaiter: true}
	cashier := models.Staff{IsActive: true, CanCashier: true}
	plain := models.Staff{IsActive: true}
	fired := models.Staff{CanCashier: true} // dismissed, token still valid

	cases := []struct {
		name    string
		staff   models.Staff
		perm    string
		allowed bool
	}{
		{"waiter on the floor", waiter, models.PermWaiter, true},
		{"waiter at the till", waiter, models.PermCashier, false},
		// A cashier who could take money but not add a dish would send every
		// correction across the room, and the branch's fix for that is one
		// shared login — the thing these permissions exist to prevent.
		{"cashier implies waiter", cashier, models.PermWaiter, true},
		{"cashier at the till", cashier, models.PermCashier, true},
		{"staff account alone grants nothing", plain, models.PermWaiter, false},
		// ⚠️ The gap the kitchen screen had: a staff token outlives a shift by
		// days, so a dismissed employee still holds a working one tonight.
		{"dismissed employee is refused", fired, models.PermCashier, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			denied := tillDenial(c.staff, c.perm)
			if c.allowed && denied != "" {
				t.Fatalf("expected access, refused with %q", denied)
			}
			if !c.allowed && denied == "" {
				t.Fatal("expected refusal, got access")
			}
		})
	}
}

// A waiter refused the payment button must be told it is the payment they lack,
// not the login. The wrong message sends them to ask for the cashier's
// password, which defeats the split.
func TestCashierRefusalNamesThePermission(t *testing.T) {
	msg := tillDenial(models.Staff{IsActive: true, CanWaiter: true}, models.PermCashier)
	if msg == "" {
		t.Fatal("waiter was allowed a cashier action")
	}
	if msg == tillDenial(models.Staff{IsActive: true}, models.PermWaiter) {
		t.Fatal("cashier refusal reuses the floor-access message")
	}
}

func line(name string, price, qty int, fired bool, void *models.CheckLineVoid) models.OrderItem {
	it := models.OrderItem{LineID: name, Name: name, Price: price, Qty: qty, Void: void}
	if fired {
		at := time.Now()
		it.FiredAt = &at
	}
	return it
}

// The screen's total is computed from the live lines every time. A voided line
// left in a stored subtotal would overcharge the table, and the receipt on
// screen would still add up — because the screen does it the honest way.
func TestViewCheckIgnoresVoidedLines(t *testing.T) {
	o := &models.Order{
		Check: &models.OrderCheck{OpenedAt: time.Now().Add(-30 * time.Minute)},
		Items: []models.OrderItem{
			line("lagmon", 30000, 2, true, nil),
			line("choy", 5000, 1, false, nil),
			line("somsa", 12000, 3, true, &models.CheckLineVoid{Reason: "mehmon qaytardi"}),
		},
	}
	v := viewCheck(o, time.Now())
	if want := 65000; v.Subtotal != want {
		t.Fatalf("subtotal = %d, want %d (voided line must not count)", v.Subtotal, want)
	}
	if v.Total != 65000 {
		t.Fatalf("total = %d, want 65000", v.Total)
	}
	// The voided line stays visible: a running total that silently drops a line
	// is a total nobody at the counter can explain to the guest.
	if len(v.Lines) != 3 {
		t.Fatalf("lines = %d, want 3 (voided lines stay on screen)", len(v.Lines))
	}
	// Only the unfired, live line counts as pending.
	if v.Unfired != 1 {
		t.Fatalf("unfired = %d, want 1", v.Unfired)
	}
	if v.OpenMin < 29 || v.OpenMin > 31 {
		t.Fatalf("openMin = %d, want ~30", v.OpenMin)
	}
}

// A discount never drives the total below zero — the pricing pipeline's rule,
// applied at the counter.
func TestViewCheckTotalNeverNegative(t *testing.T) {
	o := &models.Order{
		Check:         &models.OrderCheck{OpenedAt: time.Now()},
		Items:         []models.OrderItem{line("choy", 5000, 1, true, nil)},
		DiscountTotal: 9000,
	}
	if got := viewCheck(o, time.Now()).Total; got != 0 {
		t.Fatalf("total = %d, want 0", got)
	}
}

// ⚠️ The rule that makes progressive service possible: the pass sees a check's
// fired lines and nothing else. Showing the rest would have cooks starting a
// main course the room has not asked for yet.
func TestKitchenItemsHidesUnfiredCheckLines(t *testing.T) {
	check := &models.Order{
		Check: &models.OrderCheck{OpenedAt: time.Now()},
		Items: []models.OrderItem{
			line("salat", 20000, 1, true, nil),
			line("kabob", 45000, 2, false, nil),
			line("nog'ora", 8000, 1, true, &models.CheckLineVoid{Reason: "adashib bosildi"}),
		},
	}
	got := kitchenItems(check)
	if len(got) != 1 || got[0].Name != "salat" {
		t.Fatalf("kitchen saw %v, want only the fired live line", names(got))
	}
}

// Everything that did not come from a till is unchanged: no line carries
// FiredAt, and all of them must still reach the pass.
func TestKitchenItemsUnchangedForWebsiteOrders(t *testing.T) {
	web := &models.Order{Items: []models.OrderItem{
		line("lagmon", 30000, 1, false, nil),
		line("choy", 5000, 2, false, nil),
	}}
	if got := kitchenItems(web); len(got) != 2 {
		t.Fatalf("website order lost lines: %v", names(got))
	}
	// Never nil — a nil slice marshals as `null` and the pass maps over it.
	if kitchenItems(&models.Order{}) == nil {
		t.Fatal("kitchenItems returned nil; the screen would crash on null")
	}
}

func names(items []models.OrderItem) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Name)
	}
	return out
}

// A check is open until it is closed, and "closed" is a timestamp rather than a
// status — the same shape as ReadyAt, for the same reason.
func TestCheckIsOpen(t *testing.T) {
	now := time.Now()
	if (&models.OrderCheck{OpenedAt: now}).IsOpen() != true {
		t.Fatal("a check with no closedAt must be open")
	}
	if (&models.OrderCheck{OpenedAt: now, ClosedAt: &now}).IsOpen() != false {
		t.Fatal("a closed check must not read as open")
	}
	var nilCheck *models.OrderCheck
	if nilCheck.IsOpen() {
		t.Fatal("an order that is not a till sale must not read as an open check")
	}
}

// Only cash may be settled at the counter without a second system involved;
// the website's bank redirects are not something a cashier can drive.
func TestTillMethods(t *testing.T) {
	for _, m := range []string{models.ProviderCash, "card", "transfer"} {
		if !tillMethods[m] {
			t.Fatalf("%s should be payable at the till", m)
		}
	}
	for _, m := range []string{"payme", "click", "uzum", ""} {
		if tillMethods[m] {
			t.Fatalf("%s must not be selectable at the till", m)
		}
	}
}

// A table is only a table if the floor plan still says so, and only if it is in
// service — seating a party at a table taken out of service is a decision the
// plan already made.
func TestBranchTable(t *testing.T) {
	b := &models.Branch{}
	b.Booking.Tables = []models.FloorTable{
		{ID: "t1", Number: "3", IsActive: true},
		{ID: "t2", Number: "4", IsActive: false},
	}
	if _, ok := branchTable(b, "t1"); !ok {
		t.Fatal("an active table must be found")
	}
	if _, ok := branchTable(b, "t2"); ok {
		t.Fatal("a table out of service must not be bookable")
	}
	if _, ok := branchTable(b, "nope"); ok {
		t.Fatal("an unknown id must not resolve")
	}
	if _, ok := branchTable(nil, "t1"); ok {
		t.Fatal("a nil branch must not resolve a table")
	}
}

// The receipt still needs a name when there is no guest record — and a counter
// sale has no table either.
func TestGuestLabel(t *testing.T) {
	if got := guestLabel("7"); got != "7-stol" {
		t.Fatalf("guestLabel(7) = %q", got)
	}
	if got := guestLabel(""); got == "" {
		t.Fatal("a counter sale must still be named, or the panel shows a blank row")
	}
}

// checkFilter carries the branch, always. Without it a waiter who types another
// branch's id into the URL edits that branch's table.
func TestCheckFilterIsBranchScoped(t *testing.T) {
	branch := primitive.NewObjectID()
	f := checkFilter(primitive.NewObjectID(), branch)
	if f["branchId"] != branch {
		t.Fatal("checkFilter must scope to the branch")
	}
	if _, ok := f["check"]; !ok {
		t.Fatal("checkFilter must not match orders that are not till checks")
	}
}
