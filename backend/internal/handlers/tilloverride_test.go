package handlers

import (
	"context"
	"errors"
	"testing"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ⚠️ A Handler with no Store: any path that reaches the database panics, and
// the panic is the assertion. The three decisions below must be made before a
// query is even considered — a till that hit Mongo to find out whether the
// person in front of it holds a permission would add a round trip to the
// busiest button in the building.
func offlineHandler() *Handler { return &Handler{} }

func person(perms ...string) models.Staff {
	return models.Staff{
		ID: primitive.NewObjectID(), Name: "Aziz",
		BranchID: primitive.NewObjectID(), IsActive: true, Perms: perms,
	}
}

// Somebody who holds the permission owns the action outright — no second name,
// no lookup.
func TestActorNeedsNobodyWhenTheyHoldThePermission(t *testing.T) {
	s := person(models.PermWaiter, models.PermVoid)
	who, err := offlineHandler().resolveActor(
		context.Background(), s, models.PermVoid, "")
	if err != nil {
		t.Fatalf("a permitted action was refused: %v", err)
	}
	if who.ByID != s.ID || who.By != "Aziz" {
		t.Fatalf("the action was attributed to somebody else: %+v", who)
	}
	// ⚠️ No authoriser. Recording one where none was needed would put a
	// manager's name on work they never saw.
	if who.AuthBy != "" || !who.AuthByID.IsZero() {
		t.Fatalf("an authoriser appeared out of nowhere: %+v", who)
	}
}

// ⚠️ **Missing permission asks, it does not refuse.** This is the distinction
// the whole feature rests on: refusing is what teaches a room to share one PIN,
// and after that every void carries the same name.
func TestMissingPermissionAsksForACode(t *testing.T) {
	s := person(models.PermWaiter)
	_, err := offlineHandler().resolveActor(
		context.Background(), s, models.PermVoid, "")
	if !errors.Is(err, errNeedsOverride) {
		t.Fatalf("want errNeedsOverride, got %v", err)
	}
}

// A code that is not even the right shape is answered the same way — without a
// database lookup. Four digits is cheap to check and the endpoint is behind a
// PIN pad that cannot produce anything else.
func TestMalformedOverridePinNeverReachesTheDatabase(t *testing.T) {
	s := person(models.PermWaiter)
	for _, bad := range []string{"1", "12345", "abcd", "12 4"} {
		_, err := offlineHandler().resolveActor(
			context.Background(), s, models.PermVoid, bad)
		if !errors.Is(err, errNeedsOverride) {
			t.Fatalf("%q: want errNeedsOverride, got %v", bad, err)
		}
	}
}

// ⚠️ An inactive employee holds nothing, so their own actions need authorising
// too. A staff token outlives a shift by days — the gap the kitchen screen had
// once, arriving here.
func TestDismissedStaffCannotActOnTheirOwnAuthority(t *testing.T) {
	s := person(models.PermVoid)
	s.IsActive = false
	_, err := offlineHandler().resolveActor(
		context.Background(), s, models.PermVoid, "")
	if !errors.Is(err, errNeedsOverride) {
		t.Fatalf("a dismissed employee acted on their own authority: %v", err)
	}
}

// The refusal has to be readable by somebody holding plates: `discount` is a
// word from our database.
func TestPermissionsAreNamedInWords(t *testing.T) {
	for _, p := range models.AllPerms {
		if permLabel(p) == p {
			t.Fatalf("%q has no human name — it would be shown as-is", p)
		}
	}
}
