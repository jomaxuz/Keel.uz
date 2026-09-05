package handlers

import (
	"os"
	"strings"
	"testing"
)

// ⚠️ **What is worth sealing is the pair of orderings, not the happy path.** A
// resend that adds the dishes twice is a guest charged for food nobody ordered;
// a guard written in its own update is a crash window that does exactly the
// same thing. Neither shows up in a passing screen.

func TestAppliedOpsAreCappedSoOneTableCannotGrowForever(t *testing.T) {
	ops := []string{}
	for i := 0; i < 60; i++ {
		ops = appendOp(ops, string(rune('a'+i%26))+string(rune('0'+i%10)))
	}
	if len(ops) != 20 {
		t.Fatalf("len = %d, want the recent window only", len(ops))
	}
	// The window has to be the *recent* end: the duplicate that arrives is the
	// one from a moment ago, never the one from the start of service.
	last := appendOp(ops, "zz")
	if last[len(last)-1] != "zz" {
		t.Fatal("the newest operation was not kept")
	}
	if len(last) != 20 {
		t.Fatalf("len = %d after appending to a full window", len(last))
	}
}

func TestARepeatedTapIsAnsweredAsSuccessAndChangesNothing(t *testing.T) {
	src := readSrc(t, "tilllines.go")
	body := between(t, src, "func (h *Handler) StaffAddCheckLines", "\n}\n")

	guard := strings.Index(body, "slices.Contains(o.AppliedOps")
	if guard < 0 {
		t.Fatal("no duplicate guard: a resend adds the dishes a second time")
	}
	// ⚠️ **Before the menu is priced and before the stop list is consulted.** A
	// duplicate that reached those would be refused as "lag'mon tugadi" — a
	// sentence the waiter takes to a table where the dish is already on the
	// check, over a tap that landed the first time.
	pricing := strings.Index(body, "h.menuLines(")
	if pricing < 0 || guard > pricing {
		t.Fatal("the duplicate guard runs after pricing, so a resend can be refused for the wrong reason")
	}
	// ⚠️ **200, not 409.** The phone cannot tell "you never heard me" from "you
	// heard me and I lost the reply"; an error is read as a failure and queued
	// again, forever.
	answer := body[guard:]
	cut := strings.Index(answer, "\n\t}")
	if cut > 0 {
		answer = answer[:cut]
	}
	if !strings.Contains(answer, "httpx.JSON(w, http.StatusOK") {
		t.Fatal("a duplicate is answered as an error, so the phone will resend it forever")
	}
}

func TestTheOpIdIsWrittenWithTheDishesRatherThanBesideThem(t *testing.T) {
	src := readSrc(t, "tilllines.go")
	body := between(t, src, "func (h *Handler) StaffAddCheckLines", "\n}\n")

	// ⚠️ Two updates would be a crash window with a guest's money in it:
	// recorded first, a crash loses the food and keeps the receipt; recorded
	// after, the next resend adds everything twice.
	set := strings.Index(body, `set["appliedOps"]`)
	if set < 0 {
		t.Fatal("the applied operation is not written into the check's own update")
	}
	write := strings.Index(body, "h.Store.Orders.UpdateOne")
	if write < 0 || set > write {
		t.Fatal("the operation id is recorded in a second write, which is a crash window")
	}
}

func readSrc(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
