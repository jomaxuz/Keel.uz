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

// ⚠️ **The print alert is silent, and bounded.** Two decisions, both about the
// same habit: an alarm whose clearing act is at the printer — load paper, plug
// it in — is one nobody in this app can silence, and people who learn to ignore
// a bell carry the habit to the two that must never be ignored. A count that
// cannot be brought back to zero (a printer unplugged for a week) is read the
// same way.
func TestThePrintAlertIsABannerAndCoversTonight(t *testing.T) {
	src := readSource(t, "adminstats.go")
	fn := between(t, src, "func printFailedFilter", "\n}\n")

	if !strings.Contains(fn, "12 * time.Hour") {
		t.Fatal("the print alert counts failures nobody can act on any more")
	}
	// ⚠️ `$gte`, not equality: a job handed out once more by a second agent
	// would slip past `== MaxPrintTries`, and that row is the one somebody has
	// to act on.
	if !strings.Contains(fn, `"$gte": models.MaxPrintTries`) {
		t.Fatal("a job tried more than the limit stops being counted")
	}
	if !strings.Contains(fn, `"doneAt"`) {
		t.Fatal("receipts that did print are being counted as failures")
	}

	// No sound: the bell's two chimes are named in the frontend, and this
	// alert must never join them. Sealed here as well because the decision is
	// the server's to describe — it is the one that knows the clearing act is
	// in another room.
	alerts := between(t, src, "func (h *Handler) AdminAlerts", "\n}\n")
	if !strings.Contains(alerts, `"print": map[string]any{`) {
		t.Fatal("the print alert is no longer reported")
	}
}
