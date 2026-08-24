package handlers

import (
	"testing"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// The three stop lists are separate on purpose (CLAUDE.md, §Stop list), and the
// manual switch has to lose to the other two on the way *back*. This is the
// third screen to offer that button, and the first two learned the lesson the
// expensive way: a button that says "ok" and springs back within three minutes
// teaches a room that the software lies, and after that nobody reads any of our
// messages.
//
// Sealed here rather than in either handler because the whole point is that the
// panel and the till ask **one** function. Two copies of a rule this small is
// how the counter ends up allowed to reopen what the panel refuses.
func TestSoldOutHeldByRefusesOtherLists(t *testing.T) {
	dish := primitive.NewObjectID()
	other := primitive.NewObjectID()

	free := models.Branch{}
	if reason := soldOutHeldBy(free, dish); reason != "" {
		t.Errorf("a dish nothing holds was refused: %q", reason)
	}

	pos := models.Branch{POSSoldOut: []primitive.ObjectID{dish}}
	if reason := soldOutHeldBy(pos, dish); reason == "" {
		t.Error("a dish the till stopped was allowed back from the manual list")
	}

	// ⚠️ The stock list too, and this one was a real gap: the panel guarded
	// only the till's list, so a dish stopped by the stockroom's arithmetic
	// could be "put back" from the panel and would be stopped again by the next
	// sync. Same defect, quieter — the sweep runs on its own schedule and
	// nobody connects the two events.
	stock := models.Branch{StockSoldOut: []primitive.ObjectID{dish}}
	if reason := soldOutHeldBy(stock, dish); reason == "" {
		t.Error("a dish the stockroom stopped was allowed back from the manual list")
	}

	// A different dish being held must not block this one.
	if reason := soldOutHeldBy(models.Branch{POSSoldOut: []primitive.ObjectID{other}}, dish); reason != "" {
		t.Errorf("held the wrong dish: %q", reason)
	}
}

// ⚠️ Each refusal has to name where the real switch is. "No" on its own sends
// the cashier to the owner, the owner to the panel, and the panel refuses too —
// which is the loop this wording exists to break.
func TestSoldOutRefusalsNameTheOtherSwitch(t *testing.T) {
	dish := primitive.NewObjectID()

	posReason := soldOutHeldBy(models.Branch{POSSoldOut: []primitive.ObjectID{dish}}, dish)
	stockReason := soldOutHeldBy(models.Branch{StockSoldOut: []primitive.ObjectID{dish}}, dish)

	if posReason == stockReason {
		t.Fatal("both lists gave the same message: the two are fixed in completely different places")
	}
	for _, r := range []string{posReason, stockReason} {
		if len([]rune(r)) < 20 {
			t.Errorf("refusal too short to say what to do: %q", r)
		}
	}
}
