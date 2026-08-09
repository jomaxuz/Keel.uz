package handlers

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"keel-control/internal/models"
)

// ⚠️ The agent boundary, which is the whole reason these roles exist.
//
// Sealed as a table because the failure is silent and expensive: an agent who can see
// every customer sees every restaurant's turnover, and nothing on screen says so. The
// scope filter is the single place that decides it, so this test is the single place
// that has to be right.
func TestTenantScopeNarrowsAgentsOnly(t *testing.T) {
	id := primitive.NewObjectID()
	cases := []struct {
		role     string
		narrowed bool
	}{
		{models.RoleOwner, false},
		{models.RoleAdmin, false},
		{models.RoleManager, false},
		{models.RoleAgent, true},
		// ⚠️ The account seeded at first boot has no role, and it is the platform
		// owner. Reading it as an agent would lock them out of their own console on
		// the deploy that introduced roles.
		{"", false},
		// A typo in a role name must not widen anybody's reach.
		{"sales", true},
	}
	for _, c := range cases {
		u := &models.User{ID: id, Role: c.role}
		scope := tenantScope(u)
		if got := len(scope) > 0; got != c.narrowed {
			t.Fatalf("role %q: narrowed=%v, want %v (scope %v)", c.role, got, c.narrowed, scope)
		}
		if c.narrowed && scope["createdById"] != id {
			t.Fatalf("role %q: scope does not filter on the creator: %v", c.role, scope)
		}
	}
}

// An agent never reads a customer's business figures, and a typo'd role is an agent.
func TestStatsAndStaffPermissions(t *testing.T) {
	for role, stats := range map[string]bool{
		models.RoleOwner: true, models.RoleAdmin: true,
		models.RoleManager: false, models.RoleAgent: false, "sales": false,
	} {
		if got := models.CanSeeStats(role); got != stats {
			t.Fatalf("CanSeeStats(%q) = %v, want %v", role, got, stats)
		}
	}
	// Only the owner may create accounts or read the log.
	for _, role := range []string{models.RoleAdmin, models.RoleManager, models.RoleAgent} {
		if models.CanManageStaff(role) || models.CanSeeLog(role) {
			t.Fatalf("%q may manage staff or read the log", role)
		}
	}
	if !models.CanManageStaff(models.RoleOwner) || !models.CanSeeLog(models.RoleOwner) {
		t.Fatal("owner cannot manage staff or read the log")
	}
}

// A negative visit needs its reason, and a callback needs its date.
func TestCloseVisitRequiresReasonAndDate(t *testing.T) {
	var v models.Visit
	if err := closeVisit(&v, visitRequest{Outcome: models.OutcomeNegative}); err == nil {
		t.Fatal("a negative outcome was accepted with no comment")
	}
	if err := closeVisit(&v, visitRequest{Outcome: models.OutcomeCallback, Comment: "keyin"}); err == nil {
		t.Fatal("a callback was accepted with no date")
	}
	if err := closeVisit(&v, visitRequest{Outcome: "maybe"}); err == nil {
		t.Fatal("an unknown outcome was accepted")
	}
	if err := closeVisit(&v, visitRequest{
		Outcome: models.OutcomeNegative, Comment: "menyusi yo'q",
	}); err != nil {
		t.Fatalf("a negative outcome with a reason was refused: %v", err)
	}
	if v.Status != models.VisitDone || v.VisitedAt == nil {
		t.Fatalf("visit not closed: %+v", v)
	}
}

// ⚠️ Every route that changes something, and who may reach it.
//
// This is the test that would have caught the real gap: I gated the list, the card and
// the live figures, and left every endpoint that *changes* something open — so any
// account could suspend a customer, delete one, issue an invoice or restart the fleet.
// The dangerous routes are the ones nobody opens while testing a sales account.
//
// It asserts the permission map rather than the routes themselves, because that map is
// what the router's wrappers name.
func TestPermissionsRefuseAgentsEverywhereItMatters(t *testing.T) {
	type want struct{ provision, billing, stats bool }
	cases := map[string]want{
		models.RoleOwner:   {true, true, true},
		models.RoleAdmin:   {true, true, true},
		models.RoleManager: {false, false, false},
		models.RoleAgent:   {false, false, false},
		"":                 {true, true, true}, // the seeded owner
		"typo":             {false, false, false},
	}
	for role, w := range cases {
		if got := models.CanProvision(role); got != w.provision {
			t.Fatalf("CanProvision(%q) = %v, want %v", role, got, w.provision)
		}
		if got := models.CanBill(role); got != w.billing {
			t.Fatalf("CanBill(%q) = %v, want %v", role, got, w.billing)
		}
		if got := models.CanSeeStats(role); got != w.stats {
			t.Fatalf("CanSeeStats(%q) = %v, want %v", role, got, w.stats)
		}
	}
}
