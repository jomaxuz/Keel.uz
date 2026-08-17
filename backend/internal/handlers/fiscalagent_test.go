package handlers

import (
	"testing"
	"time"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ⚠️ The blank-token hole, sealed in both directions.
//
// Every branch that has never set up a relay stores "" — so a comparison that
// treated blank as a value would match all of them at once, and hand the live
// sales of every such restaurant to the first scanner sending an empty header.
// The endpoint is open to the internet, which is what makes this the most
// expensive single line in the file.
func TestBlankAgentTokenNeverMatches(t *testing.T) {
	branchA, branchB := primitive.NewObjectID(), primitive.NewObjectID()
	rows := []models.FiscalSettings{
		{BranchID: branchA, AgentToken: ""},
		{BranchID: branchB, AgentToken: "realtoken"},
	}

	if _, ok := matchAgentToken(rows, ""); ok {
		t.Fatal("an empty presented token authenticated")
	}
	if got, ok := matchAgentToken(rows, "realtoken"); !ok || got.BranchID != branchB {
		t.Fatal("a real token did not authenticate its own branch")
	}
	if _, ok := matchAgentToken(rows, "realtoke"); ok {
		t.Fatal("a prefix of a token authenticated")
	}
	if _, ok := matchAgentToken(rows, "RealToken"); ok {
		t.Fatal("the comparison is case-insensitive")
	}
}

// ⚠️ Whether the relay is used is a **timestamp**, never a stored flag.
//
// The relay's whole failure mode is going quiet — the register's PC is rebooted
// for Windows updates and nothing on any screen changes. A saved "connected"
// would still be true a week later and would keep every receipt queued for a
// machine that is off; an old timestamp is simply old, and the till goes back
// to making the call itself.
func TestRelayIsPreferredOnlyWhileItIsAlive(t *testing.T) {
	if agentUsable(nil) {
		t.Fatal("a missing branch counted as having a live relay")
	}
	if agentUsable(&models.FiscalSettings{}) {
		t.Fatal("a branch that has never run a relay counted as having one")
	}

	fresh := time.Now().Add(-5 * time.Second)
	if !agentUsable(&models.FiscalSettings{AgentSeenAt: &fresh}) {
		t.Fatal("a relay that asked for work five seconds ago was ignored")
	}

	// Long enough that a machine switched off at closing time is not still
	// "connected" when the restaurant opens.
	stale := time.Now().Add(-agentAlive - time.Second)
	if agentUsable(&models.FiscalSettings{AgentSeenAt: &stale}) {
		t.Fatal("a silent relay was still trusted with the next receipt")
	}
}

// The poll must return before whatever sits in front of us gives up on it, or
// every quiet minute writes an error into the agent's log and the real failures
// stop being findable.
func TestAgentPollReturnsBeforeTheRouterTimeout(t *testing.T) {
	if agentPollWait >= 30*time.Second {
		t.Fatalf("poll wait %v is not shorter than the router's 30s timeout", agentPollWait)
	}
	// And the liveness window has to outlast a poll cycle comfortably, or the
	// route flips between relay and browser mid-shift on one slow round trip.
	if agentAlive <= 2*agentPollWait {
		t.Fatalf("liveness window %v is too tight for a %v poll", agentAlive, agentPollWait)
	}
}

// A rotated token is a revocation, so the token itself must be unguessable —
// this string is the entire authentication for reading a branch's sales as they
// happen. Same rule as the order number.
func TestAgentTokensAreLongAndDistinct(t *testing.T) {
	seen := map[string]bool{}
	for range 100 {
		tok := newAgentToken()
		if len(tok) < 48 {
			t.Fatalf("token %q is too short to be unguessable", tok)
		}
		if seen[tok] {
			t.Fatal("newAgentToken repeated itself")
		}
		seen[tok] = true
	}
}

// ⚠️ The relay defaults to its own machine when no address is stored. It runs
// on the register's PC — that is the entire reason it exists — and Multikassa's
// own documentation gives localhost as the base address. Refusing to file over
// a blank field would refuse the setup the vendor recommends.
func TestRelayDefaultsToItsOwnMachine(t *testing.T) {
	if got := agentBase(&models.FiscalSettings{}); got != "http://localhost:8080" {
		t.Fatalf("agentBase() = %q, want the local register", got)
	}
	set := &models.FiscalSettings{Provider: "multikassa"}
	set.Multikassa.BaseURL = "http://192.168.1.50:9090/"
	if got := agentBase(set); got != "http://192.168.1.50:9090" {
		t.Fatalf("agentBase() = %q, want the stored address with no trailing slash", got)
	}
}
