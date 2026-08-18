package handlers

import (
	"strings"
	"testing"
)

// ⚠️ **After the discount, and rounded once.** A guest given 20% off and then
// charged service on the full bill is being charged for a discount they were
// told they had — on the receipt they read most carefully, because somebody
// just did them a favour.
func TestServiceIsChargedOnWhatTheGuestPays(t *testing.T) {
	// 100 000 with 20 000 off, 10% service: 8 000, not 10 000.
	if got := serviceOn(80000, 10); got != 8000 {
		t.Fatalf("service=%d, want 8000", got)
	}
	// Half-up, once, so the panel, the paper and the drawer cannot end up a
	// som apart — which is what somebody spends an evening looking for.
	if got := serviceOn(33333, 10); got != 3333 {
		t.Fatalf("service=%d, want 3333", got)
	}
	if got := serviceOn(1005, 10); got != 101 {
		t.Fatalf("service=%d, want 101 (half up)", got)
	}
	// Off is off, and a fully discounted bill adds nothing to nothing.
	if serviceOn(50000, 0) != 0 || serviceOn(0, 10) != 0 || serviceOn(-5, 10) != 0 {
		t.Fatal("service charged where there is none to charge")
	}
}

// ⚠️ The rate lives on the **order**, copied when the table sat down, and every
// screen computes from there. Twenty handlers draw a check; a rate passed in
// beside the order would be forgotten at one of them, and a screen showing a
// different total from the one the till charges is the worst version of this.
func TestTheRateIsCopiedOntoTheCheckNotReadAtPayment(t *testing.T) {
	src := readSource(t, "till.go")
	fn := between(t, src, "func (h *Handler) StaffOpenCheck", "\n}\n")

	if !strings.Contains(fn, "branch.Service.Enabled") {
		t.Fatal("the rate is no longer copied when the check is opened")
	}
	// ⚠️ Tables only. A service charge on a takeaway coffee is the version of
	// this feature guests complain about, and the branch setting cannot tell
	// the difference — this line can.
	if !strings.Contains(fn, `tableID != ""`) {
		t.Fatal("a counter sale is being charged for service")
	}

	view := between(t, src, "func viewCheck", "\n}\n")
	if !strings.Contains(view, "o.ServicePercent") {
		t.Fatal("the check on screen no longer shows what it will charge")
	}
}

// ⚠️ A sale taken offline carries no service charge, and that is the honest
// record rather than an oversight: the device charged what it charged and the
// guest has gone. Adding the room's percentage afterwards would book money that
// was never in the drawer, and the shortfall would surface at the count as the
// cashier's problem.
func TestAnOfflineSaleIsRecordedAtWhatWasPaid(t *testing.T) {
	src := readSource(t, "tillsync.go")
	if strings.Contains(src, "serviceOn(") {
		t.Fatal("the sync is inventing a service charge the guest never paid")
	}
}
