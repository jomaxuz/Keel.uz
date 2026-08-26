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
	v := viewCheck(o, time.Now(), primitive.NilObjectID)
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
	if got := viewCheck(o, time.Now(), primitive.NilObjectID).Total; got != 0 {
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

// What a counter may be paid with.
//
// ⚠️ The provider rails were once refused here outright, on the reasoning that
// a bank redirect is a checkout flow a cashier cannot drive. That reasoning was
// right about the redirect and wrong about the counter: the guest drives it, on
// their own phone, from a QR on the till screen. What must stay true is the
// half that mattered — a check paid this way closes on the provider's word and
// never on the cashier's (TestAnOnlinePaymentClosesOnlyOnTheProvidersWord).
func TestTillMethods(t *testing.T) {
	for _, m := range []string{
		models.ProviderCash, "card", "transfer", models.MethodDebt,
		models.ProviderPayme, models.ProviderClick, models.ProviderUzum,
	} {
		if !tillMethods[m] {
			t.Fatalf("%s should be payable at the till", m)
		}
	}
	// ⚠️ ATMOS stays out: its link is fetched rather than built, so it can fail
	// for network reasons — and a QR that sometimes does not appear is worse at
	// a counter than one that was never offered.
	for _, m := range []string{models.ProviderAtmos, "", "points"} {
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

// Tapping the same tile twice.
//
// ⚠️ **A till is used by tapping**, and four coffees is the tile pressed four
// times. Stacking four identical rows made a check nobody could read back to a
// guest, and the only way to correct a miscount was removing rows one at a
// time. These are the rules that decide when two taps are one line.
func TestMergeableLine(t *testing.T) {
	dish := primitive.NewObjectID()
	other := primitive.NewObjectID()
	fired := time.Now()
	big := []models.OrderItemOption{{Name: "Hajm", Choice: "Katta"}}
	small := []models.OrderItemOption{{Name: "Hajm", Choice: "Kichik"}}

	line := func(f func(*models.OrderItem)) models.OrderItem {
		it := models.OrderItem{MenuItemID: dish, Qty: 1, LineID: "a1"}
		f(&it)
		return it
	}

	cases := []struct {
		name  string
		items []models.OrderItem
		add   models.OrderItem
		want  int
	}{
		{
			"the same dish again lands on the same line",
			[]models.OrderItem{line(func(*models.OrderItem) {})},
			models.OrderItem{MenuItemID: dish, Qty: 1},
			0,
		},
		{
			// The ticket at the pass names a quantity. Growing it quietly would
			// leave the paper and the screen disagreeing about the same dish —
			// and "two are cooking, one more has been asked for" is the honest
			// reading anyway.
			"a fired line is left alone",
			[]models.OrderItem{line(func(it *models.OrderItem) { it.FiredAt = &fired })},
			models.OrderItem{MenuItemID: dish, Qty: 1},
			-1,
		},
		{
			// A voided line counts for nothing; adding to it would resurrect
			// food somebody wrote off, with the reason still attached.
			"a voided line is left alone",
			[]models.OrderItem{line(func(it *models.OrderItem) {
				it.Void = &models.CheckLineVoid{Reason: "xato"}
			})},
			models.OrderItem{MenuItemID: dish, Qty: 1},
			-1,
		},
		{
			"a different dish opens its own line",
			[]models.OrderItem{line(func(*models.OrderItem) {})},
			models.OrderItem{MenuItemID: other, Qty: 1},
			-1,
		},
		{
			// Different food, not more of the same.
			"a different portion opens its own line",
			[]models.OrderItem{line(func(it *models.OrderItem) { it.Options = big })},
			models.OrderItem{MenuItemID: dish, Qty: 1, Options: small},
			-1,
		},
		{
			"the same portion lands on the same line",
			[]models.OrderItem{line(func(it *models.OrderItem) { it.Options = big })},
			models.OrderItem{MenuItemID: dish, Qty: 1, Options: big},
			0,
		},
		{
			// Merging a plain one into a line that says "piyozsiz" sends the
			// wrong instruction to the kitchen for both of them.
			"a note keeps a line to itself",
			[]models.OrderItem{line(func(it *models.OrderItem) { it.Comment = "piyozsiz" })},
			models.OrderItem{MenuItemID: dish, Qty: 1},
			-1,
		},
		{
			"the second unfired line is found, not the fired one",
			[]models.OrderItem{
				line(func(it *models.OrderItem) { it.FiredAt = &fired }),
				line(func(it *models.OrderItem) { it.LineID = "a2" }),
			},
			models.OrderItem{MenuItemID: dish, Qty: 1},
			1,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if at := mergeableLine(c.items, c.add); at != c.want {
				t.Fatalf("mergeableLine = %d, want %d", at, c.want)
			}
		})
	}
}

// ⚠️ The choices arrive in whatever order the dialog listed the groups in, so a
// screen that renders "Hajm" before "Qo'shimcha" one day and after it the next
// would stop merging without anybody changing anything.
func TestSameOptionsIgnoresOrder(t *testing.T) {
	a := []models.OrderItemOption{
		{Name: "Hajm", Choice: "Katta"},
		{Name: "Qo'shimcha", Choice: "Pishloq"},
	}
	b := []models.OrderItemOption{
		{Name: "Qo'shimcha", Choice: "Pishloq"},
		{Name: "Hajm", Choice: "Katta"},
	}
	if !sameOptions(a, b) {
		t.Fatal("the same two choices in the other order read as different food")
	}
	if sameOptions(a, a[:1]) {
		t.Fatal("a subset matched")
	}
	// Two of the same choice is not one of it.
	twice := []models.OrderItemOption{a[0], a[0]}
	if sameOptions(twice, a) {
		t.Fatal("a repeated choice matched a different pair")
	}
}

// Splitting the bill, and sending a meal in courses.
//
// ⚠️ **The two rules pull in opposite directions and that is deliberate.** What
// the kitchen has to make is frozen the moment the ticket prints; who is paying
// for it is decided when the plates are cleared. A single "fired lines cannot be
// edited" rule would have made splitting a bill impossible at the only moment
// anybody ever asks for it.
func TestLineEditAfterFiring(t *testing.T) {
	guest := func(n int) *int { return &n }
	text := func(s string) *string { return &s }

	cases := []struct {
		name  string
		req   lineEditRequest
		cooks bool
	}{
		{"quantity is the kitchen's", lineEditRequest{Qty: guest(2)}, true},
		{"a note is the kitchen's", lineEditRequest{Comment: text("piyozsiz")}, true},
		{"a course is the kitchen's", lineEditRequest{Course: guest(2)}, true},
		{"who pays is not", lineEditRequest{Guest: guest(2)}, false},
		{"nothing at all is not", lineEditRequest{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := cooksAffected(c.req); got != c.cooks {
				t.Fatalf("cooksAffected = %v, want %v", got, c.cooks)
			}
		})
	}
}

func TestApplyLineEditGuestAndCourse(t *testing.T) {
	n := func(v int) *int { return &v }

	t.Run("a guest and a course are recorded", func(t *testing.T) {
		line := models.OrderItem{Qty: 1}
		if err := applyLineEdit(&line, lineEditRequest{Guest: n(3), Course: n(2)}); err != nil {
			t.Fatalf("refused: %v", err)
		}
		if line.Guest != 3 || line.Course != 2 {
			t.Fatalf("guest=%d course=%d", line.Guest, line.Course)
		}
	})

	t.Run("zero puts a line back on the shared bill", func(t *testing.T) {
		// ⚠️ Zero is a legal value, not a missing one: it is how a waiter undoes
		// a split, and how every check written before this existed reads.
		line := models.OrderItem{Qty: 1, Guest: 2, Course: 1}
		if err := applyLineEdit(&line, lineEditRequest{Guest: n(0), Course: n(0)}); err != nil {
			t.Fatalf("refused zero: %v", err)
		}
		if line.Guest != 0 || line.Course != 0 {
			t.Fatalf("guest=%d course=%d", line.Guest, line.Course)
		}
	})

	t.Run("a thumb cannot split a bill two hundred ways", func(t *testing.T) {
		line := models.OrderItem{Qty: 1}
		if err := applyLineEdit(&line, lineEditRequest{Guest: n(maxGuests + 1)}); err == nil {
			t.Fatal("accepted an impossible guest")
		}
		if err := applyLineEdit(&line, lineEditRequest{Course: n(maxCourse + 1)}); err == nil {
			t.Fatal("accepted an impossible course")
		}
		if line.Guest != 0 || line.Course != 0 {
			t.Fatalf("the line changed anyway: guest=%d course=%d", line.Guest, line.Course)
		}
	})
}

// Two guests ordering the same dish are two lines, and so are two courses of
// it. This is the case the line model was always worried about: merging them
// hands one guest a bill for both, and sends a dessert out with the starters.
func TestMergeKeepsGuestsAndCoursesApart(t *testing.T) {
	dish := primitive.NewObjectID()
	base := models.OrderItem{MenuItemID: dish, Qty: 1, LineID: "a1"}

	same := base
	if mergeableLine([]models.OrderItem{base}, same) != 0 {
		t.Fatal("the same line for the same guest did not merge")
	}

	otherGuest := base
	otherGuest.Guest = 2
	if mergeableLine([]models.OrderItem{base}, otherGuest) != -1 {
		t.Fatal("a second guest's dish merged into the first guest's line")
	}

	otherCourse := base
	otherCourse.Course = 2
	if mergeableLine([]models.OrderItem{base}, otherCourse) != -1 {
		t.Fatal("a second course merged into the first")
	}
}

// The plan's slices leave the server as arrays, never as null.
//
// ⚠️ **Third time on this one struct.** A nil slice marshals to JSON `null`,
// and the panel's editors filter and map straight over what they are handed —
// so a branch whose room was never drawn, or never split into zones, took the
// whole settings page down with "something went wrong in the kitchen". Not a
// section, not a field: the page. Sealed here because the fix has to hold on
// every path that hands a plan out, and there are four of them.
func TestBookingSlicesAreNeverNull(t *testing.T) {
	got := bookingSlices(models.BookingSettings{})
	if got.Tables == nil || got.Shapes == nil || got.Zones == nil {
		t.Fatalf("a nil slice survived: tables=%v shapes=%v zones=%v",
			got.Tables == nil, got.Shapes == nil, got.Zones == nil)
	}

	// ⚠️ And it does **not** fill in defaults. The settings page saves what it
	// was given straight back, so a slot length invented here would be written
	// into every branch somebody merely opened — emptiness is the true shape of
	// the same fact, a plan size is a guess about a room.
	if got.SlotMinutes != 0 || got.Width != 0 || got.MaxGuests != 0 {
		t.Fatalf("defaults leaked into the edit path: %+v", got)
	}

	// The drawing itself is left alone.
	drawn := bookingSlices(models.BookingSettings{
		Tables: []models.FloorTable{{Number: "7"}},
	})
	if len(drawn.Tables) != 1 || drawn.Tables[0].Number != "7" {
		t.Fatalf("the room was rewritten: %+v", drawn.Tables)
	}
}
