package handlers

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ⚠️ A dish must not recommend itself.
//
// The picker lists the whole menu, and the dish being edited is in it — so this
// is the easiest mistake in the feature to make. The result is a card offering
// the guest exactly what they are already looking at, which reads as a broken
// widget rather than a mis-set option, and makes every other suggestion on the
// page less believable.
func TestADishCannotRecommendItself(t *testing.T) {
	self := primitive.NewObjectID()
	other := primitive.NewObjectID()

	got := normalizeRecommended(self, []primitive.ObjectID{other, self, other})
	if len(got) != 1 || got[0] != other {
		t.Fatalf("cleaned to %v, want just the other dish", got)
	}
}

// Empty ids are dropped rather than stored. An unset ObjectID is twelve zero
// bytes, not an absence (§ "Bo'sh ObjectID JSON'da yo'qolmaydi"), so a form
// that submitted an empty row would otherwise store a reference to nothing —
// and the read path would spend a lookup on it every time.
func TestEmptyIdsAreDropped(t *testing.T) {
	real := primitive.NewObjectID()
	got := normalizeRecommended(primitive.NilObjectID,
		[]primitive.ObjectID{primitive.NilObjectID, real, {}})
	if len(got) != 1 || got[0] != real {
		t.Fatalf("cleaned to %v, want one real id", got)
	}
}

// The list is capped. It is stored on the dish and read on every page that
// shows it, and nothing sensible needs more than a handful.
func TestRecommendationsAreCapped(t *testing.T) {
	var ids []primitive.ObjectID
	for range 40 {
		ids = append(ids, primitive.NewObjectID())
	}
	got := normalizeRecommended(primitive.NilObjectID, ids)
	if len(got) > 12 {
		t.Fatalf("stored %d recommendations; the cap is 12", len(got))
	}
	// And the ones kept are the first ones written: the order is the owner's
	// priority, so truncating from the front would drop exactly what they put
	// at the top.
	if got[0] != ids[0] {
		t.Fatal("the cap dropped the first entry rather than the last")
	}
}

// The owner's order survives cleaning. The hand-picked list is a decision, and
// its sequence is part of the decision — the first dish is the one they most
// want offered.
func TestHandPickedOrderIsKept(t *testing.T) {
	a, b, c := primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()
	got := normalizeRecommended(primitive.NilObjectID, []primitive.ObjectID{c, a, b})
	if len(got) != 3 || got[0] != c || got[1] != a || got[2] != b {
		t.Fatalf("order changed: %v", got)
	}
}
