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
		got := tenantAttention(mk(c.ends), now, "", "running")
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
	if got := tenantAttention(tn, late, "", "running"); got.Days != 1 {
		t.Fatalf("late evening: got %d days, want 1", got.Days)
	}
	// And the same date arriving from Mongo in UTC must not shift it.
	utc := day(2026, time.August, 6).UTC()
	tn.TrialEndsAt = &utc
	if got := tenantAttention(tn, day(2026, time.August, 5), "", "running"); got.Days != 1 {
		t.Fatalf("UTC-decoded end: got %d days, want 1", got.Days)
	}
}

func TestTenantAttentionOtherStatuses(t *testing.T) {
	now := day(2026, time.August, 5)
	if got := tenantAttention(models.Tenant{Status: models.StatusSuspended}, now, "", "running"); got.Kind != AttentionUnpaid {
		t.Errorf("suspended: %q", got.Kind)
	}
	// A paying customer is never nagged about a *trial* deadline it does not
	// have — and, with the ledger already covering the period that closed, not
	// about an invoice either.
	sub := day(2026, time.July, 1)
	paid := models.Tenant{Status: models.StatusActive, SubscribedAt: &sub}
	if got := tenantAttention(paid, now, "2026-08-01", "running"); got.Kind != "" {
		t.Errorf("active: %q", got.Kind)
	}
	// A trial with no recorded end cannot be counted down; it must not be
	// reported as expired on the strength of a missing field.
	if got := tenantAttention(models.Tenant{Status: models.StatusTrial}, now, "", "running"); got.Kind != "" {
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

// The reminder that a closed period was never billed.
//
// This is the one warning about our own behaviour rather than the customer's,
// and it fails silently in both directions — an un-issued invoice produces no
// complaint from anybody, and a false alarm teaches an operator to ignore the
// queue. Both directions are pinned here.
func TestTenantAttentionInvoiceDue(t *testing.T) {
	sub := day(2026, time.July, 17)
	tn := models.Tenant{Status: models.StatusActive, SubscribedAt: &sub}

	// Still inside the first period: nothing has closed, so nothing is owed a
	// number yet.
	if got := tenantAttention(tn, day(2026, time.August, 10), "", "running"); got.Kind != "" {
		t.Errorf("inside the first period: %q", got.Kind)
	}

	// 17 Aug: the 17 Jul → 17 Aug period has closed and the ledger is empty.
	got := tenantAttention(tn, day(2026, time.August, 20), "", "running")
	if got.Kind != AttentionInvoiceDue {
		t.Fatalf("closed and unbilled: %q, want %q", got.Kind, AttentionInvoiceDue)
	}
	if got.Days != 3 {
		t.Errorf("days = %d, want 3 days since the period closed", got.Days)
	}

	// Invoiced through the day the open period began: covered exactly, and the
	// half-open bound must not read as one day short.
	if got := tenantAttention(tn, day(2026, time.August, 20), "2026-08-17", "running"); got.Kind != "" {
		t.Errorf("already invoiced: %q", got.Kind)
	}

	// An invoice that stops a day early leaves the period uncovered, and the
	// reminder has to come back rather than treat "nearly" as done.
	if got := tenantAttention(tn, day(2026, time.August, 20), "2026-08-16", "running"); got.Kind != AttentionInvoiceDue {
		t.Errorf("invoiced a day short: %q", got.Kind)
	}

	// A customer on free terms is never in the queue: the conversation about
	// money already happened and ended differently.
	free := tn
	free.Free = true
	if got := tenantAttention(free, day(2026, time.August, 20), "", "running"); got.Kind != "" {
		t.Errorf("free customer: %q", got.Kind)
	}

	// No anchor at all — neither a subscription date nor a creation date — is a
	// broken record, not a bill. Guessing a period from `now` would invoice
	// somebody for a month nobody agreed to.
	if got := tenantAttention(models.Tenant{Status: models.StatusActive}, day(2026, time.August, 20), "", "running"); got.Kind != "" {
		t.Errorf("no anchor: %q", got.Kind)
	}
}

// The dark-site warning, and the two ways it must not misfire.
//
// It exists because `provisionStatus` is a stored flag: a real tenant sat at
// "ready" with no container at all while its domain returned 502, and the
// console reported it as fully provisioned. A flag records what happened once;
// this asks Docker.
func TestTenantAttentionDown(t *testing.T) {
	now := day(2026, time.August, 20)
	sub := day(2026, time.July, 17)
	live := models.Tenant{Status: models.StatusActive, SubscribedAt: &sub}

	// No container at all — the case that started this.
	if got := tenantAttention(live, now, "2026-08-17", "absent"); got.Kind != AttentionDown {
		t.Errorf("absent: %q, want %q", got.Kind, AttentionDown)
	}
	if got := tenantAttention(live, now, "2026-08-17", "stopped"); got.Kind != AttentionDown {
		t.Errorf("stopped: %q", got.Kind)
	}
	// A crash-looping container reports itself running between restarts, which
	// is exactly how a broken tenant passes for a healthy one.
	if got := tenantAttention(live, now, "2026-08-17", "restarting"); got.Kind != AttentionDown {
		t.Errorf("restarting: %q", got.Kind)
	}

	// ⚠️ "I cannot ask" is not "it is down". No Docker socket is the normal
	// state on a laptop, and a warning that is always on is one nobody reads.
	if got := tenantAttention(live, now, "2026-08-17", ""); got.Kind != "" {
		t.Errorf("no docker: %q, want no warning", got.Kind)
	}
	if got := tenantAttention(live, now, "2026-08-17", "unknown"); got.Kind != "" {
		t.Errorf("docker errored: %q, want no warning", got.Kind)
	}

	// A suspended or closed site is dark because we made it dark. Reporting
	// that as a fault would put every departed customer in the queue forever —
	// and it must keep saying why it is actually off.
	off := models.Tenant{Status: models.StatusSuspended, SubscribedAt: &sub}
	if got := tenantAttention(off, now, "", "absent"); got.Kind != AttentionUnpaid {
		t.Errorf("suspended: %q, want %q", got.Kind, AttentionUnpaid)
	}
	gone := models.Tenant{Status: models.StatusDeleted, SubscribedAt: &sub}
	if got := tenantAttention(gone, now, "", "absent"); got.Kind != "" {
		t.Errorf("deleted: %q, want no warning", got.Kind)
	}

	// It outranks money, and it outranks free terms. A free customer is exempt
	// from being chased for payment, not from having a working site — they are
	// usually the anchor customer the early product rests on.
	if got := tenantAttention(live, now, "", "absent"); got.Kind != AttentionDown {
		t.Errorf("down beside an unpaid period: %q, want %q", got.Kind, AttentionDown)
	}
	free := live
	free.Free = true
	free.FreeReason = "anchor"
	if got := tenantAttention(free, now, "", "absent"); got.Kind != AttentionDown {
		t.Errorf("free customer with a dark site: %q, want %q", got.Kind, AttentionDown)
	}

	// A trial being evaluated right now is the worst possible moment for this.
	ends := day(2026, time.September, 1)
	trial := models.Tenant{Status: models.StatusTrial, TrialEndsAt: &ends}
	if got := tenantAttention(trial, now, "", "absent"); got.Kind != AttentionDown {
		t.Errorf("trial with a dark site: %q", got.Kind)
	}
}

// A slug Docker did not list has no container — that is the answer, not a gap.
// But an empty map means "nothing was asked", and every tenant must then read
// as unknown rather than as absent.
func TestStateOfDistinguishesAbsentFromUnasked(t *testing.T) {
	if got := stateOf(map[string]string{}, "kfc"); got != "" {
		t.Errorf("no docker: %q, want empty (unknown)", got)
	}
	states := map[string]string{"b5somsa": "running"}
	if got := stateOf(states, "kfc"); got != "absent" {
		t.Errorf("missing from a real listing: %q, want absent", got)
	}
	if got := stateOf(states, "b5somsa"); got != "running" {
		t.Errorf("listed: %q", got)
	}
}
