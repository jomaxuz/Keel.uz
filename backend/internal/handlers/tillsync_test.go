package handlers

import (
	"testing"
	"time"

	"restaurant-backend/internal/models"
)

// ⚠️ **The monoblock's clock is the quietest thing that breaks in a till.** On
// an older machine the CMOS battery is dead, and a power cut sends the date back
// years — the till then stamps an evening of sales into a year the restaurant
// did not exist, and no screen anywhere says so. It reaches the reports, the
// shift count, and on a fiscal receipt a tax document.
func TestOfflineTimesAreClamped(t *testing.T) {
	now := time.Date(2026, 8, 18, 20, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		in   time.Time
		want time.Time
	}{
		{
			"an ordinary offline sale keeps its own time",
			now.Add(-90 * time.Minute),
			now.Add(-90 * time.Minute),
		},
		{
			// The whole point: an evening's sales must not be stamped with the
			// minute the connection came back, or the hourly report says the
			// restaurant sold a day's food in sixty seconds.
			"a sale from this morning keeps its own time",
			now.Add(-11 * time.Hour),
			now.Add(-11 * time.Hour),
		},
		{
			// A dead CMOS battery: the date falls back years.
			"a clock that fell back to 2010 is refused",
			time.Date(2010, 1, 1, 12, 0, 0, 0, time.UTC),
			now,
		},
		{
			// Nothing is sold tomorrow.
			"a clock running ahead is refused",
			now.Add(48 * time.Hour),
			now,
		},
		{
			// ⚠️ A small skew is kept: till clocks drift by seconds and
			// snapping those to "now" would move every sale to the moment it
			// synced, which is the failure this function exists to prevent.
			"a minute of drift is tolerated",
			now.Add(time.Minute),
			now.Add(time.Minute),
		},
		{"a missing time becomes now", time.Time{}, now},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := clampOfflineTime(c.in, now); !got.Equal(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}

// ⚠️ **Queued when the kitchen was told, not when we heard.** Every report that
// asks how long a table waited reads this, and stamping it at sync time would
// make an evening's cooking look instantaneous.
func TestQueuedAtIsWhenTheKitchenWasTold(t *testing.T) {
	open := time.Date(2026, 8, 18, 19, 0, 0, 0, time.UTC)
	first := open.Add(4 * time.Minute)
	second := open.Add(40 * time.Minute)

	items := []models.OrderItem{
		{FiredAt: &second},
		{FiredAt: &first},
		{}, // never fired — a line still being typed when the guest paid
	}
	got := firstFired(items, open)
	if got == nil || !got.Equal(first) {
		t.Fatalf("got %v, want the first course at %v", got, first)
	}

	// A check nobody ever sent to the kitchen — a bottle of water at the
	// counter — is the restaurant's from the moment it was opened.
	if got := firstFired([]models.OrderItem{{}}, open); !got.Equal(open) {
		t.Fatalf("got %v, want %v", got, open)
	}
}
