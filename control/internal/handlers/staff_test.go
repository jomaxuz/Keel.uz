package handlers

import (
	"os"
	"regexp"
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

// ⚠️ Who sees which console section. The table *is* the request it came from:
// support sees only Yordam and Xatoliklar; admin, manager and agent see none of
// Yordam, Xatoliklar, Hamkorlar, Qidiruv, Xodimlar; only owner and admin see the
// blog; only the owner lands on the overview.
func TestSectionsPerRole(t *testing.T) {
	type sections struct{ overview, tenants, support, partners, seo, blog, staff bool }
	cases := map[string]sections{
		models.RoleOwner:   {true, true, true, true, true, true, true},
		models.RoleAdmin:   {false, true, false, false, false, true, false},
		models.RoleManager: {false, true, false, false, false, false, false},
		models.RoleAgent:   {false, true, false, false, false, false, false},
		models.RoleSupport: {false, false, true, false, false, false, false},
		"":                 {true, true, true, true, true, true, true}, // the seeded owner
	}
	for role, w := range cases {
		got := sections{
			models.CanSeeOverview(role), models.CanUseTenants(role), models.CanSupport(role),
			models.CanSeePartners(role), models.CanSeo(role), models.CanBlog(role),
			models.CanManageStaff(role),
		}
		if got != w {
			t.Fatalf("role %q: sections %+v, want %+v", role, got, w)
		}
	}
}

// ⚠️ Several roles are a union, and the widest role alone is not it: an agent who
// also answers support must reach the queue, and must still see only their own
// customers.
func TestSeveralRolesAreAUnion(t *testing.T) {
	id := primitive.NewObjectID()
	u := &models.User{ID: id, Role: models.RoleAgent, Roles: []string{models.RoleSupport, models.RoleAgent}}
	if u.RoleOf() != models.RoleAgent {
		t.Fatalf("widest role = %q, want agent", u.RoleOf())
	}
	if !u.Can(models.CanSupport) || !u.Can(models.CanUseTenants) {
		t.Fatal("agent+support lost one half of its roles")
	}
	if u.Can(models.CanSeeAllTenants) || u.Can(models.CanManageStaff) {
		t.Fatal("agent+support gained reach neither role has")
	}
	if scope := tenantScope(u); scope["createdById"] != id {
		t.Fatalf("agent+support is not narrowed to its own customers: %v", scope)
	}
	// A manager among the roles widens the list, because that is what manager is.
	u.Roles = append(u.Roles, models.RoleManager)
	if len(tenantScope(u)) != 0 {
		t.Fatal("agent+manager is still narrowed")
	}
	// Legacy accounts: no list, the single role decides — and the seeded owner is
	// still an owner.
	if !(models.User{}).Has(models.RoleOwner) {
		t.Fatal("the seeded account (no role, no roles) is not an owner")
	}
}

// ⚠️ A request that names no role must not become an owner — the stored empty
// role means owner only because of the seeded account.
func TestRequestWithNoRoleIsNotOwner(t *testing.T) {
	if got := requestRoles(staffRequest{}); got != nil {
		t.Fatalf("no roles asked, got %v", got)
	}
	if got := requestRoles(staffRequest{Roles: []string{"", "  "}}); got != nil {
		t.Fatalf("blank roles asked, got %v", got)
	}
	got := requestRoles(staffRequest{Roles: []string{"support", "sales", "support"}})
	if len(got) != 2 || got[0] != models.RoleAgent || got[1] != models.RoleSupport {
		t.Fatalf("support+typo = %v, want [agent support]", got)
	}
}

// ⚠️ Every gate the router names exists in the permission map. A misspelt gate
// fails closed — which is safe, and also a route nobody can reach, discovered by
// a customer.
func TestEveryRouterGateIsAKnownPermission(t *testing.T) {
	src, err := os.ReadFile("router.go")
	if err != nil {
		t.Fatal(err)
	}
	found := regexp.MustCompile(`h\.need\("([^"]+)"`).FindAllStringSubmatch(string(src), -1)
	if len(found) == 0 {
		t.Fatal("no gates found — has the wrapper been renamed?")
	}
	for _, m := range found {
		if _, ok := models.Permissions[m[1]]; !ok {
			t.Fatalf("router gate %q is not in models.Permissions", m[1])
		}
	}
}
