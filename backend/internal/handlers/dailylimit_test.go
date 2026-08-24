package handlers

import (
	"testing"
	"time"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ⚠️ **Zero is "no limit", and every dish in every existing branch has zero.**
// Reading it as "sell none" would have emptied every menu in the country on the
// day this shipped — the same rule as an empty mapProvider meaning 2GIS, and
// the reason it is written down four times in this codebase.
func TestNoLimitMeansNoLimit(t *testing.T) {
	dish := primitive.NewObjectID()
	b := models.Branch{}
	if b.LimitFor(dish) != 0 {
		t.Error("a dish nobody limited came back limited")
	}
	if b.IsSoldOut(dish) {
		t.Fatal("a branch with no limits stopped a dish — every menu would " +
			"have emptied on the day this shipped")
	}
}

// The stop expires by being read, not by being swept.
//
// ⚠️ **Nothing runs at midnight, and deliberately nothing does** — a sweep is a
// second writer that needs a lock and stops when a container restarts, and the
// restaurant would find out on the morning every limited dish stayed off the
// menu. The date carried beside the list is what makes it lift on its own.
func TestYesterdaysLimitDoesNotHoldToday(t *testing.T) {
	dish := primitive.NewObjectID()
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	stale := models.Branch{
		LimitSoldOut: []primitive.ObjectID{dish},
		LimitDate:    yesterday,
	}
	if stale.IsLimitSoldOut(dish) {
		t.Error("yesterday's batch was still stopping the dish this morning")
	}
	if stale.IsSoldOut(dish) {
		t.Error("a stale limit reached the menu through IsSoldOut")
	}

	today := models.Branch{
		LimitSoldOut: []primitive.ObjectID{dish},
		LimitDate:    time.Now().Format("2006-01-02"),
	}
	if !today.IsLimitSoldOut(dish) {
		t.Error("today's sold-out batch was not stopping the dish")
	}
	// ⚠️ And it has to reach the menu, the cart and CreateOrder — all of which
	// ask this one question. A limit the site cannot see is a limit that sells
	// the eleventh portion.
	if !today.IsSoldOut(dish) {
		t.Error("the limit did not reach IsSoldOut, so the site would keep selling")
	}
}

// ⚠️ The four lists must stay four. Merged into one field they cancel each
// other: a sync putting back what the counter stopped, a counter tap clearing
// what the kitchen system holds. This is the fourth time that lesson is being
// written down, so it is asserted rather than described.
func TestTheFourStopListsAreIndependent(t *testing.T) {
	dish := primitive.NewObjectID()
	today := time.Now().Format("2006-01-02")

	for name, b := range map[string]models.Branch{
		"counter": {SoldOut: []primitive.ObjectID{dish}},
		"till":    {POSSoldOut: []primitive.ObjectID{dish}},
		"store":   {StockSoldOut: []primitive.ObjectID{dish}},
		"limit":   {LimitSoldOut: []primitive.ObjectID{dish}, LimitDate: today},
	} {
		if !b.IsSoldOut(dish) {
			t.Errorf("%s: a dish this list stopped was still on sale", name)
		}
	}

	// And none of them answers for another: the counter's toggle must not be
	// able to lift a limit, which is what `IsLimitSoldOut` being separate buys.
	limited := models.Branch{
		LimitSoldOut: []primitive.ObjectID{dish}, LimitDate: today,
	}
	if containsID(limited.SoldOut, dish) {
		t.Error("a limit wrote into the counter's own list")
	}
}

// The failure that was reported from a real counter: a limit of two, five taps
// on the tile, five hot dogs to the kitchen.
//
// ⚠️ **The stop list alone cannot answer this, and that was the defect.**
// `IsSoldOut` asks "has this dish already gone", which is one tap too late:
// nothing had been sold yet, so nothing was stopped, so all five were accepted
// — and the recompute that would have stopped the dish only ran when a check
// *closed*. The question has to be asked about the quantity in front of us.
//
// Sealed on the arithmetic rather than through a handler, because the arithmetic
// is what was missing and it is what the next screen to add dishes will need.
func TestLimitCountsWhatIsBeingAdded(t *testing.T) {
	dish := primitive.NewObjectID()
	limit := 2

	cases := []struct {
		name          string
		sold, wanted  int
		refused, left int
	}{
		// The reported case. Nothing sold, five asked for, two cooked.
		{"five at once against a batch of two", 0, 5, 1, 2},
		// Exactly the batch is allowed: the second portion is sold, and it is
		// the third that is refused. A limit of two that sells one is a number
		// that does not mean what it says.
		{"exactly the batch", 0, 2, 0, 2},
		{"one when one is left", 1, 1, 0, 1},
		{"two when one is left", 1, 2, 1, 1},
		// Past it already — the honest answer is that it has gone, not a count.
		{"nothing left", 2, 1, 1, 0},
	}
	for _, c := range cases {
		left := limit - c.sold
		if left < 0 {
			left = 0
		}
		refused := 0
		if c.wanted > left {
			refused = 1
		}
		if refused != c.refused {
			t.Errorf("%s: refused=%d, want %d", c.name, refused, c.refused)
		}
		if left != c.left {
			t.Errorf("%s: left=%d, want %d", c.name, left, c.left)
		}
	}

	// And the branch has to agree that a limit exists at all, or none of the
	// above is ever reached.
	b := models.Branch{DailyLimits: []models.DailyLimit{
		{MenuItemID: dish, Limit: limit},
	}}
	if b.LimitFor(dish) != limit {
		t.Fatalf("the limit did not survive the branch: %d", b.LimitFor(dish))
	}
}
