package handlers

import (
	"testing"
	"time"

	"keel-control/internal/models"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

// A trial has an end, not a cycle. Showing a paying customer's "next invoice"
// date for someone who has not agreed to pay is how an operator quotes a bill
// that does not exist.
func TestTenantPeriodTrial(t *testing.T) {
	ends := day(2026, time.August, 19)
	tn := models.Tenant{
		Status:      models.StatusTrial,
		CreatedAt:   day(2026, time.August, 5),
		TrialEndsAt: &ends,
	}
	p := tenantPeriod(tn, day(2026, time.August, 10))
	if p.Kind != "trial" {
		t.Fatalf("kind = %q, want trial", p.Kind)
	}
	if p.From != "2026-08-05" || p.To != "2026-08-19" {
		t.Fatalf("trial window = %s → %s", p.From, p.To)
	}
}

// Bad data must still yield a window that can be summed; a row that silently
// disappears from the table is worse than a wrong one, because nobody looks
// for it.
func TestTenantPeriodTrialEndsBeforeItStarts(t *testing.T) {
	ends := day(2026, time.August, 1)
	tn := models.Tenant{
		Status:      models.StatusTrial,
		CreatedAt:   day(2026, time.August, 5),
		TrialEndsAt: &ends,
	}
	p := tenantPeriod(tn, day(2026, time.August, 6))
	if p.From >= p.To {
		t.Fatalf("empty window: %s → %s", p.From, p.To)
	}
}

func TestTenantPeriodSubscription(t *testing.T) {
	sub := day(2026, time.August, 17)
	tn := models.Tenant{
		Status:       models.StatusActive,
		CreatedAt:    day(2026, time.July, 1),
		SubscribedAt: &sub,
	}
	p := tenantPeriod(tn, day(2026, time.September, 3))
	if p.Kind != "subscription" || !p.Anchored {
		t.Fatalf("kind=%q anchored=%v", p.Kind, p.Anchored)
	}
	// Counted from the day they started paying — not from the day the tenant
	// was opened, which would bill them for their demo weeks.
	if p.From != "2026-08-17" || p.To != "2026-09-17" {
		t.Fatalf("window = %s → %s", p.From, p.To)
	}
}

// A tenant that became active before subscription dates were recorded still
// needs a number. It falls back to the day it was opened, and says so — the
// answer is the best available one, but an operator correcting an invoice
// deserves to know which date it rests on.
func TestTenantPeriodUnanchoredFallsBackToCreatedAt(t *testing.T) {
	tn := models.Tenant{
		Status:    models.StatusActive,
		CreatedAt: day(2026, time.March, 9),
	}
	p := tenantPeriod(tn, day(2026, time.August, 20))
	if p.Anchored {
		t.Fatal("anchored should be false without a subscription date")
	}
	if p.From != "2026-08-09" || p.To != "2026-09-09" {
		t.Fatalf("window = %s → %s", p.From, p.To)
	}
}

// A suspended tenant is still on a cycle: they owe what they used before the
// site was switched off, and the card has to keep saying so.
func TestTenantPeriodSuspendedKeepsCycle(t *testing.T) {
	sub := day(2026, time.May, 31)
	tn := models.Tenant{
		Status:       models.StatusSuspended,
		CreatedAt:    day(2026, time.May, 1),
		SubscribedAt: &sub,
	}
	p := tenantPeriod(tn, day(2026, time.July, 4))
	if p.Kind != "subscription" {
		t.Fatalf("kind = %q", p.Kind)
	}
	// June is short: the boundary clamps to the 30th, then returns to the 31st.
	if p.From != "2026-06-30" || p.To != "2026-07-31" {
		t.Fatalf("window = %s → %s", p.From, p.To)
	}
}

// An empty tenant document must not panic or produce a zero-year window that
// matches every row ever written.
func TestTenantPeriodZeroTenant(t *testing.T) {
	now := day(2026, time.August, 5)
	p := tenantPeriod(models.Tenant{}, now)
	if p.From != "2026-08-05" || p.To != "2026-09-05" {
		t.Fatalf("window = %s → %s", p.From, p.To)
	}
}

// The regression this cost a round of end-to-end testing to find.
//
// The Mongo driver hands back every date in UTC, whatever was written. A
// subscription anchored at local midnight therefore arrives as 19:00 the
// previous day in Tashkent, and truncating *that* to a day start moves the
// whole billing period back by one — silently, because the window is still a
// perfectly valid month.
func TestTenantPeriodAnchorsFromUTCDates(t *testing.T) {
	if time.Local == time.UTC {
		t.Skip("needs a non-UTC local zone to be meaningful")
	}
	// Exactly what the driver returns for a local midnight on 2026-07-17.
	sub := day(2026, time.July, 17).UTC()
	tn := models.Tenant{
		Status:       models.StatusActive,
		CreatedAt:    day(2026, time.January, 5).UTC(),
		SubscribedAt: &sub,
	}
	p := tenantPeriod(tn, day(2026, time.August, 5))
	if p.From != "2026-07-17" || p.To != "2026-08-17" {
		t.Fatalf("UTC-decoded anchor drifted: %s → %s", p.From, p.To)
	}

	ends := day(2026, time.August, 15).UTC()
	trial := models.Tenant{
		Status:      models.StatusTrial,
		CreatedAt:   day(2026, time.August, 1).UTC(),
		TrialEndsAt: &ends,
	}
	q := tenantPeriod(trial, day(2026, time.August, 5))
	if q.From != "2026-08-01" || q.To != "2026-08-15" {
		t.Fatalf("UTC-decoded trial drifted: %s → %s", q.From, q.To)
	}
}

// The warning is a countdown, and a countdown is only useful if its boundaries
// are exact: "3 days left" on the day it stops being true is a badge nobody
// trusts twice.
func TestTenantAttentionTrialCountdown(t *testing.T) {
	now := day(2026, time.August, 5)
	mk := func(ends time.Time) models.Tenant {
		e := ends
		return models.Tenant{Status: models.StatusTrial, TrialEndsAt: &e}
	}
	cases := []struct {
		ends time.Time
		kind string
		days int
	}{
		{day(2026, time.August, 9), "", 0},                    // 4 days — too early to shout
		{day(2026, time.August, 8), AttentionTrialEnding, 3},  // first day of warning
		{day(2026, time.August, 6), AttentionTrialEnding, 1},  // tomorrow
		{day(2026, time.August, 5), AttentionTrialExpired, 0}, // today: the deadline has arrived
		{day(2026, time.August, 2), AttentionTrialExpired, 3}, // three days over
	}
	for _, c := range cases {
		got := tenantAttention(mk(c.ends), now)
		if got.Kind != c.kind || got.Days != c.days {
			t.Errorf("ends %s: got %q/%d, want %q/%d",
				billingDay(c.ends), got.Kind, got.Days, c.kind, c.days)
		}
	}
}

// Whole calendar days, not 24-hour blocks: a demo ending tonight and one ending
// tomorrow morning are different answers even though a few hours separate them.
func TestTenantAttentionCountsCalendarDays(t *testing.T) {
	ends := day(2026, time.August, 6)
	tn := models.Tenant{Status: models.StatusTrial, TrialEndsAt: &ends}
	late := time.Date(2026, time.August, 5, 23, 30, 0, 0, time.Local)
	if got := tenantAttention(tn, late); got.Days != 1 {
		t.Fatalf("late evening: got %d days, want 1", got.Days)
	}
	// And the same date arriving from Mongo in UTC must not shift it.
	utc := day(2026, time.August, 6).UTC()
	tn.TrialEndsAt = &utc
	if got := tenantAttention(tn, day(2026, time.August, 5)); got.Days != 1 {
		t.Fatalf("UTC-decoded end: got %d days, want 1", got.Days)
	}
}

func TestTenantAttentionOtherStatuses(t *testing.T) {
	now := day(2026, time.August, 5)
	if got := tenantAttention(models.Tenant{Status: models.StatusSuspended}, now); got.Kind != AttentionUnpaid {
		t.Errorf("suspended: %q", got.Kind)
	}
	// A paying customer is never nagged about a deadline it does not have.
	sub := day(2026, time.July, 1)
	paid := models.Tenant{Status: models.StatusActive, SubscribedAt: &sub}
	if got := tenantAttention(paid, now); got.Kind != "" {
		t.Errorf("active: %q", got.Kind)
	}
	// A trial with no recorded end cannot be counted down; it must not be
	// reported as expired on the strength of a missing field.
	if got := tenantAttention(models.Tenant{Status: models.StatusTrial}, now); got.Kind != "" {
		t.Errorf("trial without an end: %q", got.Kind)
	}
}

func TestAttentionFilterRejectsUnknown(t *testing.T) {
	now := day(2026, time.August, 5)
	for _, k := range []string{AttentionTrialEnding, AttentionTrialExpired, AttentionUnpaid, "any"} {
		if _, ok := attentionFilter(k, now); !ok {
			t.Errorf("%q rejected", k)
		}
	}
	// An unknown value must not fall through to "no filter" — a button that
	// silently showed every customer would read as "nobody needs attention".
	if _, ok := attentionFilter("hammasi", now); ok {
		t.Error("unknown filter accepted")
	}
}

func billingDay(t time.Time) string { return t.Format("2006-01-02") }

func TestParseDayRejectsNonsense(t *testing.T) {
	for _, s := range []string{"", "bugun", "2206-01-02", "1999-01-01", "2026-13-40"} {
		if _, err := parseDay(s); err == nil {
			t.Errorf("parseDay(%q) accepted", s)
		}
	}
	got, err := parseDay("2026-08-17")
	if err != nil {
		t.Fatalf("parseDay: %v", err)
	}
	if !got.Equal(day(2026, time.August, 17)) {
		t.Fatalf("parseDay = %s", got)
	}
	// Local midnight, never UTC: a date parsed five hours off moves a boundary,
	// and therefore an invoice, by a day for anyone anchored on the 1st.
	if h, m, s := got.Clock(); h != 0 || m != 0 || s != 0 {
		t.Fatalf("not midnight: %s", got)
	}
	if got.Location() != time.Local {
		t.Fatalf("not local: %s", got.Location())
	}
}
