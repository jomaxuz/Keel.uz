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
