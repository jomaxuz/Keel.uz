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
func TestTheShelfLifeCapsTheHorizon(t *testing.T) {
	// Delivered every eight days: eight plus a four-day margin is twelve.
	weekly := rhythm{every: 8, deliveries: 6}
	if got := coverDays(weekly); got != 12 {
		t.Errorf("cover = %d, want 12", got)
	}
	// The same rhythm for something that keeps five days.
	perishable := rhythm{every: 8, shelfLife: 5, deliveries: 6}
	if got := coverDays(perishable); got != 5 {
		t.Errorf("cover = %d, want 5 — the shelf life did not cap the horizon", got)
	}
	// ⚠️ And never zero: a shelf life shorter than a day would otherwise order
	// nothing at all, which reads on the screen as "you have enough".
	if got := coverDays(rhythm{every: 2, shelfLife: 0.4, deliveries: 4}); got < 1 {
		t.Errorf("cover = %d — a short-lived line stopped being ordered", got)
	}
	// A margin, never more than a week of it: a fortnightly delivery does not
	// need a fortnight of slack.
	if got := coverDays(rhythm{every: 30, deliveries: 4}); got != 37 {
		t.Errorf("cover = %d, want 37 (30 + 7)", got)
	}
	// Nothing measured: a week, which is what somebody with no delivery
	// history in front of them would say. ⚠️ It reaches no row on its own —
	// the list needs two deliveries before the forecast speaks at all.
	if got := coverDays(rhythm{}); got != coverWhenUnknown+4 {
		t.Errorf("cover with no rhythm = %d", got)
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

// ⚠️ **A weekday's demand is divided by the weekdays that happened, not by the
// ones that sold.** A restaurant shut on Mondays sells nothing on eight of the
// window's fifty-six days; dividing by "the Mondays it sold something" would
// report its Monday demand as if it opened — and Monday is the day the order
// would arrive for. A zero day is a real zero.
func TestAClosedDayCountsAsZeroNotAsAbsent(t *testing.T) {
	src := readSource(t, "orderplan.go")
	fn := between(t, src, "func (h *Handler) demandProfile", "\n}\n")
	if !strings.Contains(fn, "occurrences[wd]") {
		t.Fatal("the weekday average is not divided by the calendar")
	}
	if strings.Contains(fn, "row.Qty / float64(row.Days)") {
		t.Fatal("the average is divided by the days that sold — a closed Monday reads as a busy one")
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
