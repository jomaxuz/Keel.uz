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
	// A cashier is not a manager: the exit button is hidden from exactly the
	// people who would take the machine out of service by mis-tapping it.
	if v.CanExit {
		t.Error("a cashier may retire the screen — the exit button is not for them")
	}
	// Compile-time proof by construction: the view type's fields are a name, a
	// job title and three permissions — none of them money, a phone number or a
	// hash. If somebody adds one, this count stops matching and the test has to
	// be edited deliberately, which is the point.
	if got := fieldsOfTillPerson(); got != 7 {
		t.Fatalf("tillPersonView now has %d fields — check what was added", got)
	}
}

// ⚠️ **The role has to be resolved before this view is built**, and the
// consequence of forgetting is silent in both directions.
//
// A Staff read straight out of Mongo answers Can() from three legacy booleans
// that have no `void` at all — so an unknown permission is refused. That is the
// safe default everywhere else, and here it meant an "Ish boshqaruvchi"
// unlocked the till, was labelled a cashier (canCashier fell through to the
// legacy flag), and never saw the exit button. Nothing errored and nothing was
// logged; the only symptom was a control missing for exactly the people it is
// for.
//
// Sealed as the two halves of the same fact: with the role resolved the view
// says yes and carries the name, without it the view says no.
func TestTillPersonReadsTheRoleNotTheLegacyFlags(t *testing.T) {
	manager := models.Staff{
		Name:       "Dilshod",
		IsActive:   true,
		CanCashier: true,
		RoleName:   "Ish boshqaruvchi",
		Perms: []string{
			models.PermWaiter, models.PermCashier, models.PermVoid,
			models.PermDiscount, models.PermShift, models.PermKitchen,
		},
	}
	v := tillPerson(manager)
	if !v.CanExit {
		t.Error("a manager cannot retire the screen — the role was not read")
	}
	if v.Role != "Ish boshqaruvchi" {
		t.Errorf("role = %q, want the role's own name", v.Role)
	}

	// The same person with the role left unresolved — which is what
	// `staffByPIN` returns before `withRole` runs. It must fail closed, and it
	// must not claim a role name it does not have.
	raw := manager
	raw.Perms = nil
	raw.RoleName = ""
	if unresolved := tillPerson(raw); unresolved.CanExit {
		t.Error("an unresolved staff record granted the exit button")
	}
}

func fieldsOfTillPerson() int {
	return reflect.TypeOf(tillPersonView{}).NumField()
}

// ⚠️ **A PIN used to be enough to unlock the till whatever the role granted.**
//
// A technologist — whose seeded role grants nothing — opened the counter
// normally and then met "ruxsat yo'q" on the main screen, having already taken
// over somebody else's session. Two wrong things at once: the refusal arrived
// one screen too late, and the till was left locked to a person who cannot use
// it until somebody works out how to get back.
func TestAPinWithoutPermissionIsRefusedAtThePad(t *testing.T) {
	src := readLossSource(t, "tillpin.go")
	if !strings.Contains(src, "!person.Can(models.PermWaiter) && !person.Can(models.PermKitchen)") {
		t.Fatal("anybody with a PIN can take over the till again")
	}
	// ⚠️ **`waiter` is the floor, not `cashier`.** A waiter, a barman and a host
	// all have business on this screen; the drawer is a separate permission
	// asked for separately. Requiring `cashier` would lock out most of the
	// people the till was built for.
	if strings.Contains(src, "!person.Can(models.PermCashier) {") {
		t.Fatal("the till now refuses everybody who does not hold the drawer")
	}
	// ⚠️ Refused, and not counted as a failed PIN: the code was right. Locking
	// a branch out for a minute because a technologist tried their own PIN
	// would punish the queue for somebody else's curiosity.
	i := strings.Index(src, "!person.Can(models.PermWaiter)")
	if strings.Contains(src[i:i+400], "pinGate.fail(") {
		t.Fatal("a correct PIN from the wrong person locks the whole branch out")
	}
}
