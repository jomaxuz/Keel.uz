package handlers

import (
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson"

	"restaurant-backend/internal/models"
)

// ⚠️ **A store saved without a branch is a store nobody can see again.**
//
// `primitive.ObjectID` implements `IsZero`, so `bson:",omitempty"` drops the
// field entirely rather than writing a zero id — and a list filtered with
// `{branchId: {$in: [...]}}` does not match a document that has no such key.
// The row goes in, never comes back, and the owner creates it a second time.
//
// This is the trap that already bit twice in JSON (nil slices, zero ids); the
// BSON half is quieter, because nothing anywhere prints a warning.
func TestAStoreWithNoBranchIsWrittenWithoutTheField(t *testing.T) {
	raw, err := bson.Marshal(models.Warehouse{Name: "Bar"})
	if err != nil {
		t.Fatal(err)
	}
	var doc bson.M
	if err := bson.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["branchId"]; ok {
		t.Fatal("a zero branch id now survives into BSON — the guard below can go")
	}
}

// So the branch has to come from the scope helper, not from a type assertion on
// the filter: with a brand chosen and no branch, that filter holds a `$in`
// document and the assertion silently yields the zero id.
//
// Every sibling that files a row by branch — a delivery, a write-off, a count —
// already uses `scopeBranch`, which falls back to the default branch.
func TestCreatingAStoreFilesItUnderABranch(t *testing.T) {
	src := readSource(t, "warehouses.go")
	fn := between(t, src, "func (h *Handler) AdminCreateWarehouse", "\n}\n")

	if !strings.Contains(fn, "h.scopeBranch(r, sc)") {
		t.Fatal("a new store no longer takes its branch from scopeBranch")
	}
	if strings.Contains(fn, `scope["branchId"].(primitive.ObjectID)`) {
		t.Fatal("the branch is being read off the filter again — it is a $in there")
	}
}
