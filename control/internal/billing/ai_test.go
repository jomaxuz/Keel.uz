package billing

import "testing"

// ⚠️ **The defect this file's design exists to avoid**, asserted rather than
// described: a restaurant with no till subscription must be able to hold the
// assistant, because otherwise buying a Start till later would take it away.
func TestNoPlanCanStillHoldTheAssistant(t *testing.T) {
	if !AIEntitled("", []string{AddonAI}) {
		t.Fatal("a website-only customer could not buy the assistant")
	}
	if AIEntitled("", nil) {
		t.Fatal("it was given away to somebody who did not buy it")
	}
	// And buying a Start till afterwards must not revoke it.
	if !AIEntitled(PlanStart, []string{AddonAI}) {
		t.Fatal("paying us more took the feature away")
	}
}

func TestTheTwoTopRungsIncludeIt(t *testing.T) {
	for _, p := range []string{PlanPro, PlanEnterprise} {
		if !AIEntitled(p, nil) {
			t.Fatalf("%s had to buy what its price already covers", p)
		}
	}
	for _, p := range []string{PlanStart, PlanStandard} {
		if AIEntitled(p, nil) {
			t.Fatalf("%s got it without buying it", p)
		}
	}
}

// ⚠️ Every rung has a cap, including the ones that pay nothing extra and the
// one that does not exist. A plan id we have never seen — a rung added later,
// a typo in a document — must land on the smallest cap, not on no cap at all.
func TestEveryPlanIsCappedIncludingUnknownOnes(t *testing.T) {
	for _, p := range []string{
		"", PlanStart, PlanStandard, PlanPro, PlanEnterprise, "kelajakdagi-tarif",
	} {
		if n := AIDailyCap(p); n <= 0 {
			t.Fatalf("plan %q has cap %d", p, n)
		}
	}
	if AIDailyCap("kelajakdagi-tarif") != AIDailyCap(PlanStart) {
		t.Fatal("an unknown plan did not fall to the smallest cap")
	}
}

// The ladder must actually climb: a customer paying more meeting a tighter cap
// is the complaint that would arrive first and be hardest to answer.
func TestTheCapsClimbWithTheLadder(t *testing.T) {
	rungs := []string{PlanStart, PlanStandard, PlanPro, PlanEnterprise}
	for i := 1; i < len(rungs); i++ {
		if AIDailyCap(rungs[i]) <= AIDailyCap(rungs[i-1]) {
			t.Fatalf("%s is capped no higher than %s", rungs[i], rungs[i-1])
		}
	}
}

// ⚠️ `cleanAddons` uses AddonPrice as its allowlist, so an add-on with a
// constant but no price is one the console can name and cannot sell — and the
// failure is silent: the console offers it, the save drops it, and the setting
// comes back off with no error anywhere.
func TestTheAssistantIsActuallySellable(t *testing.T) {
	if AddonPrice(AddonAI) != AIMonthly {
		t.Fatalf("AddonPrice says %d, the price is %d", AddonPrice(AddonAI), AIMonthly)
	}
}

// ⚠️ **Bought blocks are added to the plan's cap, never replace it.** A Pro
// restaurant that buys one block must get thirty a day, not ten — otherwise
// buying more is a downgrade for anybody above the smallest rung, which is the
// sort of thing nobody notices until a customer does.
func TestBuyingMoreIsNeverADowngrade(t *testing.T) {
	for _, p := range []string{"", PlanStart, PlanStandard, PlanPro, PlanEnterprise} {
		base := AIDailyCap(p)
		if got := AIDailyCapWith(p, 1); got != base+AIExtraBlock {
			t.Fatalf("%s: one block gave %d, want %d", p, got, base+AIExtraBlock)
		}
		if AIDailyCapWith(p, 0) != base {
			t.Fatalf("%s: buying nothing changed the cap", p)
		}
	}
}

// ⚠️ A negative count is a typo in the console, not a way to take a
// restaurant's allowance away.
func TestANegativeBlockCountTakesNothingAway(t *testing.T) {
	if AIDailyCapWith(PlanPro, -5) != AIDailyCap(PlanPro) {
		t.Fatal("a negative number in the console reduced a paying customer's cap")
	}
	if AIExtraMonthlyFor(-5) != 0 {
		t.Fatal("a negative number produced a negative bill")
	}
}

func TestExtraBlocksArePricedPerBlock(t *testing.T) {
	if AIExtraMonthlyFor(3) != 3*AIExtraMonthly {
		t.Fatalf("three blocks cost %d", AIExtraMonthlyFor(3))
	}
	if AIExtraMonthlyFor(0) != 0 {
		t.Fatal("a restaurant that bought nothing was billed")
	}
}
