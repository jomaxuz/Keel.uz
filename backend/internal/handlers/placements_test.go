package handlers

import (
	"strings"
	"testing"
)

// ⚠️ **There is no useful sum of stock across branches**, and before this the
// arithmetic produced one anyway. "The company holds nine kilos of beef" is
// spread over three fridges in three districts: it cannot be counted, ordered
// against, or cooked from. Worse, it was filed against whichever branch's store
// the ingredient happened to name — so a chain's second kitchen could not count
// anything at all, and its consumption came off the first one's shelf.
//
// Refusing is the honest answer. A single-branch restaurant never sees it,
// because the branch is resolved for them.
func TestStockScreensInsistOnOneBranch(t *testing.T) {
	src := readSource(t, "placements.go")
	fn := between(t, src, "func (h *Handler) onlyBranch", "\n}\n")

	if !strings.Contains(fn, "if len(rows) == 1 {") {
		t.Fatal("a single-branch install no longer has its branch resolved for it")
	}
	if !strings.Contains(fn, "errPickBranch") {
		t.Fatal("a lens spanning branches no longer refuses")
	}
	// ⚠️ And no branches at all is not an error: a fresh install has an
	// ingredient list before it has a branch, and the undivided store answers.
	if !strings.Contains(fn, "if len(rows) == 0 {") {
		t.Fatal("an install with no branches yet cannot open the stock screens")
	}
}

// ⚠️ Every stock screen goes through the same resolver. A handler that kept
// using `orderScope` would quietly merge branches again, and the symptom —
// numbers that are wrong only for chains — is the kind nobody reproduces.
func TestEveryStockScreenResolvesOneBranch(t *testing.T) {
	for _, f := range []struct{ file, fn string }{
		{"stockbalance.go", "func (h *Handler) AdminStockBalances"},
		{"stockbalance.go", "func (h *Handler) AdminStockMovement"},
		{"stocktake.go", "func (h *Handler) AdminStocktakeSheet"},
		{"stocktake.go", "func (h *Handler) AdminSaveStocktake"},
		{"shoppinglist.go", "func (h *Handler) AdminShoppingList"},
		{"transfers.go", "func (h *Handler) AdminCreateTransfer"},
	} {
		body := between(t, readSource(t, f.file), f.fn, "\n}\n")
		if !strings.Contains(body, "h.stockBranch(r)") {
			t.Errorf("%s does not resolve a single branch", f.fn)
		}
	}
}

// ⚠️ **The legacy field is read by the migration and by nothing else.** Two
// sources for where a thing is kept drift, and the drift is invisible: both
// screens stay internally consistent and one of them describes the wrong
// building.
func TestWhereAThingIsKeptHasOneSource(t *testing.T) {
	for _, file := range []string{
		"stockbalance.go", "stocktake.go", "stockstop.go",
		"shoppinglist.go", "transfers.go",
	} {
		if strings.Contains(readSource(t, file), ".WarehouseID") &&
			!strings.Contains(readSource(t, file), "in.WarehouseID //") {
			// The stocktake's posted count legitimately names a store; the
			// ingredient's own field must not appear.
			for _, bad := range []string{"in.WarehouseID][", "ing.WarehouseID", "src.WarehouseID"} {
				if strings.Contains(readSource(t, file), bad) {
					t.Errorf("%s still reads the store off the ingredient (%s)", file, bad)
				}
			}
		}
	}
}
