package handlers

import (
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
)

// ⚠️ **Withdrawing by delivery, never by date.** Two invoices can land on one
// day, and removing an entry by its date would take the other one's claim with
// it — surfacing weeks later as a dish that quietly changed price in a month
// nobody edited.
func TestCorrectingOneInvoiceLeavesTheOtherAlone(t *testing.T) {
	day := time.Date(2026, 3, 4, 9, 0, 0, 0, time.UTC)
	mine, theirs := primitive.NewObjectID(), primitive.NewObjectID()

	history := []models.PriceEntry{
		{Price: 20000, At: day.Add(-48 * time.Hour)},
		{Price: 24000, At: day, PurchaseID: mine},
		{Price: 26000, At: day, PurchaseID: theirs},
	}

	kept := make([]models.PriceEntry, 0, len(history))
	for _, e := range history {
		if e.PurchaseID == mine {
			continue
		}
		kept = append(kept, e)
	}

	if len(kept) != 2 {
		t.Fatalf("withdrawing one invoice removed %d entries", len(history)-len(kept))
	}
	for _, e := range kept {
		if e.PurchaseID == mine {
			t.Fatal("the corrected invoice's claim survived")
		}
	}
	if kept[len(kept)-1].PurchaseID != theirs {
		t.Fatal("the other invoice's claim was the one removed")
	}
}

// ⚠️ **A hand edit belongs to nobody and is never withdrawn.** Entries written
// before deliveries carried an id, and every price somebody typed deliberately,
// have no invoice behind them — matching them would silently delete a
// correction that was made on purpose.
func TestAHandEditIsNeverWithdrawn(t *testing.T) {
	src := readSource(t, "purchases.go")
	fn := between(t, src, "func (h *Handler) withdrawDeliveryPrices", "\n}\n")

	if !strings.Contains(fn, "if p.ID.IsZero() {") {
		t.Fatal("an invoice with no id could withdraw every untagged entry")
	}
	if !strings.Contains(fn, "e.PurchaseID == p.ID") {
		t.Fatal("entries are no longer matched on the delivery that wrote them")
	}
}

// ⚠️ **The headline price never falls to zero.** When nothing survives, it
// stays where it is: zeroing it makes every dish containing the ingredient cost
// nothing, which reads on a margin report as very good news.
func TestWithdrawingEverythingDoesNotZeroThePrice(t *testing.T) {
	src := readSource(t, "purchases.go")
	fn := between(t, src, "func (h *Handler) withdrawDeliveryPrices", "\n}\n")

	if !strings.Contains(fn, "if len(kept) > 0 {") {
		t.Fatal("the headline price can now be set from an empty history")
	}
}

// ⚠️ Withdrawn against the **stored** version, before the new lines land: an
// ingredient dropped from the invoice has to lose its price claim too, and
// reading the incoming lines would leave exactly that one behind.
func TestAnIngredientDroppedFromAnInvoiceLosesItsClaim(t *testing.T) {
	src := readSource(t, "purchases.go")
	fn := between(t, src, "func (h *Handler) AdminUpdatePurchase", "\n}\n")

	withdraw := strings.Index(fn, "h.withdrawDeliveryPrices(r, stored)")
	apply := strings.Index(fn, "h.applyDeliveryPrices(r, stored)")
	if withdraw < 0 || apply < 0 || withdraw > apply {
		t.Fatal("the old claims are no longer withdrawn against the stored invoice first")
	}
}
