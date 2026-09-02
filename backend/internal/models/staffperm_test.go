package models

import "testing"

// ⚠️ **A role that grants nothing was indistinguishable from no role at all,
// and somebody walked through it.**
//
// `Can` fell back to the pre-role booleans whenever the permission list was
// empty — and seeded roles do grant an empty list on purpose: Xostes and
// Yordamchi xodim (Texnolog was the third until it was given `stock`). So
// somebody assigned the role that grants nothing kept whatever `CanCashier` had
// been left at, opened the till, sent food to the kitchen, and cancelled a
// check without being asked for anybody's code.
func TestARoleThatGrantsNothingGrantsNothing(t *testing.T) {
	// Exactly the shape that shipped: a real role, no permissions, and the old
	// booleans still set on the account.
	s := &Staff{
		IsActive: true, RoleApplied: true, Perms: []string{},
		CanCashier: true, CanWaiter: true, CanKitchen: true,
	}
	for _, p := range []string{PermCashier, PermWaiter, PermKitchen, PermVoid} {
		if s.Can(p) {
			t.Fatalf("a role granting nothing still allowed %q", p)
		}
	}
}

// ⚠️ **A deleted role must not lock a shift out of the till mid-service.** The
// flag is set only where a role document was actually read, so a role that has
// gone missing leaves the old booleans in charge — the belt behind the panel's
// refusal to delete a role in use. Three states, and only one is the legacy
// path.
func TestAMissingRoleKeepsTheOldBooleans(t *testing.T) {
	s := &Staff{IsActive: true, CanCashier: true} // RoleApplied false
	if !s.Can(PermCashier) {
		t.Fatal("a deleted role locked somebody out of the till")
	}
}

// The ordinary case, unchanged: a role with permissions answers for them.
func TestARoleWithPermissionsStillWorks(t *testing.T) {
	s := &Staff{IsActive: true, RoleApplied: true,
		Perms: []string{PermWaiter, PermKitchen}}
	if !s.Can(PermWaiter) || !s.Can(PermKitchen) {
		t.Fatal("a granted permission was refused")
	}
	if s.Can(PermCashier) || s.Can(PermVoid) {
		t.Fatal("a permission nobody granted was allowed")
	}
}

// ⚠️ Cashier implies waiter, and that implication has to survive the rewrite:
// somebody trusted with the drawer is trusted to carry a plate, and making a
// restaurant tick both would mean discovering the second is missing at the
// counter on a Friday.
func TestCashierStillImpliesWaiter(t *testing.T) {
	s := &Staff{IsActive: true, RoleApplied: true, Perms: []string{PermCashier}}
	if !s.Can(PermWaiter) {
		t.Fatal("a cashier can no longer take an order")
	}
}

// An inactive employee is refused whatever their role says — somebody who has
// left is somebody who has left.
func TestAnInactiveEmployeeIsRefused(t *testing.T) {
	s := &Staff{RoleApplied: true, Perms: []string{PermCashier, PermVoid}}
	if s.Can(PermCashier) {
		t.Fatal("a deactivated account still opens the till")
	}
}
