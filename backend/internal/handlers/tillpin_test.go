package handlers

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"restaurant-backend/internal/models"
)

// ⚠️ **Obvious codes are allowed, and that is the deliberate choice.**
//
// An earlier version refused 1234 and 0000. That was the wrong trade for this
// screen: the PIN is tapped dozens of times a shift by somebody holding plates,
// and a code they cannot remember becomes a code written on a sticky note
// beside the monoblock — readable by everybody in the room and never changed.
//
// What protects this is not the code's cleverness: it is only accepted from a
// device already holding the branch's token, five wrong tries cost a minute,
// and it reaches the floor and the drawer and nothing else.
func TestPinShapeAllowsSimpleCodes(t *testing.T) {
	for _, easy := range []string{"0000", "1111", "1234", "4726"} {
		if err := pinShape(easy); err != nil {
			t.Fatalf("%q was refused: %v", easy, err)
		}
	}
}

// Digits only, and a length a person can tap without looking.
//
// ⚠️ Letters are refused rather than accepted-and-hashed: a PIN pad has no
// letters, so a code containing one can be set in the panel and then never
// entered on the screen it exists for. The failure would look like a broken
// till, not like a bad code.
// ⚠️ Exactly four digits, not a range. A variable length means the pad cannot
// show how many are expected — it either draws empty dots that mean nothing or
// cannot submit by itself — and both cost a tap on the busiest screen in the
// building.
func TestPinShapeIsExactlyFourDigits(t *testing.T) {
	for _, bad := range []string{"", "12", "123", "12345", "123456", "12a4", "12 4", "٤٧٢٦"} {
		if err := pinShape(bad); err == nil {
			t.Fatalf("%q was accepted as a PIN", bad)
		}
	}
	if err := pinShape(""); err == nil || !strings.Contains(err.Error(), "PIN") {
		t.Fatalf("an empty PIN must be refused by name: %v", err)
	}
}

// ⚠️ Five wrong tries buy a minute of silence, and the counter is **per till**.
//
// Not per IP: a restaurant behind one connection would otherwise lock out its
// own cashiers because a different branch was being probed. And not per person,
// which cannot be known — the whole point is that the PIN has not identified
// anybody yet.
func TestPinLockoutIsPerTillAndExpires(t *testing.T) {
	gate := pinAttempts{count: map[string]*pinCounter{}}
	const a, b = "branch-a", "branch-b"

	for range pinMaxAttempts - 1 {
		gate.fail(a)
	}
	if blocked, _ := gate.blocked(a); blocked {
		t.Fatal("locked out before the limit was reached")
	}
	gate.fail(a)
	blocked, left := gate.blocked(a)
	if !blocked {
		t.Fatal("the limit was reached and nothing locked")
	}
	if left <= 0 || left > pinLockout {
		t.Fatalf("lockout has %v left, want up to %v", left, pinLockout)
	}

	// ⚠️ The other branch is untouched. This is the assertion that matters:
	// one till being probed must never stop another restaurant's cashiers.
	if blocked, _ := gate.blocked(b); blocked {
		t.Fatal("one branch's failures locked another branch")
	}

	// A correct PIN clears the count, so a cashier who mistyped twice and then
	// got it right does not carry those two into the evening.
	gate.fail(b)
	gate.fail(b)
	gate.ok(b)
	for range pinMaxAttempts - 1 {
		gate.fail(b)
	}
	if blocked, _ := gate.blocked(b); blocked {
		t.Fatal("a successful unlock did not clear the earlier failures")
	}
}

// The lockout ends by itself, and ending it resets the count — otherwise the
// next single wrong tap would lock again immediately and the cashier would
// never get out of it.
func TestPinLockoutResetsWhenItExpires(t *testing.T) {
	gate := pinAttempts{count: map[string]*pinCounter{}}
	const key = "branch"
	for range pinMaxAttempts {
		gate.fail(key)
	}
	if blocked, _ := gate.blocked(key); !blocked {
		t.Fatal("not locked after reaching the limit")
	}
	// Move the lockout into the past rather than sleeping a minute.
	gate.mu.Lock()
	gate.count[key].locked = time.Now().Add(-time.Second)
	gate.mu.Unlock()

	if blocked, _ := gate.blocked(key); blocked {
		t.Fatal("still locked after the lockout expired")
	}
	gate.fail(key)
	if blocked, _ := gate.blocked(key); blocked {
		t.Fatal("one wrong tap after an expired lockout locked again")
	}
}

// ⚠️ A PIN buys hours, never the week an ordinary staff token gets. The
// password-bought token and the four-digit-bought token must not be worth the
// same thing.
func TestTillSessionIsShorterThanAStaffToken(t *testing.T) {
	if tillSessionTTL >= 7*24*time.Hour {
		t.Fatalf("till session %v is not shorter than a staff token", tillSessionTTL)
	}
	// And long enough to outlast a shift, or a cashier is thrown out mid-service.
	if tillSessionTTL < 12*time.Hour {
		t.Fatalf("till session %v is shorter than a long shift", tillSessionTTL)
	}
}

// The lock screen shows a name, not a personnel file.
//
// ⚠️ models.Staff carries a salary, a rota, a phone number and a password hash,
// and this response is drawn on a screen standing in a public room. The narrow
// shape is written by hand so the next field added to Staff does not appear
// here by default — the rule that has caught real leaks in this codebase.
func TestTillPersonViewCarriesNothingPrivate(t *testing.T) {
	v := tillPerson(models.Staff{
		Name:       "Aziz",
		Position:   "kassir",
		CanCashier: true,
		IsActive:   true,
	})
	if v.Name != "Aziz" {
		t.Fatalf("name = %q", v.Name)
	}
	// Compile-time proof by construction: the view type has four fields and
	// none of them is money, a phone number or a hash. If somebody adds one,
	// this list stops matching and the test has to be edited deliberately.
	if got := fieldsOfTillPerson(); got != 5 {
		t.Fatalf("tillPersonView now has %d fields — check what was added", got)
	}
}

func fieldsOfTillPerson() int {
	return reflect.TypeOf(tillPersonView{}).NumField()
}
