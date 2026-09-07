package handlers

import (
	"strings"
	"testing"
	"time"
)

// ⚠️ **Monday is not Saturday, and an average cannot say so.** A restaurant
// that sells three times as much at the weekend gets, from a flat average, an
// order that is short every Friday and long every Tuesday — every week, in
// exactly the businesses where the weekend is most of the trade. The forecast
// walks the days it is actually covering.
func TestTheForecastWalksTheDaysItCoversRatherThanAveragingThem(t *testing.T) {
	var d demand
	// Quiet all week, busy on Saturday and Sunday.
	d.byWeekday[int(time.Saturday)] = 30
	d.byWeekday[int(time.Sunday)] = 30
	for _, wd := range []time.Weekday{
		time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday,
	} {
		d.byWeekday[int(wd)] = 5
	}

	// A Thursday. The next three days are Friday, Saturday, Sunday.
	thursday := time.Date(2026, 9, 3, 10, 0, 0, 0, time.Local)
	if got := d.forecast(thursday, 3); got != 65 {
		t.Errorf("Thursday's three-day cover = %v, want 65 (5+30+30)", got)
	}
	// A Sunday. The next three days are Monday, Tuesday, Wednesday.
	sunday := time.Date(2026, 9, 6, 10, 0, 0, 0, time.Local)
	if got := d.forecast(sunday, 3); got != 15 {
		t.Errorf("Sunday's three-day cover = %v, want 15 (5+5+5)", got)
	}
	// ⚠️ The flat average of that week is 12.9 a day — 38.6 over three days.
	// It under-orders the weekend by 40% and over-orders midweek by 150%, and
	// it is right on no day of the week at all.
}

// ⚠️ **A week's cover for something that lasts five days is a write-off with a
// delay.** The horizon is the delivery rhythm plus a margin, and then whatever
// the goods themselves allow — which is the difference between a pharmacy's
// yoghurt and its paracetamol, measured rather than asked about.
func TestTheHorizonIsLeadPlusCyclePlusMargin(t *testing.T) {
	// Delivered every eight days, and one arrived today: the order goes on the
	// next van (8 days), then has to last the cycle after it (8) plus the
	// margin (4) — twenty days.
	fresh := rhythm{every: 8, sinceLast: 0, deliveries: 6}
	if got := coverDays(fresh); got != 20 {
		t.Errorf("cover = %d, want 20 (8 until the next + 8 cycle + 4 margin)", got)
	}
	// Seven days since the last one: the next is due tomorrow, so only a day of
	// waiting is left to cover.
	due := rhythm{every: 8, sinceLast: 7, deliveries: 6}
	if got := coverDays(due); got != 13 {
		t.Errorf("cover = %d, want 13 (1 + 8 + 4)", got)
	}
	// ⚠️ A supplier three days late is due *today*, not minus three days.
	late := rhythm{every: 8, sinceLast: 11, deliveries: 6}
	if got := coverDays(late); got != 12 {
		t.Errorf("cover = %d, want 12 (0 + 8 + 4)", got)
	}
	// ⚠️ **And the whole thing is capped by how long the goods keep.** The same
	// arithmetic then reads correctly for a pharmacy's yoghurt and its
	// paracetamol, with neither being a special case in the code.
	perishable := rhythm{every: 8, sinceLast: 0, shelfLife: 5, deliveries: 6}
	if got := coverDays(perishable); got != 5 {
		t.Errorf("cover = %d, want 5 — the shelf life did not cap the horizon", got)
	}
	// Never zero: a shelf life under a day would otherwise order nothing at
	// all, which reads on the screen as "you have enough".
	if got := coverDays(rhythm{every: 2, shelfLife: 0.4, deliveries: 4}); got < 1 {
		t.Errorf("cover = %d — a short-lived line stopped being ordered", got)
	}
	// A margin, never more than a week of it.
	if got := coverDays(rhythm{every: 30, sinceLast: 30, deliveries: 4}); got != 37 {
		t.Errorf("cover = %d, want 37 (0 + 30 + 7)", got)
	}
}

// ⚠️ **Always up, and pieces are whole.** This screen exists to stop a shelf
// running out; rounding 71.9 kilos down to 71 saves nothing and brings the
// failure back. "Order 2.4 bottles" is a quantity nobody can hand over, so the
// person reading it rounds in whichever direction they feel like — which is the
// same as this screen not having decided.
func TestAnOrderIsRoundedUpAndPiecesAreWhole(t *testing.T) {
	if got := orderQty(2.4, "pcs"); got != 3 {
		t.Errorf("2.4 pieces = %v, want 3", got)
	}
	if got := orderQty(0.02, "kg"); got != 0.1 {
		t.Errorf("0.02 kg = %v, want 0.1 — it rounded away to nothing", got)
	}
	if got := orderQty(71.93, "kg"); got != 72 {
		t.Errorf("71.93 kg = %v, want 72", got)
	}
	if got := orderQty(3.14, "l"); got != 3.2 {
		t.Errorf("3.14 l = %v, want 3.2", got)
	}
	if got := orderQty(0, "kg"); got != 0 {
		t.Errorf("nothing needed came back as %v", got)
	}
}

