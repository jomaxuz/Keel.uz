package handlers

import (
	"os"
	"strings"
	"testing"
)

// The advertising section's rules about money and credentials, tested by
// reading the source. Every one of them fails quietly: a budget in the wrong
// units creates a campaign that runs, a stamped-but-unsent order is simply
// never counted, and a token in a response looks like nothing at all.

func adsSource(t *testing.T, name string) string {
	t.Helper()
	src, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(src)
}

// ⚠️ **The token can spend money and must never reach a browser.** The same
// rule as the payment keys, and the same failure mode: one forgotten field in
// one response, discovered by nobody.
func TestTheMetaTokenNeverReachesTheBrowser(t *testing.T) {
	state := adsSource(t, "ads.go")
	fn := between(t, state, "func (h *Handler) AdminAdsState", "\n}\n")
	// ⚠️ Only the part that becomes JSON is examined: the handler reads
	// `s.Token` once to decide whether there is one, which is the whole point.
	body := fn[strings.Index(fn, "httpx.JSON("):]
	if strings.Contains(body, "s.Token") {
		t.Fatal("the stored Meta token is being written into the state response")
	}
	if !strings.Contains(fn, `"hasToken"`) {
		t.Fatal("the state no longer answers with a flag in place of the token")
	}
	model := adsSource(t, "../models/ads.go")
	if !strings.Contains(model, `Token string `+"`"+`bson:"token,omitempty" json:"-"`) {
		t.Fatal("AdsSettings.Token lost its json:\"-\" — it will now be " +
			"marshalled by anything that returns the settings document")
	}
}

// ⚠️ **The currency is read from Meta before a budget is converted, never
// assumed.** Meta counts in the ad account currency's minor units and an Uzbek
// restaurant's account is almost never in so'm: an assumed offset creates a
// campaign that spends a hundred times what the owner typed, immediately, on a
// real card.
func TestTheBudgetIsConvertedAgainstTheAccountMetaReported(t *testing.T) {
	fn := between(t, adsSource(t, "adscampaign.go"),
		"func (h *Handler) AdminAdsCreateCampaign", "\n}\n")
	account := strings.Index(fn, "client.Account(ctx")
	convert := strings.Index(fn, "meta.MinorUnits(acc.Currency)")
	if account < 0 || convert < 0 {
		t.Fatal("the campaign no longer reads the ad account's own currency " +
			"before working out a budget")
	}
	if account > convert {
		t.Fatal("the budget is converted before the account is read")
	}
	if !strings.Contains(fn, "acc.MinDailyBudget > 0 && daily < acc.MinDailyBudget") {
		t.Fatal("Meta's own floor is no longer checked — the one guard that " +
			"catches a currency-unit mistake before the money moves")
	}
}

// ⚠️ **The owner's ceiling binds the owner too.** It is the answer to "how much
// am I willing to lose in a day"; a screen that let the next field exceed it
// would make it decoration — including for the rules, which read the same
// number.
func TestTheDailyCapBindsEveryPathThatMovesABudget(t *testing.T) {
	src := adsSource(t, "adscampaign.go")
	create := between(t, src, "func (h *Handler) AdminAdsCreateCampaign", "\n}\n")
	if !strings.Contains(create, "daily > capMinor") {
		t.Fatal("a campaign can be created above its own cap")
	}
	update := between(t, src, "func (h *Handler) AdminAdsUpdateCampaign", "\n}\n")
	if !strings.Contains(update, "row.CapMinor > 0 && next > row.CapMinor") {
		t.Fatal("the budget can be raised past the cap by editing it afterwards")
	}
	rules := between(t, adsSource(t, "adsrules.go"),
		"func (h *Handler) applyAdsRule(", "\n}\n")
	if !strings.Contains(rules, "rules.MaxDailyMinor < ceiling") {
		t.Fatal("the rules no longer take the lower of the two ceilings — the " +
			"stricter setting has become decoration")
	}
}

// ⚠️ **Switching a campaign on has to reach all three objects.** Meta pauses
// independently at every level, and an ad set left paused under an active
// campaign reports itself as running and spends nothing — which reads as our
// bug and is invisible from our own screen.
func TestSwitchingACampaignReachesEveryLevel(t *testing.T) {
	fn := between(t, adsSource(t, "adscampaign.go"),
		"func (h *Handler) adsSwitch", "\n}\n")
	for _, id := range []string{"row.MetaCampaignID", "row.AdSetID", "row.AdID"} {
		if !strings.Contains(fn, id) {
			t.Fatalf("%s is no longer switched with the rest of the chain", id)
		}
	}
}

