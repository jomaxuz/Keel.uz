package handlers

import (
	"strings"
	"testing"

	"restaurant-backend/internal/models"
)

// ⚠️ **A debt is delivered and not paid, and the `delivered` half of the
// takings test would otherwise count it.** The guest walked out with the food —
// which is exactly why the sale is `delivered`, and exactly why it is not
// money. Booking it as revenue on the day of the meal is the mistake this
// system was already fixed of once, at the dashboard.
func TestADebtIsNotTakingsUntilItIsPaid(t *testing.T) {
	owed := models.Order{
		Status:        models.StatusDelivered,
		PaymentMethod: models.MethodDebt,
		PaymentStatus: models.PayUnpaid,
	}
	if received(owed) {
		t.Fatal("an unpaid debt is being counted as revenue")
	}
	// Settled: the money arrived, and the method is the one it arrived in.
	settled := models.Order{
		Status:        models.StatusDelivered,
		PaymentMethod: models.ProviderCash,
		PaymentStatus: models.PayPaid,
	}
	if !received(settled) {
		t.Fatal("a settled debt stopped counting")
	}
}

// ⚠️ The repayment is guarded by the debt filter rather than by the id alone:
// two people pressing "paid" on two screens must take the money once, and a
// debt that was cancelled or already settled must refuse rather than quietly
// move a second payment into today's drawer.
func TestPayingADebtIsGuardedAndDatedToday(t *testing.T) {
	src := readSource(t, "debts.go")
	fn := between(t, src, "func (h *Handler) AdminPayDebt", "\n}\n")

	if !strings.Contains(fn, "filter := debtFilter(scope)") {
		t.Fatal("the repayment no longer checks that this is an open debt")
	}
	if !strings.Contains(fn, `"paidAt":        now`) {
		t.Fatal("the repayment is not dated when the money arrived")
	}
	// ⚠️ Never "debt" again: that would leave the sale unpaid and produce a
	// repayment that changed nothing.
	if !strings.Contains(fn, "method == models.MethodDebt") {
		t.Fatal("a debt can be repaid with a debt")
	}

	// A cancelled check is not owed: the food never left, and chasing somebody
	// for it is how a restaurant loses a regular over its own bookkeeping.
	filter := between(t, src, "func debtFilter", "\n}\n")
	if !strings.Contains(filter, "models.StatusCancelled") {
		t.Fatal("cancelled checks are being chased as debts")
	}
}

// ⚠️ **A debt must not fall through the shift report's payment switch.**
// The three cases are cash, card and *everything else* — so before this was
// guarded, a check taken on the slate was printed as a bank transfer and added
// to the shift's sales. Nothing looked wrong: the total stayed plausible, the
// paper stayed the same shape, and the only symptom was a drawer that could
// never be reconciled against it. Somebody would eventually be accused of the
// difference.
func TestTheShiftReportDoesNotSellWhatWasTakenOnTheSlate(t *testing.T) {
	src := readSource(t, "tillshiftreport.go")
	fn := between(t, src, "func (h *Handler) salesInShift", "\n}\n")

	i := strings.Index(fn, "out.Sales += o.Total")
	if i < 0 {
		t.Fatal("the sales total is no longer summed here")
	}
	// The debt has to be taken out of the running *before* the sale is added,
	// not filtered afterwards: a later fix-up would leave the switch below
	// still choosing a payment method for money nobody paid.
	before := fn[:i]
	if !strings.Contains(before, "models.MethodDebt") ||
		!strings.Contains(before, "out.Debt += o.Total") {
		t.Fatal("a debt is still counted as a sale (and as a transfer)")
	}
	if !strings.Contains(before, "continue") {
		t.Fatal("a debt falls through into the payment-method switch")
	}
}
