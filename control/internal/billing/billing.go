// Package billing decides which window a tenant is being charged for.
//
// The period belongs to the customer, not to the calendar: a restaurant that
// subscribed on the 17th of August is billed 17 August → 16 September, and the
// next one starts on the 17th of September. Anchoring on the calendar month
// instead would hand every customer who signed up late in the month a nearly
// free first period, and every customer who signed up early one they did not
// use.
//
// Nothing here touches the database or the tenant model on purpose. It is
// arithmetic over dates, which is exactly the kind of code that is wrong in one
// month of the year and right in the other eleven — so it is kept where it can
// be tested against those months directly.
package billing

import "time"

// AddMonths advances an anchor date by n whole months.
//
// The one rule worth stating: when the anchor day does not exist in the target
// month it is **clamped to the last day of that month**, and the clamp is
// always measured from the original anchor, never from the previous clamped
// result. A subscription anchored on the 31st therefore reads
// 31 Jan → 28 Feb → 31 Mar, not 31 Jan → 28 Feb → 28 Mar. Carrying the clamp
// forward would walk a customer's billing day backwards a little every short
// month, and nobody would find out until the day had drifted by a week.
func AddMonths(anchor time.Time, n int) time.Time {
	y, m, d := anchor.Date()
	// time.Date normalizes an out-of-range month, so n may be any integer,
	// including a negative one.
	first := time.Date(y, m+time.Month(n), 1, 0, 0, 0, 0, anchor.Location())
	if dim := daysIn(first.Year(), first.Month()); d > dim {
		d = dim
	}
	return time.Date(first.Year(), first.Month(), d, 0, 0, 0, 0, anchor.Location())
}

// Cycle returns the billing window [from, to) that contains now, counted in
// whole months from anchor.
//
// A now that falls before the anchor — an operator typing next month's date by
// mistake, or a clock that slipped — returns the *first* period rather than a
// window in the past. Billing a customer for time before they subscribed is
// worse than billing them a few days late.
func Cycle(anchor, now time.Time) (from, to time.Time) {
	a := startOfDay(anchor)
	if now.Before(a) {
		return a, AddMonths(a, 1)
	}

	// A first guess from the plain month difference, then corrected. The guess
	// is off by at most one because the day-of-month clamp can only move a
	// boundary within its own month.
	n := (now.Year()-a.Year())*12 + int(now.Month()) - int(a.Month())
	for n > 0 && AddMonths(a, n).After(now) {
		n--
	}
	for AddMonths(a, n+1).Compare(now) <= 0 {
		n++
	}
	return AddMonths(a, n), AddMonths(a, n+1)
}

// Day formats a boundary the way the aggregate rows are keyed. Every stored
// number is already bucketed by local calendar day, so a period is summed by
// comparing these strings — no second notion of "which day is it" is
// introduced, and the invoice cannot disagree with the dashboard.
func Day(t time.Time) string { return t.Format("2006-01-02") }

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// daysIn asks the calendar rather than a table: day 0 of the next month is the
// last day of this one, leap years included.
func daysIn(y int, m time.Month) int {
	return time.Date(y, m+1, 0, 0, 0, 0, 0, time.Local).Day()
}

// WatermarkFee is what the "remove the Keel badge" add-on costs for one period.
//
// ⚠️ **Charged by the day, not by the month.** A restaurant that switches it on today and
// gets a full month's 3 million so'm on the same invoice has been overcharged in their own
// reading of it, and the argument that follows costs more than the fee. So the period is
// counted in days and the customer pays for the days the badge was actually hidden.
//
// ⚠️ **The last day is exclusive**, like every other period in this system: `[from, to)`.
// Counting it inclusively would charge one day twice a year — once at the end of a period and
// again at the start of the next — which is the kind of error nobody notices and everybody
// disputes when they finally do.
//
// A period with no days, a fee of zero or a badge that was never hidden all give zero. The
// function is pure so the arithmetic can be argued with in a test rather than on the phone.
func WatermarkFee(monthly int, from, to, since time.Time) int {
	if monthly <= 0 || !to.After(from) {
		return 0
	}
	// Never switched on: `since` zero means the flag is on but nobody recorded when, which is
	// every tenant that had it before this was billed. Treated as "the whole period", because
	// the alternative is billing them nothing for something they have been getting.
	start := from
	if !since.IsZero() && since.After(from) {
		start = since
	}
	if !to.After(start) {
		return 0
	}
	days := int(to.Sub(from).Hours() / 24)
	billed := int(to.Sub(start).Hours() / 24)
	if days <= 0 {
		return 0
	}
	if billed >= days {
		return monthly
	}
	// Rounded to the nearest so'm rather than truncated: over a year the difference is
	// pennies, and a figure that is consistently a hair low reads as sloppy arithmetic to the
	// person checking it.
	return (monthly*billed + days/2) / days
}
