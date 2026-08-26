package handlers

import (
	"strings"
	"testing"
)

// ⚠️ **A queue nobody is reading is a drawer, and the panel called it a queue.**
//
// The till app on the monoblock is what carries paper to a printer. With it
// switched off, a job queued from the panel sits there until somebody turns the
// till on — which may be tomorrow, or never if it was a test. The person who
// pressed the button was told "added to the queue", which is true and is not
// the answer to what they asked, and they went to stand next to a printer that
// was never going to produce anything. That sequence cost a live restaurant an
// evening.
func TestThePanelRefusesRatherThanQueuingIntoNothing(t *testing.T) {
	for _, f := range []string{"adminchecks.go", "receipts.go"} {
		src := readLossSource(t, f)
		if !strings.Contains(src, "h.noAgentHere(r.Context(), ") {
			t.Fatalf("%s queues without checking anybody will collect it", f)
		}
		if !strings.Contains(src, "errTillOff") {
			t.Fatalf("%s refuses without saying why", f)
		}
	}
}

// ⚠️ **`/staff/print` is deliberately not refused, and the difference is that
// it has a fallback.** A branch with no thermal printer relies on the lines
// coming back so the browser can print them; a refusal would take away the only
// way those restaurants print anything. The job is simply not queued, and the
// screen is told which silence it is looking at.
func TestTheTillEndpointStillReturnsItsLines(t *testing.T) {
	src := readLossSource(t, "tillprint.go")
	if strings.Contains(src, "errTillOff") {
		t.Fatal("the endpoint with a browser fallback now refuses outright")
	}
	if !strings.Contains(src, "tillOff := h.noAgentHere(") {
		t.Fatal("it queues into nothing again")
	}
	if !strings.Contains(src, `"tillOff": tillOff`) {
		t.Fatal("the screen cannot tell 'no printer' from 'till switched off'")
	}
}

// ⚠️ **A branch that has never run a till has no settings document, and that is
// not 'probably fine'.** Reading a missing record as an agent being present is
// precisely what let the first version queue into nothing — and it is the exact
// state of every restaurant on its first evening.
func TestAMissingAgentRecordCountsAsNoAgent(t *testing.T) {
	src := readLossSource(t, "printqueue.go")
	i := strings.Index(src, "func (h *Handler) noAgentHere")
	if i < 0 {
		t.Fatal("the guard is gone")
	}
	// ⚠️ Bounded by the end of the file rather than by a fixed offset: the
	// function is last today and a slice past the end panics, which is a test
	// failing for a reason that has nothing to do with what it asserts.
	body := src[i:]
	if j := strings.Index(body, "\n// errTillOff"); j > 0 {
		body = body[:j]
	}
	// The error branch must return true — "no agent" — not fall through.
	if !strings.Contains(body, "err != nil {") ||
		!strings.Contains(body, "return true") {
		t.Fatal("a branch with no fiscal settings is treated as having an agent")
	}
}

// ⚠️ The message names the machine and the fix, because the person reading it
// is somewhere else in the building and the useful next action is "go and
// switch the till on", not "try again".
func TestTheRefusalSaysWhatToDo(t *testing.T) {
	for _, want := range []string{"Kassa", "monoblok"} {
		if !strings.Contains(errTillOff, want) {
			t.Fatalf("%q is missing from %q", want, errTillOff)
		}
	}
}
