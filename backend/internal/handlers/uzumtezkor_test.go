package handlers

import (
	"strings"
	"testing"
	"time"

	"restaurant-backend/internal/models"
)

// ⚠️ Uzum allows no step backwards and cancels an order that is not accepted in
// fifteen minutes, so this mapping is the integration. READY comes from
// `readyAt` — the kitchen's "done" is a timestamp here, not a status — and only
// once the order was accepted.
func TestTezkorStatusFollowsTheOrder(t *testing.T) {
	ready := time.Now()
	cases := []struct {
		status models.OrderStatus
		ready  bool
		want   string
	}{
		{models.StatusPending, false, "NEW"},
		// A ready stamp on an unaccepted order must not skip acceptance.
		{models.StatusPending, true, "NEW"},
		{models.StatusConfirmed, false, "ACCEPTED_BY_RESTAURANT"},
		{models.StatusConfirmed, true, "READY"},
		{models.StatusPreparing, false, "COOKING"},
		{models.StatusPreparing, true, "READY"},
		{models.StatusOnTheWay, true, "TAKEN_BY_COURIER"},
		{models.StatusDelivered, true, "DELIVERED"},
		{models.StatusCancelled, true, "CANCELLED"},
	}
	for _, c := range cases {
		o := &models.Order{Status: c.status}
		if c.ready {
			o.ReadyAt = &ready
		}
		if got := tezkorStatusOf(o); got != c.want {
			t.Fatalf("%s ready=%v: got %s, want %s", c.status, c.ready, got, c.want)
		}
	}
}

// Every problem at once, and a fractional quantity refused rather than rounded:
// half a portion of lag'mon is not a line a kitchen can cook.
func TestTezkorCheckReportsEverythingWrong(t *testing.T) {
	errs := tezkorCheck(&tezkorOrderIn{
		RestaurantID: "not-an-id",
		Items: []tezkorItemIn{
			{ID: "66d9a1b2c3d4e5f601234567", Quantity: 1.5, Price: 32000},
			{ID: "x", Quantity: 0, Price: -1},
		},
	})
	// eatsId, restaurantId, items[0].quantity, items[1].id, items[1].quantity, items[1].price
	if len(errs) != 6 {
		t.Fatalf("got %d errors, want 6: %+v", len(errs), errs)
	}
	for _, e := range errs {
		if e.Code != tezkorCodeInvalid || e.Description == "" {
			t.Fatalf("error without code or description: %+v", e)
		}
	}
	ok := tezkorCheck(&tezkorOrderIn{
		EatsID: "190330-123456", RestaurantID: "66d9a1b2c3d4e5f601234500",
		Items: []tezkorItemIn{{ID: "66d9a1b2c3d4e5f601234567", Quantity: 2, Price: 32000}},
	})
	if len(ok) != 0 {
		t.Fatalf("a valid order was refused: %+v", ok)
	}
}

// ⚠️ An unset secret must never match — the empty hash compared with the hash
// of an empty string is exactly the comparison that has to fail.
func TestTezkorSecretMatchesOnlyItsOwn(t *testing.T) {
	if tezkorSecretMatches(&models.UzumTezkorSettings{}, "") {
		t.Fatal("an unset secret matched an empty one")
	}
	s := &models.UzumTezkorSettings{SecretHash: tezkorSecretHash("s3cret")}
	if !tezkorSecretMatches(s, "s3cret") {
		t.Fatal("the right secret was refused")
	}
	if tezkorSecretMatches(s, "s3cret ") || tezkorSecretMatches(s, "") {
		t.Fatal("a wrong secret matched")
	}
}

// Uzum's format has fractional seconds, and the latest of status change and
// ready stamp is what "last moved" means.
func TestTezkorUpdatedAtFormat(t *testing.T) {
	at := time.Date(2026, 9, 14, 12, 0, 27, 870000000, time.UTC)
	ready := at.Add(time.Minute)
	o := &models.Order{
		StatusHistory: []models.StatusEvent{{Status: models.StatusPreparing, At: at}},
		ReadyAt:       &ready,
	}
	got := tezkorUpdatedAt(o)
	parsed, err := time.Parse(time.RFC3339Nano, got)
	if err != nil {
		t.Fatalf("%q is not RFC 3339: %v", got, err)
	}
	if !parsed.Equal(ready) {
		t.Fatalf("updatedAt = %s, want the ready stamp %s", parsed, ready)
	}
	if !strings.Contains(got, ".") {
		t.Fatalf("%q has no fractional seconds", got)
	}
}

// Modifications go on the line's comment, never as priced options — the line
// price already includes them.
func TestTezkorModsText(t *testing.T) {
	got := tezkorModsText([]tezkorModIn{
		{ID: "a", Name: "Achchiq sous", Quantity: 2},
		{ID: "b", Name: "", Quantity: 1},
	})
	if got != "Achchiq sous ×2, b" {
		t.Fatalf("got %q", got)
	}
	if tezkorModsText(nil) != "" {
		t.Fatal("no modifications should be an empty comment")
	}
}
