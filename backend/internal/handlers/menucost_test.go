package handlers

import (
	"encoding/json"
	"strings"
	"testing"

	"restaurant-backend/internal/models"
)

// ⚠️ **What a plate costs the kitchen must never reach the public API**, and
// the only reliable way to guarantee that is for the model to refuse to
// serialise it. A dish goes out through a dozen handlers — the menu, one dish,
// the recommendations, favourites, the basket, combo contents, the mini app —
// and a guard added to each of them is a guard missing from the thirteenth.
func TestTheCostNeverLeavesThroughADish(t *testing.T) {
	raw, err := json.Marshal(models.MenuItem{Name: "Lag'mon", Price: 42000, Cost: 15000})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "15000") || strings.Contains(string(raw), "cost") {
		t.Fatalf("a dish carries its cost to whoever asks: %s", raw)
	}
	// ...and the panel's own shape does carry it, or the field could never be
	// filled in.
	panelRaw, _ := json.Marshal(withCost(models.MenuItem{Cost: 15000}))
	if !strings.Contains(string(panelRaw), `"cost":15000`) {
		t.Fatalf("the panel cannot read the cost back: %s", panelRaw)
	}
}

// ⚠️ A form that does not send the field keeps what is stored. `UpdateMenuItem`
// replaces the whole document, so an older tab saving a dish's name would
// otherwise erase every cost somebody typed — the same trap `soldOut` and
// `kioskSecret` are guarded against.
func TestASaveWithoutTheFieldKeepsTheStoredCost(t *testing.T) {
	src := readSource(t, "admin.go")
	fn := between(t, src, "func (h *Handler) UpdateMenuItem", "\n}\n")

	if !strings.Contains(fn, "h.keepCost(") {
		t.Fatal("saving a dish can wipe its cost")
	}
	if !strings.Contains(fn, "in.Cost") {
		t.Fatal("the cost is no longer read from the panel's own field")
	}
}
