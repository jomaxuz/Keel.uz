package handlers

import (
	"testing"

	"restaurant-backend/internal/models"
)

// Who may open the kitchen screen.
//
// ⚠️ Until this existed, **any** staff token could read the KDS and press
// "Tayyor" — a button that makes a ticket vanish from the pass and tells the
// panel the food is done. A shared tablet is the whole point of the staff role,
// so "has a login" cannot be the same question as "runs the kitchen screen".
func TestKitchenDenial(t *testing.T) {
	cook := models.Staff{IsActive: true, CanKitchen: true}
	if r := kitchenDenial(cook); r != "" {
		t.Fatalf("a permitted, active cook must be let in, got %q", r)
	}

	waiter := models.Staff{IsActive: true}
	if kitchenDenial(waiter) == "" {
		t.Fatal("a staff account without the permission must be refused")
	}

	// ⚠️ A staff token outlives a shift by a long way, so somebody deactivated
	// this morning still holds a working one this afternoon. The clock-in
	// endpoint already checked this; the KDS did not, which meant a dismissed
	// cook kept the pass.
	fired := models.Staff{IsActive: false, CanKitchen: true}
	if kitchenDenial(fired) == "" {
		t.Fatal("a deactivated account must be refused even with the permission")
	}

	// The refusals say different things, because the next step differs: one is
	// "ask your manager for access", the other is "your account is switched
	// off". One message for both would send half the people to the wrong
	// person.
	if kitchenDenial(waiter) == kitchenDenial(fired) {
		t.Fatal("the two refusals must be distinguishable to the person reading them")
	}

	// The zero value refuses. Says out loud that this is the one place the
	// codebase's usual "empty means today's behaviour" rule is deliberately
	// inverted: a permission everybody has by default is not a permission, and
	// existing staff are grandfathered by EnsureKitchenAccess instead.
	if kitchenDenial(models.Staff{}) == "" {
		t.Fatal("a blank staff record must not have kitchen access")
	}
}