// ⚠️ **A half-built chain is unwound.** Meta has no transaction across the four
// objects, and the alternative is a restaurant's Ads Manager filling with empty
// campaigns from failed attempts — objects they did not make and will not
// delete.
func TestAFailedChainIsUnwound(t *testing.T) {
	fn := between(t, adsSource(t, "adscampaign.go"),
		"func (h *Handler) AdminAdsCreateCampaign", "\n}\n")
	if !strings.Contains(fn, "client.Delete(context.WithoutCancel(ctx), made[i])") {
		t.Fatal("a failure part-way through no longer removes what was created")
	}
	if !strings.Contains(fn, `"PAUSED"`) && !strings.Contains(
		adsSource(t, "../meta/campaign.go"), `"status":                {"PAUSED"}`) {
		t.Fatal("a campaign is no longer created paused — there is now no " +
			"moment in which a mistake is free")
	}
}

// ⚠️ **Nothing is stamped as reported until Meta has taken it.** Stamping first
// loses exactly the orders the report exists to count, and loses them
// silently; stamping after a failure is the same bug with an extra step.
func TestOrdersAreStampedOnlyAfterMetaTookThem(t *testing.T) {
	fn := between(t, adsSource(t, "adscapi.go"),
		"func (h *Handler) sendAdsEventsOnce", "\n}\n")
	send := strings.Index(fn, "client.SendEvents")
	stamp := strings.Index(fn, `"adsEventAt": now`)
	if send < 0 || stamp < 0 {
		t.Fatal("the reporting sweep no longer sends, or no longer stamps")
	}
	if stamp < send {
		t.Fatal("orders are stamped as reported before Meta has taken them")
	}
	if !strings.Contains(fn, "return") ||
		!strings.Contains(fn, "log.Printf(\"ads events: %v\", err)") {
		t.Fatal("a failed batch no longer leaves the orders unstamped for the " +
			"next tick")
	}
}

// ⚠️ **The order number is the deduplication key.** The same order reaches Meta
// from the browser pixel and from this server, and Meta collapses the pair only
// when both carry the same id. Without it every online order is counted twice —
// and a doubled number is worse than a missing one, because an owner believes
// it.
func TestEveryReportedOrderCarriesItsNumberAsTheEventID(t *testing.T) {
	fn := between(t, adsSource(t, "adscapi.go"), "func (h *Handler) adsEventOf", "\n}\n")
	if !strings.Contains(fn, "EventID: o.Number") {
		t.Fatal("the event id is no longer the order number; the pixel's copy " +
			"of the same order will be counted a second time")
	}
	if !strings.Contains(fn, `Currency:    "UZS"`) {
		t.Fatal("the order's value stopped being reported in the currency it " +
			"was actually charged in")
	}
}

// ⚠️ **Only a stop is worth a message.** A budget nudged inside the owner's own
// ceiling is the rules doing what was asked; a notification for it every six
// hours mutes the channel that carries the ones that matter.
func TestTheOwnerIsToldWhenARuleStopsACampaign(t *testing.T) {
	fn := between(t, adsSource(t, "adsrules.go"), "func (h *Handler) adsRuleAct", "\n}\n")
	if !strings.Contains(fn, "models.AlertAdsPaused") {
		t.Fatal("an automatic pause no longer reaches the owner — it will be " +
			"found days later as 'the orders fell off'")
	}
	if !strings.Contains(fn, `if action == "pause" || action == "stop"`) {
		t.Fatal("every budget nudge now sends a message; the channel carrying " +
			"the stops will be muted")
	}
}

// ⚠️ **The rules do nothing at all until the owner has switched them on.** A
// rule with no ceiling behind it is a machine acting on its own judgement with
// somebody else's card.
func TestTheRulesDoNothingUntilTheyAreTurnedOn(t *testing.T) {
	fn := between(t, adsSource(t, "adsrules.go"), "func (h *Handler) applyAdsRules", "\n}\n")
	if !strings.Contains(fn, "if !s.Rules.On") {
		t.Fatal("the rules run without having been switched on")
	}
}

// ⚠️ **The app secret stays on the platform.** A value copied into every tenant
// container is a value that has to be rotated in every tenant container, and a
// container is the customer's own machine.
func TestTheTenantNeverHoldsTheMetaAppSecret(t *testing.T) {
	for _, name := range []string{"adsconnect.go", "adscampaign.go", "ads.go"} {
		src := adsSource(t, name)
		for _, bad := range []string{"client_secret", "MetaAppSecret", "APP_SECRET"} {
			if strings.Contains(src, bad) {
				t.Fatalf("%s reaches for %q: the platform's app secret is "+
					"being handled on a customer's server", name, bad)
			}
		}
	}
	connect := between(t, adsSource(t, "adsconnect.go"),
		"func (h *Handler) AdminAdsConnect", "\n}\n")
	if !strings.Contains(connect, `"/internal/ads-token"`) {
		t.Fatal("the code is no longer exchanged through the console")
	}
}
