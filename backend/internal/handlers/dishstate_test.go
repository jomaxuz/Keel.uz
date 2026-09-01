package handlers

import (
	"testing"
	"time"

	"restaurant-backend/internal/models"
)

// The order-wide "ready" is derived from the lines, and it is the flag that
// decides whether a ticket is on the pass at all. Getting it wrong in one
// direction hides food nobody has cooked; in the other it leaves a finished
// ticket on the screen for ever. Both are arithmetic, so both are tested here
// rather than in a dining room.

// agoMin is a timestamp that many minutes in the past. Named for what it
// means rather than `at`, which this package already uses for a fixed clock in
// the pre-order tests.
func agoMin(min int) *time.Time {
	t := time.Now().Add(time.Duration(-min) * time.Minute)
	return &t
}

func TestAllDishesReadyIgnoresWhatTheKitchenDoesNotOwn(t *testing.T) {
	fired := agoMin(30)
	cases := []struct {
		name  string
		items []models.OrderItem
		want  bool
	}{
		{
			// A website order: no line was ever "fired", every live line counts.
			name: "delivery order, one dish left",
			items: []models.OrderItem{
				{Name: "Osh", ReadyAt: agoMin(2)},
				{Name: "Somsa"},
			},
		},
		{
			name: "delivery order, all ticked",
			items: []models.OrderItem{
				{Name: "Osh", ReadyAt: agoMin(2)},
				{Name: "Somsa", ReadyAt: agoMin(1)},
			},
			want: true,
		},
		{
			// ⚠️ The mains are still a draft on the waiter's tablet. A check
			// whose starters are done is ready as far as the kitchen is
			// concerned — counting the unfired half would keep the ticket on
			// the pass with nothing on it to cook.
			name: "till check, unfired course does not count",
			items: []models.OrderItem{
				{Name: "Salat", LineID: "a", FiredAt: fired, ReadyAt: agoMin(1)},
				{Name: "Kabob", LineID: "b"},
			},
			want: true,
		},
		{
			// ⚠️ A voided line was cancelled precisely so it would not be made.
			name: "voided line does not hold the ticket open",
			items: []models.OrderItem{
				{Name: "Salat", LineID: "a", FiredAt: fired, ReadyAt: agoMin(1)},
				{Name: "Kabob", LineID: "b", FiredAt: fired, Void: &models.CheckLineVoid{Reason: "mijoz bekor qildi"}},
			},
			want: true,
		},
		{
			// ⚠️ Nothing cookable is not "ready": there is nothing to be ready.
			// Answering true here would fire the waiter's notification for a
			// table whose only dish had just been sent back.
			name:  "everything voided is not ready",
			items: []models.OrderItem{{Name: "Kabob", LineID: "a", FiredAt: fired, Void: &models.CheckLineVoid{Reason: "x"}}},
		},
		{name: "no lines at all", items: nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := allDishesReady(c.items); got != c.want {
				t.Fatalf("allDishesReady = %v, want %v", got, c.want)
			}
		})
	}
}

// ⚠️ The ticket-wide button has to leave the lines in the state the floor will
// read: a ticket marked ready with no dish on it green is the contradiction
// that makes a room stop trusting the colour.
func TestMarkAllDishesReadyStampsWhatTheKitchenOwns(t *testing.T) {
	fired := agoMin(20)
	already := agoMin(9)
	items := []models.OrderItem{
		{Name: "Salat", LineID: "a", FiredAt: fired, ReadyAt: already},
		{Name: "Kabob", LineID: "b", FiredAt: fired},
		{Name: "Choy", LineID: "c"}, // still a draft
		{Name: "Lag'mon", LineID: "d", FiredAt: fired, Void: &models.CheckLineVoid{Reason: "x"}},
	}
	now := time.Now()
	markAllDishesReady(items, now)

	if items[0].ReadyAt != already {
		// ⚠️ An earlier tick keeps its own time. Overwriting it would tell the
		// floor a dish that has been standing for nine minutes was just cooked.
		t.Fatalf("an already-ready dish was re-stamped")
	}
	if items[1].ReadyAt == nil {
		t.Fatalf("a fired dish was left unstamped")
	}
	if items[2].ReadyAt != nil {
		t.Fatalf("an unfired draft was marked cooked")
	}
	if items[3].ReadyAt != nil {
		t.Fatalf("a voided line was marked cooked")
	}
	if !allDishesReady(items) {
		t.Fatalf("the ticket-wide button left the order not ready")
	}
}

// ⚠️ Two kinds of order reach the pass and they are named differently. Mixing
// the two addresses is how a tick lands on the wrong dish — which the cook sees
// as the screen moving under their hand.
func TestDishRefNamesTheRightLine(t *testing.T) {
	items := []models.OrderItem{
		{Name: "Osh", LineID: "a"},
		{Name: "Somsa", LineID: "b"},
	}
	if got := (dishRef{LineID: "b"}).find(items); got != 1 {
		t.Fatalf("by line id: got %d", got)
	}
	if got := (dishRef{LineID: "zzz"}).find(items); got != -1 {
		t.Fatalf("an unknown line id addressed something: %d", got)
	}
	// ⚠️ A position may not address a line that has an id: a check that was
	// split a second ago would be edited at the wrong row.
	i := 0
	if got := (dishRef{Index: &i}).find(items); got != -1 {
		t.Fatalf("an index reached a till line: %d", got)
	}

	web := []models.OrderItem{{Name: "Osh"}, {Name: "Somsa"}}
	if got := (dishRef{Index: &i}).find(web); got != 0 {
		t.Fatalf("by index: got %d", got)
	}
	out := 7
	if got := (dishRef{Index: &out}).find(web); got != -1 {
		t.Fatalf("an index past the end addressed something: %d", got)
	}
	if got := (dishRef{}).find(web); got != -1 {
		t.Fatalf("an empty reference addressed something: %d", got)
	}
}

// The floor screen's badge: what is standing at the pass, and what has already
// gone out.
func TestServedCountSeparatesWaitingFromDelivered(t *testing.T) {
	fired := agoMin(15)
	served, waiting := servedCount([]models.OrderItem{
		{LineID: "a", FiredAt: fired, ReadyAt: agoMin(5), ServedAt: agoMin(1)},
		{LineID: "b", FiredAt: fired, ReadyAt: agoMin(4)},
		{LineID: "c", FiredAt: fired}, // still cooking
		{LineID: "d"},                 // not sent
		{LineID: "e", FiredAt: fired, ReadyAt: agoMin(3), Void: &models.CheckLineVoid{Reason: "x"}},
	})
	if served != 1 || waiting != 1 {
		t.Fatalf("served=%d waiting=%d, want 1 and 1", served, waiting)
	}
}
