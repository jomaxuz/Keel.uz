package billing

import "testing"

// ⚠️ **`cleanAddons` uses `AddonPrice` as its allowlist**, so an add-on with a
// constant and no price is one the console can name and cannot sell — and the
// failure is silent: the console offers it, the save drops it, and the switch
// comes back off with no error anywhere. The assistant has the same test next
// door, and this is the second time the trap is worth catching rather than
// remembering.
func TestTheAdsSectionIsActuallySellable(t *testing.T) {
	if AddonPrice(ModAds) != AdsMonthly {
		t.Fatalf("AddonPrice says %d, the price is %d", AddonPrice(ModAds), AdsMonthly)
	}
}

// ⚠️ **No plan includes it.** The assistant ships inside Pro and Enterprise;
// this one never does, because it costs us a model call per question and an API
// quota per ad account. A rung that quietly granted it would be a feature we
// give away to the customers most likely to use it hardest.
func TestNoPlanGrantsAdvertising(t *testing.T) {
	for _, p := range Plans() {
		if AdsEntitled(p.Modules) {
			t.Fatalf("plan %q grants the advertising section", p.ID)
		}
	}
}

// Bought, and only then.
func TestAdvertisingIsGrantedByBuyingIt(t *testing.T) {
	if !AdsEntitled([]string{ModStock, ModAds}) {
		t.Fatal("a restaurant that bought it may not open it")
	}
	if AdsEntitled([]string{ModStock, AddonAI}) {
		t.Fatal("the assistant granted the advertising section")
	}
	if AdsEntitled(nil) {
		t.Fatal("an empty list granted the advertising section")
	}
}
