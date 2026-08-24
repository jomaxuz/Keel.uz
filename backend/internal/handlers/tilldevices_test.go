package handlers

import (
	"testing"

	"restaurant-backend/internal/models"
)

// ⚠️ **The two answers that must never be got backwards.**
//
// An install with no subscription document is every restaurant that predates
// the till being sold, plus every one we host without a counter. A plan with
// `registers: 0` is Enterprise, which is uncapped on purpose. Reading either as
// "cap of zero" would refuse the *first* screen every one of them tries to
// pair — a total outage of a feature they already had, produced by a limit
// nobody sold them.
func TestNoSubscriptionMeansNoRegisterCap(t *testing.T) {
	cases := []struct {
		name  string
		sub   *models.Subscription
		count int
	}{
		{"no document at all", nil, 99},
		{"till not sold", &models.Subscription{Enabled: false, Registers: 1}, 99},
		{"uncapped plan", &models.Subscription{Enabled: true, Registers: 0}, 99},
	}
	for _, c := range cases {
		if tillCapExceeded(c.sub, c.count) {
			t.Errorf("%s: refused a register — there is no cap here", c.name)
		}
	}
}

// The cap itself: N registers means the Nth pairs and the N+1th does not.
//
// ⚠️ `>=` rather than `>`, and the off-by-one is the whole feature: `count` is
// what is already bound, so a Standard branch holding two must refuse the
// third. Written the other way the plan silently sells one extra register to
// everybody.
func TestRegisterCapAllowsExactlyThePlan(t *testing.T) {
	sub := &models.Subscription{Enabled: true, Registers: 2}
	for count, refuse := range map[int]bool{0: false, 1: false, 2: true, 3: true} {
		if got := tillCapExceeded(sub, count); got != refuse {
			t.Errorf("with %d bound: refused=%v, want %v", count, got, refuse)
		}
	}
}

// The refusal has to name a way out. "Limit reached" on its own leaves the
// manager holding a new monoblock with exactly one action — ringing us to ask
// the question the message could have answered.
func TestRefusalNamesTheRungThatLiftsIt(t *testing.T) {
	sub := &models.Subscription{
		Enabled:   true,
		Registers: 2,
		Plans: []models.SubscriptionPlan{
			{ID: "start", Registers: 1},
			{ID: "standard", Registers: 2},
			{ID: "pro", Registers: 5},
			{ID: "enterprise", Registers: 0},
		},
	}
	if got := nextPlanAbove(sub, 2); got != "pro" {
		t.Errorf("nextPlanAbove(limit 2) = %q, want \"pro\"", got)
	}
	// ⚠️ An uncapped rung counts as "more", not as zero. Read numerically,
	// Enterprise's 0 is smaller than every cap and the customer at the top of
	// the ladder would be offered nothing.
	if got := nextPlanAbove(sub, 5); got != "enterprise" {
		t.Errorf("nextPlanAbove(limit 5) = %q, want \"enterprise\"", got)
	}
	// Already uncapped: no button rather than a button offering nothing.
	uncapped := &models.Subscription{Enabled: true, Plans: []models.SubscriptionPlan{
		{ID: "start", Registers: 1},
	}}
	if got := nextPlanAbove(uncapped, 1); got != "" {
		t.Errorf("nextPlanAbove with nothing above = %q, want empty", got)
	}
}

// The exit button's audience, stated as the permission rather than as a job
// title. Cashiers and waiters must not see it: retiring the screen mid-shift
// takes the machine out of service, and getting it back needs somebody with a
// panel login to fetch a fresh link.
//
// ⚠️ Pinned against the seeded roles, because that is where the line actually
// lands for a real restaurant — and "Barmen" is the case worth having a test
// for: they hold cashier *and* kitchen, and still must not be able to do this.
func TestOnlyManagersMayRetireAScreen(t *testing.T) {
	byName := map[string][]string{}
	for _, r := range models.SeedRoles() {
		byName[r.Name] = r.Perms
	}
	mayNot := []string{"Kassir", "Ofitsiant", "Barmen", "Xostes", "Oshpaz"}
	for _, name := range mayNot {
		perms, ok := byName[name]
		if !ok {
			t.Fatalf("seeded role %q is gone — this test names the line", name)
		}
		s := models.Staff{IsActive: true, Perms: perms}
		if s.Can(models.PermVoid) {
			t.Errorf("%q can retire a till screen — it must not", name)
		}
	}
	mayDo := []string{"Ish boshqaruvchi", "Menejer", "Zal administratori"}
	for _, name := range mayDo {
		s := models.Staff{IsActive: true, Perms: byName[name]}
		if !s.Can(models.PermVoid) {
			t.Errorf("%q cannot retire a till screen — then nobody in the room can", name)
		}
	}
}
