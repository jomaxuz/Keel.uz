package handlers

import (
	"testing"
	"time"
)

// ⚠️ **A ticket is a credential, however short its life.** It rides in a query
// string — the one place a session token must never go — and the whole reason
// that is acceptable is that it is single use and expires in half a minute.
// These are the two properties that make it so.

func TestATicketOpensOneSocketAndNoMore(t *testing.T) {
	b := ticketBook{live: map[string]time.Time{}}
	id := b.issue()
	if !b.spend(id) {
		t.Fatal("a fresh ticket was refused")
	}
	if b.spend(id) {
		t.Fatal("a spent ticket opened a second socket: a copied URL is a session")
	}
}

func TestAnExpiredTicketIsWorthless(t *testing.T) {
	b := ticketBook{live: map[string]time.Time{}}
	id := b.issue()
	b.mu.Lock()
	b.live[id] = time.Now().Add(-time.Second)
	b.mu.Unlock()
	if b.spend(id) {
		t.Fatal("an expired ticket was accepted")
	}
}

func TestAnUnknownTicketIsRefused(t *testing.T) {
	b := ticketBook{live: map[string]time.Time{}}
	if b.spend("") || b.spend("deadbeef") {
		t.Fatal("a ticket nobody issued was accepted")
	}
}

// ⚠️ The book only grows when somebody opens the widget, so issuing is when it
// is swept — there is no timer goroutine to leak.
func TestIssuingSweepsTheExpiredOnes(t *testing.T) {
	b := ticketBook{live: map[string]time.Time{}}
	stale := b.issue()
	b.mu.Lock()
	b.live[stale] = time.Now().Add(-time.Minute)
	b.mu.Unlock()

	b.issue()
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, still := b.live[stale]; still {
		t.Fatal("an expired ticket was kept")
	}
}

// Two tickets from the same millisecond must not be related.
func TestTicketsAreNotGuessable(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		id := supportTicketID()
		if len(id) != 32 {
			t.Fatalf("ticket %q is not 32 hex characters", id)
		}
		if seen[id] {
			t.Fatal("two tickets collided")
		}
		seen[id] = true
	}
}

// ⚠️ The socket's own wait must outlast the platform's, or the request is
// abandoned here every half minute and the loop restarts having delivered
// nothing.
func TestTheSocketWaitsLongerThanThePlatform(t *testing.T) {
	if supportWaitTimeout <= 25*time.Second {
		t.Fatalf("the socket gives up after %v; the platform holds for 25s",
			supportWaitTimeout)
	}
	// And the ping has to beat an ordinary proxy's idle timeout.
	if supportPing >= time.Minute {
		t.Fatalf("a %v ping will not keep a 60s proxy timeout open", supportPing)
	}
}
