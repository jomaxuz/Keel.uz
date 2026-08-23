package handlers

import (
	"strings"
	"testing"
)

// ⚠️ **A debt does not belong to a period.** Everything else on the supplier
// page is measured over the window the owner chose; what is owed is not. A
// March invoice is still a debt in May, and a figure that clears itself when
// the month rolls over is not a debt at all — the same rule the courier's cash
// in hand follows, arrived at for the same reason.
func TestWhatIsOwedIgnoresTheChosenPeriod(t *testing.T) {
	src := readSource(t, "suppliers.go")
	fn := between(t, src, "func (h *Handler) AdminSupplierReport", "\n}\n")

	owed := fn[strings.Index(fn, "owedFilter :="):]
	if strings.Contains(owed, `"at"`) {
		t.Fatal("the debt total is being filtered by the period — it clears itself monthly")
	}
	if !strings.Contains(owed, `"paid": bson.M{"$ne": true}`) {
		t.Fatal("unpaid deliveries are no longer selected by the absent-or-false rule")
	}
}

// ⚠️ **`$ne: true`, not `false`.** The field is missing on every delivery
// entered before invoices could be unpaid, and in Mongo a missing field matches
// `$ne` — which is exactly what is wanted here, and is also why the migration
// has to settle those rows first. Written down because the same trap has
// already been hit the other way round (`pendingTillFilter` needed `$nin`).
func TestSettlingOldDeliveriesRunsOnceOnTheMissingField(t *testing.T) {
	src := readSourceIn(t, "../repository", "migrate.go")
	fn := between(t, src, "func EnsureDeliveriesSettled", "\n}\n")

	if !strings.Contains(fn, `bson.M{"paid": bson.M{"$exists": false}}`) {
		t.Fatal("the migration no longer matches only deliveries that predate the field")
	}
	if strings.Contains(fn, `"paid": false`) {
		t.Fatal("the migration would re-settle an invoice somebody marked unpaid")
	}
}

// ⚠️ Marking an invoice paid is guarded by the unpaid filter, never by id
// alone: "paid" pressed on two screens must settle it once, and an invoice
// already settled has to answer 404 rather than writing a second date over the
// first. The same rule a guest's slate follows.
func TestAnInvoiceIsSettledOnce(t *testing.T) {
	src := readSource(t, "suppliers.go")
	fn := between(t, src, "func (h *Handler) AdminPayPurchase", "\n}\n")

	if !strings.Contains(fn, `"paid": bson.M{"$ne": true}`) {
		t.Fatal("an already-settled invoice can be settled again")
	}
}
