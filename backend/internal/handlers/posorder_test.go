package handlers

import (
	"errors"
	"testing"
	"time"

	"restaurant-backend/internal/models"
	"restaurant-backend/internal/pos"

	"go.mongodb.org/mongo-driver/bson"
)

func TestTillUpdateAcceptance(t *testing.T) {
	now := time.Date(2026, 8, 16, 19, 0, 0, 0, time.UTC)

	// Poster's `status: 0`: filed, and nobody at the counter has taken it. The
	// state this whole file exists to make visible, so it must never be left
	// blank — blank is "never asked", and the panel's warning turns on the
	// difference.
	waiting := tillUpdate(nil, pos.Status{State: "unknown", Raw: "kutilmoqda"}, nil, now)
	if waiting.State != models.TillWaiting {
		t.Fatalf("unknown should be waiting, got %q", waiting.State)
	}
	if waiting.AcceptedAt != nil {
		t.Fatal("nothing was accepted")
	}
	if waiting.CheckedAt == nil {
		t.Fatal("checkedAt must always be written: a state without a time ages into a lie")
	}

	// Everything past acceptance counts as accepted. iiko answers with where
	// the order is in the kitchen, and treating "cooking" as unconfirmed would
	// keep warning about an order already being made.
	for _, state := range []string{"accepted", "cooking", "ready", "closed"} {
		got := tillUpdate(waiting, pos.Status{State: state}, nil, now)
		if got.State != models.TillAccepted {
			t.Fatalf("%q should count as accepted, got %q", state, got.State)
		}
		if got.AcceptedAt == nil {
			t.Fatalf("%q must stamp acceptedAt", state)
		}
	}

	// Accepted once, accepted for good: the timestamp is the answer to "when
	// did the kitchen get it", and re-stamping it on every poll would make it
	// the answer to "when did we last ask" instead.
	accepted := tillUpdate(waiting, pos.Status{State: "accepted"}, nil, now)
	later := tillUpdate(accepted, pos.Status{State: "cooking"}, nil, now.Add(time.Hour))
	if later.AcceptedAt == nil || !later.AcceptedAt.Equal(*accepted.AcceptedAt) {
		t.Fatal("acceptedAt must be set once and never moved")
	}
}

// A till that cannot be reached has not un-accepted anything. Blanking the
// state on a network error would turn every blip into "waiting for the
// cashier" — an alarm about somebody else's kitchen raised by our own
// connectivity.
func TestTillUpdateKeepsVerdictOnError(t *testing.T) {
	now := time.Now()
	accepted := tillUpdate(nil, pos.Status{State: "accepted", Raw: "chek №42"}, nil, now)

	broken := tillUpdate(accepted, pos.Status{}, errors.New("ulanib bo'lmadi"), now.Add(time.Minute))
	if broken.State != models.TillAccepted {
		t.Fatalf("verdict must survive an unreachable till, got %q", broken.State)
	}
	if broken.AcceptedAt == nil {
		t.Fatal("acceptedAt must survive too")
	}
	if broken.Error == "" {
		t.Fatal("the reason must be recorded, or the stale check time has no explanation")
	}

	// And the error clears once the till answers again, rather than sitting on
	// the receipt as a permanent scar from one bad minute.
	fixed := tillUpdate(broken, pos.Status{State: "accepted"}, nil, now.Add(2*time.Minute))
	if fixed.Error != "" {
		t.Fatal("a recovered check must clear the error")
	}
}

// r_keeper has no document to ask about. That is a shrug, not a fault: recorded
// once so the poller stops asking (pendingTillFilter excludes it) and the panel
// stops implying a verdict is coming.
func TestTillUpdateUnsupported(t *testing.T) {
	got := tillUpdate(nil, pos.Status{}, pos.ErrUnsupported, time.Now())
	if got.State != models.TillUnsupported {
		t.Fatalf("want unsupported, got %q", got.State)
	}
	if got.Error != "" {
		t.Fatal("unsupported is not an error and must not be shown as one")
	}
}

// The backoff: fresh orders every tick, old ones rarely. A till nobody is
// watching must not cost one API call a minute for six hours.
func TestTillCheckDueBacksOff(t *testing.T) {
	now := time.Now()
	sent := now.Add(-30 * time.Second)

	if !tillCheckDue(nil, sent, now) {
		t.Fatal("never checked must always be due")
	}
	if !tillCheckDue(&models.OrderTill{State: models.TillWaiting}, sent, now) {
		t.Fatal("a state with no checkedAt must be due, not skipped forever")
	}

	justChecked := now.Add(-20 * time.Second)
	if tillCheckDue(&models.OrderTill{CheckedAt: &justChecked}, sent, now) {
		t.Fatal("20s after a check, on a 1m schedule, is not due")
	}
	minuteAgo := now.Add(-time.Minute)
	if !tillCheckDue(&models.OrderTill{CheckedAt: &minuteAgo}, sent, now) {
		t.Fatal("a fresh order is due every minute")
	}

	// Past the backoff point the same gap is no longer enough.
	old := now.Add(-2 * time.Hour)
	if tillCheckDue(&models.OrderTill{CheckedAt: &minuteAgo}, old, now) {
		t.Fatal("an old order must fall onto the slow schedule")
	}
	fiveAgo := now.Add(-5 * time.Minute)
	if !tillCheckDue(&models.OrderTill{CheckedAt: &fiveAgo}, old, now) {
		t.Fatal("the slow schedule is still a schedule")
	}
}

