package handlers

import (
	"os"
	"strings"
	"testing"
)

// The campaign plan's rules, tested by reading the source: none of them is
// visible from a screen when it goes wrong, and each one fails quietly.

func adsPlanSource(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile("adsplan.go")
	if err != nil {
		t.Fatal(err)
	}
	return string(src)
}

// ⚠️ **The share is arithmetic done here, not a number asked of the model.**
// The panel draws its bars from it and the owner budgets against it; a share
// written by a model would be a figure nobody computed, sitting in a bar that
// looks measured. Moving this into the prompt would be invisible in every test
// that does not read for it.
func TestTheShareIsComputedOnThisServer(t *testing.T) {
	fn := between(t, adsPlanSource(t), "func (h *Handler) adsFacts", "\n}\n")
	if !strings.Contains(fn, "d.Money * 100 / total") {
		t.Fatal("the weekly share is no longer computed here; a model may be " +
			"supplying the number the bars are drawn from")
	}
}

// ⚠️ **No guest ever travels to the platform for an advertising plan.** The
// advisor pays for its lapsed-guest rows with an alias table; a campaign plan
// needs none of that — it needs what sold and how far the van goes. A payload
// that quietly grew a customer list would put one on the control plane for the
// sake of a sentence about osh.
func TestThePlanCarriesNoGuests(t *testing.T) {
	fn := between(t, adsPlanSource(t), "func (h *Handler) adsFacts", "\n}\n")
	for _, bad := range []string{"Store.Users", "lapsedSample", "aliasOf", "phone"} {
		if strings.Contains(fn, bad) {
			t.Fatalf("the plan payload reaches for %q: customers are leaving "+
				"this server for an advert", bad)
		}
	}
}

// ⚠️ **A delivery radius from a branch that does not deliver is a leftover
// setting**, and an advert aimed at it is money spent on people who cannot
// order. The guard is one `&&` and reads like a redundant one.
func TestTheRadiusOnlyTravelsWhenTheKitchenDelivers(t *testing.T) {
	fn := between(t, adsPlanSource(t), "func (h *Handler) adsFacts", "\n}\n")
	if !strings.Contains(fn, "b.Delivery.Enabled && b.Delivery.MaxKm > 0") {
		t.Fatal("maxKm may now be planned against on a branch that does not deliver")
	}
}

// ⚠️ **One plan a day per lens.** A week's sales do not change between two
// presses of a button, and the add-on is sold with thirty requests a day. The
// cache check sits before the paid call, which is the only position that saves
// anything — after it, the money is already spent.
func TestTodaysPlanIsAnsweredBeforeAnythingIsPaidFor(t *testing.T) {
	fn := between(t, adsPlanSource(t), "func (h *Handler) AdminAdsPlan", "\n}\n")
	cached := strings.Index(fn, "h.Store.AdsPlans.FindOne")
	paid := strings.Index(fn, "callControlPath")
	if cached < 0 || paid < 0 {
		t.Fatal("the plan no longer reads its cache or no longer calls the platform")
	}
	if cached > paid {
		t.Fatal("the cache is read after the model has already been paid for")
	}
}

// ⚠️ **Owner only.** This reads the whole menu's takings on the way to a
// campaign that spends money, and the two limited panel roles must not reach it
// even if somebody widens their allow-list one day.
func TestPlanningACampaignIsTheOwnersAlone(t *testing.T) {
	fn := between(t, adsPlanSource(t), "func (h *Handler) AdminAdsPlan", "\n}\n")
	if !strings.Contains(fn, "h.requireOwner(r)") {
		t.Fatal("a manager can now plan a campaign against the whole menu's takings")
	}
}
