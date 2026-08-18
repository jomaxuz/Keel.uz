package handlers

import (
	"strings"
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

// The quantity stepper on an open check.
//
// ⚠️ This is the only place a line's quantity goes **up**, and the three rules
// below are the ones a screen cannot be trusted with: a plus held down by a
// thumb, a minus walked to zero, and a request that carries a number but no
// note.
func TestApplyLineEdit(t *testing.T) {
	qty := func(n int) *int { return &n }
	text := func(s string) *string { return &s }

	t.Run("quantity alone keeps the guest's note", func(t *testing.T) {
		// ⚠️ The trap the pointer fields exist for: pressing "+" used to send a
		// request with no comment in it, and a plain string field would have
		// read that as "clear the comment" — wiping "piyozsiz" with nothing on
		// either screen saying so.
		line := models.OrderItem{Qty: 1, Comment: "piyozsiz"}
		if err := applyLineEdit(&line, lineEditRequest{Qty: qty(3)}); err != nil {
			t.Fatalf("refused a legal quantity: %v", err)
		}
		if line.Qty != 3 {
			t.Fatalf("qty = %d, want 3", line.Qty)
		}
		if line.Comment != "piyozsiz" {
			t.Fatalf("comment = %q, want it untouched", line.Comment)
		}
	})

	t.Run("a note alone keeps the quantity", func(t *testing.T) {
		line := models.OrderItem{Qty: 4}
		if err := applyLineEdit(&line, lineEditRequest{Comment: text("achchiq emas")}); err != nil {
			t.Fatalf("refused a note: %v", err)
		}
		if line.Qty != 4 || line.Comment != "achchiq emas" {
			t.Fatalf("got qty=%d comment=%q", line.Qty, line.Comment)
		}
	})

	t.Run("zero is refused rather than treated as a removal", func(t *testing.T) {
		// Taking a line off is a different act with a different record — an
		// unfired line is dropped, a fired one needs a cashier and a reason.
		// A stepper that voids at zero is how a till stops being able to say
		// where the food went.
		line := models.OrderItem{Qty: 1}
		if err := applyLineEdit(&line, lineEditRequest{Qty: qty(0)}); err == nil {
			t.Fatal("zero was accepted")
		}
		if line.Qty != 1 {
			t.Fatalf("the line changed anyway: qty = %d", line.Qty)
		}
	})

	t.Run("a held-down thumb cannot send a hundred to the kitchen", func(t *testing.T) {
		line := models.OrderItem{Qty: 1}
		if err := applyLineEdit(&line, lineEditRequest{Qty: qty(maxLineQty + 1)}); err == nil {
			t.Fatalf("accepted %d", maxLineQty+1)
		}
		if err := applyLineEdit(&line, lineEditRequest{Qty: qty(maxLineQty)}); err != nil {
			t.Fatalf("refused the limit itself: %v", err)
		}
	})

	t.Run("a long note is clamped, not refused", func(t *testing.T) {
		line := models.OrderItem{Qty: 1}
		long := strings.Repeat("a", 400)
		if err := applyLineEdit(&line, lineEditRequest{Comment: text(long)}); err != nil {
			t.Fatalf("refused a long note: %v", err)
		}
		if len(line.Comment) != 200 {
			t.Fatalf("comment length = %d, want 200", len(line.Comment))
		}
	})
}