// ⚠️ **One missed delivery over a holiday must not decide the year's buying.**
// A forty-day gap in a list of threes moves an average to eight and leaves the
// kitchen holding a week of fish; the median does not notice it.
func TestTheRhythmIsAMedianSoOneHolidayDoesNotDecideIt(t *testing.T) {
	if got := median([]float64{3, 3, 3, 3, 40}); got != 3 {
		t.Errorf("median = %v, want 3", got)
	}
	if got := median([]float64{2, 4}); got != 3 {
		t.Errorf("median of two = %v, want 3", got)
	}
	if got := median(nil); got != 0 {
		t.Errorf("median of nothing = %v, want 0", got)
	}
}

// ⚠️ **A day nobody traded is not a zero, and a day the shelf was empty is not
// one either.** Both are absences of measurement wearing the same clothes, and
// treating either as demand is wrong in a direction that compounds: a closed
// Monday would be ordered for, and an item that ran out would be ordered less —
// so it runs out again, and is ordered less again.
func TestAClosedDayAndAnEmptyShelfAreNotZeroDemand(t *testing.T) {
	src := readSource(t, "orderplan.go")
	fn := between(t, src, "func (h *Handler) demandProfile", "\n}\n")
	// The divisor is the days the branch actually traded.
	if !strings.Contains(fn, "trading := h.tradingDays(") {
		t.Fatal("the weekday average is divided by the calendar, not by the days the doors were open")
	}
	if !strings.Contains(fn, "divisor = a.sold[wd]") {
		t.Fatal("a day the shelf was empty still drags the average down")
	}
	// ⚠️ And only for a line that normally sells every day — a weekend-only
	// item keeps its honest zeros, or the profile this whole file is built on
	// stops meaning anything.
	if !strings.Contains(fn, "regularSellerShare") {
		t.Fatal("every quiet day is being read as a stock-out")
	}
	// ⚠️ And the weekday is Mongo's, taken in the restaurant's timezone: the
	// driver speaks UTC, so an evening's trade would otherwise be filed under
	// tomorrow — which here does not shift a total, it moves Saturday night's
	// cooking onto Sunday's profile.
	if !strings.Contains(fn, `"$dayOfWeek"`) ||
		!strings.Contains(fn, `"timezone": tz`) {
		t.Fatal("the weekday is taken without the restaurant's timezone")
	}
}

// ⚠️ **What is already on somebody's list is not bought twice.** The list is
// read in the morning and again after lunch; without this the second reading
// suggests everything the buyer is at that moment standing in a market holding.
func TestWhatIsAlreadyOnAListIsNotSuggestedAgain(t *testing.T) {
	src := readSource(t, "orderplan.go")
	fn := between(t, src, "func (h *Handler) requestedQty", "\n}\n")
	if !strings.Contains(fn, "models.ShoppingSent") {
		t.Fatal("finished shopping trips are being counted as outstanding requests")
	}
	list := between(t, readSource(t, "shoppinglist.go"),
		"func (h *Handler) shoppingList", "\n}\n")
	if !strings.Contains(list, "requested[in.ID]") {
		t.Fatal("the suggestion ignores what has already been asked for")
	}
}

// ⚠️ **A timezone Mongo will not accept makes a screen empty and says nothing.**
// With `TZ` unset the clock is still right and `time.Local.String()` degrades to
// the literal "Local", which an aggregation refuses — the caller then returns an
// empty map, the row count is zero and no error reaches any screen. Two features
// were found this way, so the fallback is a function with a test rather than a
// line of care in three pipelines.
func TestTheTimezoneHandedToMongoIsOneItAccepts(t *testing.T) {
	// A real zone travels through untouched.
	tashkent, err := time.LoadLocation("Asia/Tashkent")
	if err != nil {
		t.Skip("no tzdata in this environment")
	}
	old := time.Local
	defer func() { time.Local = old }()

	time.Local = tashkent
	if got := mongoTZ(); got != "Asia/Tashkent" {
		t.Errorf("mongoTZ() = %q, want the zone's own name", got)
	}
	time.Local = time.UTC
	if got := mongoTZ(); got != "UTC" {
		t.Errorf("mongoTZ() = %q, want UTC", got)
	}
	// And the case that broke: a location whose name is not a zone name.
	time.Local = time.FixedZone("Local", 5*3600)
	got := mongoTZ()
	if got == "Local" {
		t.Fatal("\"Local\" is being handed to Mongo — the pipeline is refused and the screen is empty")
	}
	if !strings.HasPrefix(got, "+") && !strings.HasPrefix(got, "-") {
		t.Errorf("fallback %q is neither a zone name nor a UTC offset", got)
	}
}

// And every pipeline that groups by day or weekday goes through it — the point
// of the function is that the next one cannot walk past it.
func TestNoPipelineHandsMongoTheRawLocationName(t *testing.T) {
	for _, file := range []string{
		"orderplan.go", "stockbalance.go", "insightgrowth.go",
	} {
		if strings.Contains(readSource(t, file), "time.Local.String()") {
			t.Errorf("%s hands Mongo time.Local.String() — see mongoTZ", file)
		}
	}
}
