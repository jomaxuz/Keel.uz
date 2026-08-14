package handlers

import (
	"testing"

	"restaurant-backend/internal/models"
)

// ⚠️ The zero value must be today's screen.
//
// Every admin account that already exists has no preferences. A default that
// meant "show nothing", or a stored allowlist that started empty, would blank
// the dashboard for every admin of every install on the day this shipped —
// and it would look like the update had broken the panel, not like a setting.
// Same rule as `hidePlan` (§ "Stol bron qilish").
func TestNoPreferencesIsTodaysDashboard(t *testing.T) {
	got := resolveDashboard(models.DashboardPrefs{})
	if len(got) != len(dashboardTileIDs) {
		t.Fatalf("%d tiles for an admin with no preferences, want all %d",
			len(got), len(dashboardTileIDs))
	}
	for i, id := range dashboardTileIDs {
		if got[i] != id {
			t.Fatalf("tile %d is %q, want %q — the default order must be untouched", i, got[i], id)
		}
	}
}

// A tile added in a later version appears for everybody, because the stored
// list names what is off. A show-list would hide every new figure from every
// existing account, forever, with nothing on screen to explain it.
func TestHidingIsOptOutNotOptIn(t *testing.T) {
	prefs := models.DashboardPrefs{Hidden: []string{"menu.dishes"}}
	got := resolveDashboard(prefs)

	if len(got) != len(dashboardTileIDs)-1 {
		t.Fatalf("%d tiles, want %d", len(got), len(dashboardTileIDs)-1)
	}
	for _, id := range got {
		if id == "menu.dishes" {
			t.Fatal("a hidden tile was drawn")
		}
	}
	// Everything else survived, including tiles the preference never mentions.
	if !hasTile(got, "orders.total") || !hasTile(got, "menu.categories") {
		t.Fatal("hiding one tile removed others")
	}
}

// A partial order moves the tiles it names and leaves the rest alone. An admin
// who dragged two tiles to the top is not also deciding the order of the
// eighteen they never touched.
func TestPartialOrderKeepsTheRestInPlace(t *testing.T) {
	got := resolveDashboard(models.DashboardPrefs{
		Order: []string{"money.revenue", "orders.cancelled"},
	})
	if got[0] != "money.revenue" || got[1] != "orders.cancelled" {
		t.Fatalf("first two tiles %v, want the two that were named", got[:2])
	}
	if len(got) != len(dashboardTileIDs) {
		t.Fatalf("%d tiles, want all %d — ordering must not drop any",
			len(got), len(dashboardTileIDs))
	}
	// The unnamed ones keep their relative default order behind them.
	rest := got[2:]
	var expected []string
	for _, id := range dashboardTileIDs {
		if id != "money.revenue" && id != "orders.cancelled" {
			expected = append(expected, id)
		}
	}
	for i := range expected {
		if rest[i] != expected[i] {
			t.Fatalf("unnamed tile %d is %q, want %q", i, rest[i], expected[i])
		}
	}
}

// ⚠️ The submitted list is narrowed to ids the server knows.
//
// Without it, `hidden` is an unbounded array of arbitrary strings written
// straight onto the account document — a client can grow it until the document
// is a problem, and nothing in the panel would ever show it.
func TestUnknownTilesAreDropped(t *testing.T) {
	got := cleanTileIDs([]string{"orders.total", "not-a-tile", "", "orders.total"})
	if len(got) != 1 || got[0] != "orders.total" {
		t.Fatalf("cleaned to %v, want [orders.total] — unknown ids and duplicates dropped", got)
	}
}

// An empty preference marshals as `[]`, never `null`.
//
// The panel does `hidden.includes(...)`; Go turns a nil slice into `null`, and
// `null.includes` takes the settings screen down with a React error. This trap
// has bitten this codebase twice already, which is why the fix is a function
// with a test rather than a habit.
func TestEmptyPrefsAreArraysNotNull(t *testing.T) {
	if nonNil(nil) == nil {
		t.Fatal("a nil slice reached the response and will marshal as null")
	}
	if len(nonNil(nil)) != 0 {
		t.Fatal("nonNil invented an entry")
	}
}

func hasTile(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
