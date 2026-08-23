package handlers

import (
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
)

func shelf(unit string, store primitive.ObjectID) models.Ingredient {
	return models.Ingredient{ID: primitive.NewObjectID(), Unit: unit, WarehouseID: store}
}

// ⚠️ **The units have to match**, and this is the one refusal that could not be
// caught afterwards. A quantity moved out of a kilo into a litre leaves both
// balances arithmetically consistent and both wrong — no count, no report and
// no reconciliation downstream can see it, because nothing records what the
// number was supposed to mean.
func TestAKiloCannotBeMovedIntoALitre(t *testing.T) {
	cellar, bar := primitive.NewObjectID(), primitive.NewObjectID()

	if err := transferRefusal(shelf(models.UnitKg, cellar), shelf(models.UnitL, bar)); err == nil {
		t.Fatal("a kilo was moved into a litre")
	}
	if err := transferRefusal(shelf(models.UnitL, cellar), shelf(models.UnitL, bar)); err != nil {
		t.Fatalf("a litre could not be moved into a litre: %v", err)
	}
}

// ⚠️ **A prep item is not on a shelf as itself** — it is a pot made this
// morning, and its balance is derived from the ingredients it was made of.
// Moving one would subtract from a figure nobody holds.
func TestAPrepItemCannotBeMoved(t *testing.T) {
	cellar, bar := primitive.NewObjectID(), primitive.NewObjectID()
	sauce := shelf(models.UnitKg, cellar)
	sauce.Recipe = []models.RecipeLine{{IngredientID: primitive.NewObjectID(), Qty: 500}}
	sauce.Output = 400

	if err := transferRefusal(sauce, shelf(models.UnitKg, bar)); err != errTransferPrep {
		t.Fatalf("a sauce was moved between stores: %v", err)
	}
	if err := transferRefusal(shelf(models.UnitKg, bar), sauce); err != errTransferPrep {
		t.Fatalf("a sauce was moved into: %v", err)
	}
}

// ⚠️ Two shelves in the same store is always a mis-click. It nets to zero
// today, and stored it becomes a row somebody later tries to explain.
func TestAMoveWithinOneStoreIsRefused(t *testing.T) {
	one := primitive.NewObjectID()
	if err := transferRefusal(shelf(models.UnitKg, one), shelf(models.UnitKg, one)); err != errTransferSameStore {
		t.Fatalf("a move inside one store was accepted: %v", err)
	}
}

// ⚠️ **Both directions, never one net figure.** A shelf that received twelve
// and sent eleven away is not the shelf that received one, and the movement
// report exists to tell those apart.
func TestAMoveIsRecordedInBothDirections(t *testing.T) {
	src := readSource(t, "transfers.go")
	fn := between(t, src, "func (h *Handler) transferredInPeriod", "\n}\n")

	if !strings.Contains(fn, "out[x.FromID] += x.Qty") ||
		!strings.Contains(fn, "in[x.ToID] += x.Qty") {
		t.Fatal("a transfer no longer moves the quantity in both directions")
	}
}
