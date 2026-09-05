package handlers

import (
	"regexp"
	"strings"
	"testing"
)

// ⚠️ **Whitespace is not part of the claim.** gofmt aligns the keys of a
// `bson.M` to the longest one in the literal, so adding a field elsewhere in
// the query silently changes the spacing of this one — and a guard matched
// against the old spacing stops looking without saying so. The first version of
// this test did exactly that, which is the failure mode the i18n test's own
// comment warns about.
var spaces = regexp.MustCompile(`[ \t]+`)

func flat(s string) string { return spaces.ReplaceAllString(s, " ") }

// ⚠️ **The one thing a repeating notification must never do is repeat.**
//
// Everything the queue watch reports is a *state*, not an event: an order
// nobody accepted at eleven is still unaccepted at midnight, and the query that
// found it will find it again on every tick. The only thing standing between
// that and a phone buzzing every five minutes all night is the stamp — and the
// stamp is written by one line that a later edit can move, reorder or drop
// without anything failing. It would be found in production, at night, by the
// person it was written for.
func TestNothingIsNudgedTwice(t *testing.T) {
	src := readSource(t, "queuewatch.go")

	// Every query the watcher runs asks only for things nobody has been told
	// about. Three queries, three chances to forget.
	if n := strings.Count(flat(src), `"nudgedAt": nil`); n < 3 {
		t.Fatalf("a query does not exclude what has already been sent (found %d of 3)", n)
	}

	for _, fn := range []string{
		"func (h *Handler) nudgeOrders", "func (h *Handler) nudgeBookings",
	} {
		body := between(t, src, fn, "\n}\n")
		body = flat(body)
		set := strings.Index(body, `"nudgedAt": now`)
		send := strings.Index(body, "h.notifyAdmins")
		if set < 0 || send < 0 {
			t.Fatalf("%s no longer both stamps and sends", fn)
		}
		// ⚠️ **Stamped before sending, not after.** The send is a network round
		// trip in its own goroutine; a crash in between would leave the row
		// eligible again on the next tick — which is the repeating message this
		// whole file exists to prevent, arriving through the error path.
		if set > send {
			t.Fatalf("%s sends before it stamps — a crash in between repeats the message", fn)
		}
	}
}

// ⚠️ **A table a waiter opened is not an order waiting to be accepted.** The
// panel's bell learned this the hard way: an open check is stored `status:
// pending`, so every table somebody sat down at rang it, and the list it sent
// the operator to was empty. The watcher reads the same shape of query and
// would reproduce the same failure — except on a phone, at night, once per
// table.
func TestTheQueueWatchIgnoresTillChecks(t *testing.T) {
	body := between(t, readSource(t, "queuewatch.go"), "func (h *Handler) checkQueue", "\n}\n")
	if n := strings.Count(flat(body), `"check": noTill`); n < 2 {
		t.Fatal("an order query no longer excludes till checks — every open table will buzz")
	}
}

// ⚠️ **Managers hear this, and that is the opposite of the loss channel.** The
// loss alerts are about people and go to owners only: telling Dilnoza that
// Dilnoza was noticed is not the point of them. Work standing still is the
// other kind of message — a manager is precisely who acts on it, and sending it
// past them to an owner asleep at midnight helps nobody.
func TestWorkAlertsReachManagersToo(t *testing.T) {
	src := readSource(t, "queuewatch.go")
	if strings.Contains(src, "h.notifyAdmins(o.BranchID, true") ||
		strings.Contains(src, "h.notifyAdmins(b.BranchID, true") {
		t.Fatal("the queue watch went owners-only — a manager can no longer be told to accept an order")
	}
}
