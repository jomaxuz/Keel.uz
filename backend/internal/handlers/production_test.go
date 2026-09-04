package handlers

import (
	"strings"
	"testing"
)

// ⚠️ **Both halves of one document, and neither works alone.** A batch puts a
// prep item on the shelf and takes its inputs off. Count only the output and a
// central kitchen becomes a machine that creates sauce out of nothing; count
// only the inputs and it becomes a write-off with no waste. Both failures are
// arithmetically consistent, which is why they would be found at a count weeks
// later and blamed on whoever counted.
func TestABatchIsBothAnInAndAnOut(t *testing.T) {
	src := readSource(t, "stocktake.go")
	fn := between(t, src, "func (h *Handler) expectedStockByWarehouse", "\n}\n")

	if !strings.Contains(fn, "producedInPeriod") {
		t.Fatal("a batch no longer reaches the balance at all")
	}
	if !strings.Contains(fn, "add(batched, 1)") {
		t.Fatal("what a batch made is not being added to the store")
	}
	if !strings.Contains(fn, "add(batchTook, -1)") {
		t.Fatal("what a batch consumed is not being taken off the store")
	}
}

// ⚠️ **The expansion has to stop at a batched item**, and this is the whole of
// the central-kitchen change on the consumption side. A dish using sauce that
// arrived in a tub took the sauce; the tomatoes were never in this branch, and
// reading them as used would drive a shelf negative on something nobody has.
func TestABatchedPrepItemIsConsumedAsItself(t *testing.T) {
	src := readSource(t, "stockreport.go")
	fn := between(t, src, "func rawInputs", "\n}\n")

	if !strings.Contains(fn, "in.DerivedOnly()") {
		t.Fatal("the expansion no longer distinguishes a batch from a derived item")
	}
	if strings.Contains(fn, "in.MadeInHouse()") {
		t.Fatal("a batched item is being expanded into its inputs again")
	}
}

// ⚠️ **A batch may only be made where somebody cooked it**, and only of an item
// that is kept on a shelf. Producing a prep item that is not batched subtracts
// its inputs here *and* through the card when a dish is sold; producing in an
// ordinary store takes tomatoes off shelves in another building.
func TestABatchIsRefusedWhereItCouldNotHaveHappened(t *testing.T) {
	src := readSource(t, "production.go")
	fn := between(t, src, "func (h *Handler) AdminCreateProduction", "\n}\n")

	if !strings.Contains(fn, "!item.MadeInHouse() || !item.Batched") {
		t.Fatal("a batch can be made of something that is not a batched prep item")
	}
	if !strings.Contains(fn, "store.IsProduction()") {
		t.Fatal("a batch can be made in a store that is not a central kitchen")
	}
}

// ⚠️ **What a batch took is frozen onto it.** The card is corrected as recipes
// change, and a batch made in March took what it took in March — re-deriving it
// later rewrites a month that has already been counted and reconciled.
func TestWhatABatchTookIsWrittenDown(t *testing.T) {
	src := readSource(t, "production.go")
	fn := between(t, src, "func (h *Handler) AdminCreateProduction", "\n}\n")

	if !strings.Contains(fn, "in.Lines = lines") {
		t.Fatal("a batch no longer records what it consumed")
	}
	if !strings.Contains(fn, "in.Value = value") {
		t.Fatal("a batch no longer records what it was worth")
	}
}

// ⚠️ **The movement report has to count every fact the balance counts.** Its
// opening and closing figures are `expectedStockByWarehouse` run to two dates,
// and that function counts batches; the columns between them did not. So in any
// restaurant with a central kitchen the one screen an owner opens *to explain* a
// shortfall produced a shortfall of its own — of exactly the size of everything
// the tsex had made that month — and the file's own comment promised the two
// screens could never disagree.
func TestTheMovementReportCountsBatchesToo(t *testing.T) {
	src := readSource(t, "stockbalance.go")
	fn := between(t, src, "func (h *Handler) AdminStockMovement", "\n}\n")

	if !strings.Contains(fn, "producedInPeriod") {
		t.Fatal("the movement report does not read batches — its columns cannot reach its own closing figure")
	}
	for _, field := range []string{`"produced"`, `"producedUsed"`} {
		if !strings.Contains(fn, field) {
			t.Fatalf("%s is not reported — half a batch is a different lie than none of it", field)
		}
	}
	// The documents, not only the totals: a column with nothing behind it is
	// the gap this report exists to close.
	if !strings.Contains(fn, "h.Store.Productions.Find") {
		t.Fatal("batches are totalled but not listed — the reader cannot open what moved")
	}
}
