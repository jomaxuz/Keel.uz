package handlers

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
)

func lineOf(name string, qty int, fired bool) models.OrderItem {
	it := models.OrderItem{
		MenuItemID: primitive.NewObjectID(),
		Name:       name,
		Qty:        qty,
		LineID:     name + "-1",
	}
	if fired {
		at := time.Now()
		it.FiredAt = &at
	}
	return it
}

func checkOrder(items ...models.OrderItem) *models.Order {
	return &models.Order{
		ID:    primitive.NewObjectID(),
		Items: items,
		Check: &models.OrderCheck{OpenedAt: time.Now()},
	}
}

// ⚠️ **The question the whole "write it off when it is rung up" choice turns
// on.** A cashier taps a tile, sees it was the wrong one and removes it: the
// kitchen was never told, no pan was touched, and the shelf must be exactly
// where it was. The till already drops such a line from the check outright —
// there is nothing left to want, so the row written a second earlier is
// reversed and the balance is unchanged.
func TestALineRemovedBeforeTheKitchenSawItWantsNothing(t *testing.T) {
	o := checkOrder(lineOf("Osh", 1, false))
	// What the till does on an unfired void: the line is gone from the check.
	o.Items = nil

	if got := wantedMovements(o); len(got) != 0 {
		t.Fatalf("a mis-tap still takes food off the shelf: %+v", got)
	}
}

// ⚠️ **And the other half, which is the one that must not be "fixed".** A dish
// voided after it was fired was cooked. Putting its ingredients back would file
// real waste as a correction — the shelf would agree with the books, the books
// would be wrong, and a count would find nothing because nothing is missing
// against a record that expects it gone.
func TestAVoidedDishThatWasCookedStaysOffTheShelf(t *testing.T) {
	cooked := lineOf("Lag'mon", 2, true)
	cooked.Void = &models.CheckLineVoid{
		At: time.Now(), Reason: "mehmon qaytardi", Wasted: true,
	}
	got := wantedMovements(checkOrder(cooked))

	if len(got) != 1 {
		t.Fatalf("cooked food came back to the shelf: %+v", got)
	}
	if !got[0].wasted {
		t.Fatal("it is still on the shelf's books but not marked as waste — the waste report cannot see it")
	}
	if got[0].qty != 2 {
		t.Fatalf("qty=%v, wanted 2", got[0].qty)
	}
}

// The kitchen caught it at the pass: told to cook it, had not started. Nothing
// left the store, so nothing stays written.
func TestAVoidedDishTheKitchenCaughtInTimeIsReturned(t *testing.T) {
	caught := lineOf("Lag'mon", 1, true)
	caught.Void = &models.CheckLineVoid{
		At: time.Now(), Reason: "noto'g'ri yozilgan", Wasted: false,
	}

	if got := wantedMovements(checkOrder(caught)); len(got) != 0 {
		t.Fatalf("food nobody made is still written off: %+v", got)
	}
}

// ⚠️ **A cancelled check is voiding all of it, line by line, and the same test
// applies.** A table opened by mistake cooked nothing; a check cancelled with
// food on the pass did. One product must not have two answers to "was anything
// cooked" — `cookedValue` already draws this line for the manager's PIN and for
// the loss alert.
func TestCancellingACheckSplitsOnWhatTheKitchenWasTold(t *testing.T) {
	o := checkOrder(lineOf("Osh", 1, true), lineOf("Choy", 3, false))
	o.Status = models.StatusCancelled
	o.CancelReason = "mehmon ketdi"

	got := wantedMovements(o)
	if len(got) != 1 {
		t.Fatalf("wanted only the fired line, got %d: %+v", len(got), got)
	}
	if got[0].item.Name != "Osh" {
		t.Fatalf("the wrong line survived: %s", got[0].item.Name)
	}
	if !got[0].wasted || got[0].reason != "mehmon ketdi" {
		t.Fatalf("a cancelled cooked dish is not recorded as waste: %+v", got[0])
	}
}

// ⚠️ **A merged check is cancelled and keeps its lines on purpose** — they are
// the record of where the table's food went. Read as an ordinary cancellation
// the food would be counted twice: once on the check that now carries it, and
// once here as waste. The shelf would then come up short by a whole merged
// table every time two bills were joined, and nothing would say why.
func TestAMergedCheckReleasesItsLinesRatherThanBinningThem(t *testing.T) {
	o := checkOrder(lineOf("Osh", 2, true))
	o.Status = models.StatusCancelled
	o.MergedIntoID = primitive.NewObjectID()
	o.CancelReason = "birlashtirildi → A-12"

	if got := wantedMovements(o); len(got) != 0 {
		t.Fatalf("merged food is taken off the shelf twice: %+v", got)
	}
}

// ⚠️ Half a portion takes half the card. The map counts fractions for exactly
// this reason: while it was integers the store was told a half was a whole, and
// the shelf came up short by the halves nobody counted, once a month.
func TestHalfAPortionWantsHalfTheCard(t *testing.T) {
	half := lineOf("Non", 3, false)
	half.Portion = 50

	got := wantedMovements(checkOrder(half))
	if len(got) != 1 || got[0].qty != 1.5 {
		t.Fatalf("qty=%+v, wanted 1.5", got)
	}
}

// ⚠️ **A check is edited all evening and a website order never is.** The till
// mints a line id precisely because two guests ordering the same dish are two
// lines; a web order has none, and its position cannot shift under us. Keying a
// check on position would attach a movement to whatever line slid into the slot
// when an earlier one was removed.
func TestLinesAreKeyedByTheirOwnIdWhereverThereIsOne(t *testing.T) {
	withID := models.OrderItem{LineID: "abc"}
	if got := lineKeyOf(withID, 7); got != "abc" {
		t.Fatalf("key=%q, wanted the line's own id", got)
	}
	if got := lineKeyOf(models.OrderItem{}, 7); got != "#7" {
		t.Fatalf("key=%q, wanted the position", got)
	}
}

// ⚠️ **A till that was offline sends a week of checks the moment it
// reconnects.** Stamped with the write, the whole backlog lands on one
// afternoon — and the stocktakes either side of the gap are then wrong in
// opposite directions, which is the shape of an error nobody traces back to a
// network cable.
func TestARowIsStampedWithTheSaleNotTheWrite(t *testing.T) {
	sold := time.Now().AddDate(0, 0, -3)
	o := &models.Order{CreatedAt: sold}

	if got := movementTime(o, models.OrderItem{}); !got.Equal(sold) {
		t.Fatalf("stamped %v, wanted the order's own time %v", got, sold)
	}
	// Where the kitchen was told separately, that is the truer moment still.
	fired := sold.Add(20 * time.Minute)
	if got := movementTime(o, models.OrderItem{FiredAt: &fired}); !got.Equal(fired) {
		t.Fatalf("stamped %v, wanted the moment it was fired", got)
	}
}

// A website order has no firing step: the kitchen sees the whole of it at once,
// so the order's own queue time is what says whether anything was cooked.
func TestAWebOrderIsJudgedByTheWholeOrderNotByLines(t *testing.T) {
	o := &models.Order{Items: []models.OrderItem{lineOf("Osh", 1, false)}}
	if cookedLine(o, o.Items[0]) {
		t.Fatal("an unqueued web order counts as cooked")
	}
	now := time.Now()
	o.QueuedAt = &now
	if !cookedLine(o, o.Items[0]) {
		t.Fatal("a queued web order does not count as cooked")
	}
}