// ⚠️ The filter has to match orders that have never been asked about — the
// field is absent on every order sent before this existed. Sealed because the
// obvious tightening (an explicit `$in` of the unresolved states) reads as
// equivalent and would quietly exclude exactly those orders.
func TestPendingTillFilterMatchesNeverChecked(t *testing.T) {
	cond, ok := pendingTillFilter(time.Now())["pos.till.state"].(bson.M)
	if !ok {
		t.Fatal("the filter must constrain the till state")
	}
	// $nin is the operator that treats a missing field as a match; $in is not.
	if _, bad := cond["$in"]; bad {
		t.Fatal("$in would skip every order never checked — see the comment on pendingTillFilter")
	}
	if _, good := cond["$nin"]; !good {
		t.Fatal("expected $nin so absent states still match")
	}
}

// The warning is about a person at a counter. Our own silence must not raise
// it: an order we never managed to ask about, or one on a till that cannot
// answer, is not evidence that anybody ignored anything.
func TestUnacceptedFilterIsWaitingExactly(t *testing.T) {
	f := unacceptedTillFilter(time.Now())
	if f["pos.till.state"] != models.TillWaiting {
		t.Fatalf("the alert must require an explicit waiting verdict, got %v", f["pos.till.state"])
	}
}

// The stop-list poller's hours rule. The first half is the trap: isOpenNow
// answers false for an empty schedule, so reading it straight would switch the
// mirror off for every restaurant that never filled its hours in — silently,
// and with nothing naming the cause.
func TestSyncOpenAt(t *testing.T) {
	// 2026-08-16 is a Sunday (weekday 0).
	at := func(h, m int) time.Time {
		return time.Date(2026, 8, 16, h, m, 0, 0, time.UTC)
	}
	if !syncOpenAt(nil, at(4, 0)) {
		t.Fatal("no working hours must mean poll, not skip")
	}

	hours := []models.WorkingHour{{Day: 0, Open: "10:00", Close: "22:00"}}
	if syncOpenAt(hours, at(4, 0)) {
		t.Fatal("4am on a closed kitchen must not be polled")
	}
	if !syncOpenAt(hours, at(13, 0)) {
		t.Fatal("mid-service must be polled")
	}
	// Opened early, so the list is current when the doors are.
	if !syncOpenAt(hours, at(9, 50)) {
		t.Fatal("the lead window must start before opening")
	}
	if syncOpenAt(hours, at(9, 30)) {
		t.Fatal("the lead is 15 minutes, not an hour")
	}

	// A kitchen that closes at 02:00 is open past midnight, and the overnight
	// branch of isOpenNow is exactly where an hours check usually breaks.
	night := []models.WorkingHour{{Day: 0, Open: "18:00", Close: "02:00"}}
	if !syncOpenAt(night, at(1, 0)) {
		t.Fatal("1am is still service for an overnight kitchen")
	}
	if syncOpenAt(night, at(12, 0)) {
		t.Fatal("noon is not")
	}

	closed := []models.WorkingHour{{Day: 0, IsClosed: true}}
	if syncOpenAt(closed, at(13, 0)) {
		t.Fatal("a day off must not be polled")
	}
}

// The failure the panel was blindest to: the order never reached the till.
//
// ⚠️ Deliberately **not** time-bounded, unlike the till check. An order that
// failed an hour ago and is still open is more worth showing, not less — the
// guest is still waiting for food nobody is cooking. Settled orders drop out
// through the status filter instead.
func TestFailedPOSFilter(t *testing.T) {
	f := failedPOSFilter()
	if f["pos.status"] != models.POSFailed {
		t.Fatalf("must select failed sends, got %v", f["pos.status"])
	}
	if _, bounded := f["pos.sentAt"]; bounded {
		t.Fatal("no time bound: an old unfixed failure is the important one")
	}
	// Cancelled and delivered orders were settled by people; whatever the till
	// thinks about them is history, not a question.
	cond, ok := f["status"].(bson.M)
	if !ok {
		t.Fatal("settled orders must be excluded")
	}
	if _, has := cond["$nin"]; !has {
		t.Fatal("expected $nin on order status")
	}
}

// Which missing links are worth warning about, before any order exists.
//
// Both exclusions look like details and are the difference between a warning
// people act on and one they switch off.
func TestUnmappedMatters(t *testing.T) {
	plain := models.MenuItem{IsAvailable: true}
	if !unmappedMatters(plain, false) {
		t.Fatal("a sellable, unmapped dish is exactly the thing being prevented")
	}
	if unmappedMatters(plain, true) {
		t.Fatal("a mapped dish is not a problem")
	}

	// ⚠️ A combo is never sent as itself — posItems expands it into its members
	// and prices them individually, because the till has no product for a
	// bundle this site invented. Counting it would report a problem the mapping
	// screen cannot fix, and a warning with no possible action is one the owner
	// learns to scroll past.
	combo := models.MenuItem{
		IsAvailable: true,
		ComboItems:  []models.ComboLine{{Qty: 1}},
	}
	if unmappedMatters(combo, false) {
		t.Fatal("a combo needs no mapping of its own")
	}

	// Off the menu, so it cannot reach an order. It will be counted if it comes
	// back — warning now would mean a permanent count nobody can clear without
	// mapping dishes the restaurant deliberately stopped selling.
	hidden := models.MenuItem{IsAvailable: false}
	if unmappedMatters(hidden, false) {
		t.Fatal("a dish nobody can order cannot break an order")
	}
}
