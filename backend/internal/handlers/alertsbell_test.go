package handlers

import (
	"strings"
	"testing"
)

// ⚠️ **The bell must not ring for the dining room.**
//
// Excluding till checks from the orders *list* was only half the fix: the alert
// is a different query and still counted them. An open check is stored
// `status: pending, paymentStatus: "unpaid"`, and `"unpaid" != "pending"`
// matched — so every table a waiter opened rang the panel.
//
// The failure was worse than either half alone: the list was empty and the
// sound played, so the panel called an operator to something they could not
// find and could not silence. That is the shape this test exists to prevent
// coming back — a filter added in one place and not the other.
func TestDeliveryAlertsExcludeTillChecks(t *testing.T) {
	fn := between(t, readSource(t, "adminstats.go"),
		"func (h *Handler) AdminAlerts", "\n}\n")

	// Every alert that means "a delivery order needs accepting".
	for _, v := range []string{
		"pendingOrders", "plain", "placed", "due", "dueWaiting", "upcoming",
	} {
		if !strings.Contains(fn, v+`["check"] = noTill`) {
			t.Fatalf("%s can still count till checks — the bell will ring for a table", v)
		}
	}
}

// ⚠️ And the counterpart: the POS and fiscal alerts are **about** till sales,
// so excluding checks there would switch off the warning that a sale took money
// and was never registered. The rule is not "hide the dining room everywhere".
func TestTillAlertsStillCountTillChecks(t *testing.T) {
	fn := between(t, readSource(t, "adminstats.go"),
		"func (h *Handler) AdminAlerts", "\n}\n")

	for _, v := range []string{"unaccepted", "posFailed", "fiscalUnfiled"} {
		if strings.Contains(fn, v+`["check"] = noTill`) {
			t.Fatalf("%s excludes till checks — it exists to count them", v)
		}
	}
}
