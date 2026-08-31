package models

import (
	"testing"
	"time"
)

// ⚠️ **Rounding a commission up pays out money that was never collected.** One
// so'm at a time, on every payment, and the ledger stops balancing for a reason
// nobody can find in a list of correct-looking numbers.
func TestCommissionRoundsDown(t *testing.T) {
	rf := Referrer{Percent: 15}
	// 15% of 333 is 49.95.
	if got := rf.Commission(333); got != 49 {
		t.Fatalf("49 kutilgan, %d keldi", got)
	}
	if got := rf.Commission(0); got != 0 {
		t.Fatalf("nol to'lovdan komissiya chiqdi: %d", got)
	}
	if got := (Referrer{Percent: 0}).Commission(1_000_000); got != 0 {
		t.Fatalf("foizsiz hamkorga komissiya hisoblandi: %d", got)
	}
}

// ⚠️ **The window is counted from the day they started paying, not from the day
// they were created.** A trial pays nobody anything, so a window opened at
// signup would be half spent before the first invoice exists — and the referrer
// would be short-changed by exactly the length of the trial.
func TestWindowStartsWhenTheCustomerStartsPaying(t *testing.T) {
	subscribed := time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC)
	rf := Referrer{Percent: 10, Months: 3}

	end := rf.CommissionWindow(&subscribed)
	want := time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC)
	if !end.Equal(want) {
		t.Fatalf("%s kutilgan, %s keldi", want, end)
	}

	// ⚠️ A customer still on trial has no window at all rather than one that
	// has already closed: the zero time means "no limit", and reading a
	// not-yet-subscribed tenant as "expired" would silently zero the
	// commission on the customer who is about to start paying.
	if got := rf.CommissionWindow(nil); !got.IsZero() {
		t.Fatalf("obuna boshlanmagan mijozga oyna chizildi: %s", got)
	}

	// Zero months is "no limit", and it has to be typed.
	if got := (Referrer{Months: 0}).CommissionWindow(&subscribed); !got.IsZero() {
		t.Fatalf("cheklovsiz shartda oyna chiqdi: %s", got)
	}
}

// The code ends up in a URL that is printed on paper and read out over a
// phone, so it is held to the same alphabet a slug is.
func TestReferrerCodeAlphabet(t *testing.T) {
	for _, ok := range []string{"fiskal", "kassa-servis", "b5", "a1-b2-c3"} {
		if !ValidReferrerCode(ok) {
			t.Errorf("%q rad etildi", ok)
		}
	}
	for _, bad := range []string{"", "a", "Fiskal", "fiskal servis", "фискал", "-fiskal", "a_b"} {
		if ValidReferrerCode(bad) {
			t.Errorf("%q qabul qilindi", bad)
		}
	}
	// ⚠️ Case is dropped rather than refused: the code is typed off a leaflet
	// on a phone keyboard, and "Fiskal" failing where "fiskal" works is a
	// support message about a link that does not work.
	if got := NormalizeReferrerCode("  FISKAL "); got != "fiskal" {
		t.Fatalf("normalizatsiya: %q", got)
	}
}
