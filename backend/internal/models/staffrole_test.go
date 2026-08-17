package models

import "testing"

// ⚠️ Cashier implies waiter, resolved in one place.
//
// A cashier who could take money but not add a dish would send every correction
// back across the room. The implication has to live in exactly one function, or
// a floor screen and a till screen will one day disagree about it — and the
// disagreement shows up as "the button works on his tablet and not on mine".
func TestCashierImpliesWaiter(t *testing.T) {
	r := &StaffRole{Perms: []string{PermCashier}}
	if !r.Allows(PermWaiter) {
		t.Fatal("a cashier cannot open the floor screen")
	}
	// And not the other way round: a waiter must not reach the drawer.
	w := &StaffRole{Perms: []string{PermWaiter}}
	if w.Allows(PermCashier) {
		t.Fatal("a waiter was granted the till")
	}
	if (&StaffRole{}).Allows(PermWaiter) {
		t.Fatal("an empty role grants something")
	}
	var nilRole *StaffRole
	if nilRole.Allows(PermWaiter) {
		t.Fatal("a missing role grants something")
	}
}

// ⚠️ **The most important assertion in this file.**
//
// `void` and `discount` did not exist before roles, so no legacy flag can mean
// "yes" to them. An unmigrated account must be refused rather than guessed at —
// guessing yes would hand every waiter in every restaurant that has not run the
// migration the ability to write off cooked food.
func TestLegacyAccountIsRefusedTheNewPermissions(t *testing.T) {
	legacy := &Staff{IsActive: true, CanCashier: true}
	if legacy.Can(PermVoid) || legacy.Can(PermDiscount) || legacy.Can(PermShift) {
		t.Fatal("an unmigrated account was granted a permission that has no old flag")
	}
	// What it could always do, it still can.
	if !legacy.Can(PermCashier) || !legacy.Can(PermWaiter) {
		t.Fatal("an unmigrated cashier lost the till")
	}
}

// The role wins once there is one; the legacy booleans stop being consulted.
//
// ⚠️ Both directions matter. A role that grants something the flags do not must
// work (that is the whole feature), and a role that withholds something the
// flags allowed must actually withhold it — otherwise tightening a role does
// nothing and the panel is lying to the owner.
func TestRoleOverridesTheLegacyFlags(t *testing.T) {
	granting := &Staff{
		IsActive: true,
		Perms:    []string{PermWaiter, PermVoid},
	}
	if !granting.Can(PermVoid) {
		t.Fatal("the role's permission was ignored")
	}
	withholding := &Staff{
		IsActive:   true,
		CanCashier: true, // the old flag says yes
		Perms:      []string{PermWaiter},
	}
	if withholding.Can(PermCashier) {
		t.Fatal("tightening a role did nothing — the legacy flag still answered")
	}
}

// ⚠️ An inactive employee holds nothing, whatever their role says. A staff
// token outlives a shift by days, so somebody dismissed this morning still has
// a working one tonight — the gap the kitchen screen had once.
func TestDismissedStaffHoldNothing(t *testing.T) {
	gone := &Staff{IsActive: false, Perms: []string{PermCashier, PermVoid}}
	for _, p := range AllPerms {
		if gone.Can(p) {
			t.Fatalf("a dismissed employee still holds %q", p)
		}
	}
}

// The shipped roles say what the plan says they say.
//
// ⚠️ Kassir deliberately ships **without** void and discount: taking payment
// and writing off cooked food are different decisions. Existing cashiers keep
// both through the migration, which maps them to Zal administratori rather than
// Kassir — a rename would have silently removed two abilities.
func TestSeedRoles(t *testing.T) {
	byName := map[string]StaffRole{}
	for _, r := range SeedRoles() {
		byName[r.Name] = r
	}

	kassir, ok := byName["Kassir"]
	if !ok {
		t.Fatal("no Kassir role is seeded")
	}
	if kassir.Allows(PermVoid) || kassir.Allows(PermDiscount) {
		t.Fatal("Kassir ships with void or discount — see the note above")
	}
	if !kassir.Allows(PermCashier) || !kassir.Allows(PermShift) {
		t.Fatal("Kassir cannot take payment or close the drawer")
	}

	// The role the migration moves today's cashiers onto has to be a superset
	// of what they can do today, or the deploy takes abilities away.
	admin, ok := byName["Zal administratori"]
	if !ok {
		t.Fatal("no Zal administratori role is seeded")
	}
	for _, p := range []string{PermWaiter, PermCashier, PermVoid, PermDiscount, PermShift} {
		if !admin.Allows(p) {
			t.Fatalf("Zal administratori lacks %q — migrating a cashier would demote them", p)
		}
	}

	ofitsiant := byName["Ofitsiant"]
	if !ofitsiant.Allows(PermWaiter) {
		t.Fatal("Ofitsiant cannot open a check")
	}
	if ofitsiant.Allows(PermVoid) {
		t.Fatal("Ofitsiant can write off cooked food")
	}
	oshpaz := byName["Oshpaz"]
	if !oshpaz.Allows(PermKitchen) {
		t.Fatal("Oshpaz cannot open the kitchen screen")
	}
	// ⚠️ A role with no permissions is a real answer, not an empty one: a
	// hostess or a technologist has an account and no business at the till.
	if len(byName["Xostes"].Perms) != 0 {
		t.Fatal("Xostes was given till permissions")
	}
}

// ⚠️ Nil slices marshal to `null` and the panel renders `null.length`. The JSON
// trap this codebase has been bitten by twice — this time it would arrive as a
// blank role editor.
func TestSeedRolesNeverCarryNilPerms(t *testing.T) {
	for _, r := range SeedRoles() {
		if r.Perms == nil {
			t.Fatalf("%q has nil perms — it will serialise as null", r.Name)
		}
	}
}
