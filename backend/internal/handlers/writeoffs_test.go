package handlers

import (
	"strings"
	"testing"
	"time"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ⚠️ **Valued at the prices of the day it happened, and frozen.** This is the
// figure that changes behaviour — "3 200 000 thrown away this month" moves an
// owner in a way "eleven write-offs" does not — and it must not shift under
// them when the next delivery arrives at a different price.
func TestWriteOffIsValuedOnTheDayItHappened(t *testing.T) {
	march := time.Date(2026, 3, 15, 0, 0, 0, 0, time.Local)
	beef := models.Ingredient{
		Unit: models.UnitKg, Price: 120000,
		History: []models.PriceEntry{
			{Price: 90000, At: time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)},
			{Price: 120000, At: time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)},
		},
	}
	if got := writeOffValue(beef, 2, march, nil); got != 180000 {
		t.Fatalf("value=%d, want 180000 (March's price)", got)
	}
}

// ⚠️ A prep item was never bought, so it has no price of its own — and refusing
// to value it would record the most expensive thing a kitchen throws away, a
// ruined batch of sauce, as costing nothing.
func TestARuinedBatchIsWorthWhatItCostToMake(t *testing.T) {
	id := primitive.NewObjectID()
	sauce := models.Ingredient{
		ID: id, Unit: models.UnitKg, Output: 2000,
		Recipe: []models.RecipeLine{{IngredientID: primitive.NewObjectID(), Qty: 3000}},
	}
	// 18 so'm per gram of sauce; three kilos thrown away.
	got := writeOffValue(sauce, 3, time.Now(), map[primitive.ObjectID]float64{id: 18})
	if got != 54000 {
		t.Fatalf("value=%d, want 54000", got)
	}
}

// ⚠️ A reason is required, as everywhere else in this system where something
// disappears: a void, a refund, a cancelled order. "Twelve kilos of beef,
// written off" with no sentence beside it is the line every argument starts
// from, and by then the person who could answer has gone home.
func TestAWriteOffWithoutAReasonIsRefused(t *testing.T) {
	src := readSource(t, "writeoffs.go")
	fn := between(t, src, "func (h *Handler) AdminCreateWriteOff", "\n}\n")

	if !strings.Contains(fn, `in.Reason == ""`) {
		t.Fatal("food can be written off with no explanation")
	}
	// Never into the future: it would sit outside every report until the day
	// arrives and then change a month already read.
	if !strings.Contains(fn, "in.At.After(now)") {
		t.Fatal("a write-off can be dated forward")
	}
}

// ⚠️ Write-offs come off the difference — that is the point of recording them —
// and stay in their own column: one figure is what the cards say the dishes
// took, the other is what somebody wrote down, and merging them hides which of
// the two a gap came from.
func TestWriteOffsCloseTheGapWithoutHidingWhereItWent(t *testing.T) {
	src := readSource(t, "stockreport.go")
	fn := between(t, src, "func (h *Handler) AdminStockReport", "\n}\n")

	if !strings.Contains(fn, "got - out - off") {
		t.Fatal("write-offs no longer come off the unexplained difference")
	}
	if !strings.Contains(fn, "Written: round3(off)") {
		t.Fatal("write-offs have been folded into the usage column")
	}
	if !strings.Contains(stockNote("uz"), "hech kim tushuntirmagan") {
		t.Fatal("the note no longer says what the difference is")
	}
}
