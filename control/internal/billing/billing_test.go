package billing

import (
	"testing"
	"time"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

func at(y int, m time.Month, d, h int) time.Time {
	return time.Date(y, m, d, h, 0, 0, 0, time.Local)
}

func TestAddMonthsClampsToShortMonths(t *testing.T) {
	anchor := day(2026, time.January, 31)
	cases := []struct {
		n    int
		want time.Time
	}{
		{0, day(2026, time.January, 31)},
		{1, day(2026, time.February, 28)}, // clamped
		{2, day(2026, time.March, 31)},    // and back to 31 — not stuck on 28
		{3, day(2026, time.April, 30)},
		{12, day(2027, time.January, 31)},
		{13, day(2027, time.February, 28)},
	}
	for _, c := range cases {
		if got := AddMonths(anchor, c.n); !got.Equal(c.want) {
			t.Errorf("AddMonths(31 Jan 2026, %d) = %s, want %s",
				c.n, Day(got), Day(c.want))
		}
	}
}

func TestAddMonthsLeapYear(t *testing.T) {
	// 2028 is a leap year: the same anchor lands on the 29th, not the 28th.
	if got := AddMonths(day(2028, time.January, 31), 1); !got.Equal(day(2028, time.February, 29)) {
		t.Errorf("leap February = %s, want 2028-02-29", Day(got))
	}
}

func TestCycleOrdinaryMonth(t *testing.T) {
	anchor := day(2026, time.August, 17)

	// The day the subscription starts is inside its own first period.
	from, to := Cycle(anchor, at(2026, time.August, 17, 0))
	if !from.Equal(day(2026, time.August, 17)) || !to.Equal(day(2026, time.September, 17)) {
		t.Fatalf("first day: %s → %s", Day(from), Day(to))
	}

	// The last hour before the boundary still belongs to the first period.
	from, to = Cycle(anchor, at(2026, time.September, 16, 23))
	if !from.Equal(day(2026, time.August, 17)) || !to.Equal(day(2026, time.September, 17)) {
		t.Fatalf("last hour: %s → %s", Day(from), Day(to))
	}

	// Midnight on the boundary has already rolled over.
	from, to = Cycle(anchor, at(2026, time.September, 17, 0))
	if !from.Equal(day(2026, time.September, 17)) || !to.Equal(day(2026, time.October, 17)) {
		t.Fatalf("boundary: %s → %s", Day(from), Day(to))
	}
}

// The trap the plan called out: a customer anchored on the 31st, asked about a
// February day. A period must exist, contain the day, and not be empty.
func TestCycleShortMonth(t *testing.T) {
	anchor := day(2026, time.January, 31)

	from, to := Cycle(anchor, at(2026, time.February, 10, 12))
	if !from.Equal(day(2026, time.January, 31)) || !to.Equal(day(2026, time.February, 28)) {
		t.Fatalf("mid-February: %s → %s", Day(from), Day(to))
	}

	// 28 February is itself a boundary that year: the next period starts.
	from, to = Cycle(anchor, at(2026, time.February, 28, 0))
	if !from.Equal(day(2026, time.February, 28)) || !to.Equal(day(2026, time.March, 31)) {
		t.Fatalf("28 Feb: %s → %s", Day(from), Day(to))
	}

	// And a March day sits in the period that came back to the 31st.
	from, to = Cycle(anchor, at(2026, time.March, 15, 9))
	if !from.Equal(day(2026, time.February, 28)) || !to.Equal(day(2026, time.March, 31)) {
		t.Fatalf("mid-March: %s → %s", Day(from), Day(to))
	}
}

// Whatever the anchor and whatever the day, the window must contain the day and
// join the next one exactly — no gap where orders are billed to nobody, no
// overlap where they are billed twice.
func TestCycleCoversEveryDayExactlyOnce(t *testing.T) {
	for _, anchorDay := range []int{1, 15, 28, 29, 30, 31} {
		anchor := day(2026, time.January, anchorDay)
		if anchor.Day() != anchorDay {
			continue // January has 31 days; this cannot happen, but be sure.
		}
		cur := anchor
		for i := 0; i < 400; i++ { // more than a year of days
			from, to := Cycle(anchor, cur)
			if from.After(cur) || to.Compare(cur) <= 0 {
				t.Fatalf("anchor %d: %s not inside [%s, %s)",
					anchorDay, Day(cur), Day(from), Day(to))
			}
			if !to.After(from) {
				t.Fatalf("anchor %d: empty period at %s", anchorDay, Day(cur))
			}
			// The period after this one starts exactly where this one ends.
			nf, _ := Cycle(anchor, to)
			if !nf.Equal(to) {
				t.Fatalf("anchor %d: gap at %s — next period starts %s",
					anchorDay, Day(to), Day(nf))
			}
			cur = cur.AddDate(0, 0, 1)
		}
	}
}

func TestCycleBeforeAnchorReturnsFirstPeriod(t *testing.T) {
	anchor := day(2026, time.September, 10)
	from, to := Cycle(anchor, at(2026, time.August, 1, 12))
	if !from.Equal(anchor) || !to.Equal(day(2026, time.October, 10)) {
		t.Fatalf("before anchor: %s → %s", Day(from), Day(to))
	}
}

// An anchor carrying a time of day must not shift the boundary: periods are
// whole days, because the rows they are summed from are whole days.
func TestCycleIgnoresTimeOfDay(t *testing.T) {
	anchor := time.Date(2026, time.August, 17, 19, 42, 0, 0, time.Local)
	from, to := Cycle(anchor, at(2026, time.August, 17, 3))
	if !from.Equal(day(2026, time.August, 17)) || !to.Equal(day(2026, time.September, 17)) {
		t.Fatalf("time of day leaked: %s → %s", Day(from), Day(to))
	}
}
