package billing

// ---- The advertising section: who has it, and how much of it ----
//
// ⚠️ **An add-on and never a rung, unlike the assistant next door.** The
// briefing is included at Pro and Enterprise because it costs us one model call
// a morning; this section runs a restaurant's Meta campaigns, which costs us a
// model call per question *and* an API quota that is measured per ad account.
// Folding that into a plan would price it into every customer who never
// advertises, and the ones who do would be the cheapest to serve.
//
// ⚠️ **Nothing existing is withdrawn by this file.** The section is new — no
// restaurant is using it today — so gating it takes nothing from anybody. That
// is the only condition under which a paid feature may be introduced, and it is
// worth saying out loud because it will not be true the second time.
//
// ⚠️ **The ad money is the restaurant's and never ours.** They pay Meta with
// their own card; this price is for the tool. A screen that blurred the two
// would be the screen an owner reads before deciding we spent their money.

// ModAds is the advertising section: connecting a Meta ad account, the
// assistant that proposes campaigns, and what the campaigns brought back.
//
// ⚠️ **A module id rather than a bare add-on string**, because `resolveModules`
// turns every bought add-on into a module the restaurant's own server checks —
// which is what gates `/admin/ads` there and in the panel. One id, three
// gates, no second list to disagree with the first.
const ModAds = "ads"

// AdsMonthly is what the section costs per month, in whole so'm.
//
// ⚠️ **Per restaurant, not per branch.** A chain advertises as one brand from
// one ad account; charging four branches for one campaign would be a bill
// nobody could defend on the phone — the same rule the assistant follows.
//
// Roughly $100 at the rate this was priced at. ⚠️ Written in so'm rather than
// converted at runtime: an invoice that moves with the exchange rate is an
// invoice the customer has to re-read every month.
const AdsMonthly = 1_250_000

// AdsDailyCap is how many assistant requests the advertising section may make
// in a day.
//
// ⚠️ **Its own counter, deliberately not shared with the briefing.** One pot
// would mean a restaurant that asked the advisor four questions before lunch
// finds tomorrow's morning briefing missing — two features failing as one, and
// neither screen able to say why.
//
// ⚠️ **A runaway guard rather than a budget anybody is expected to spend.**
// Writing one campaign is a handful of requests; thirty a day binds only when
// something is looping.
const AdsDailyCap = 30

// AdsEntitled is whether this restaurant may open the advertising section.
//
// ⚠️ **No plan includes it**, which is why there is no `AdsIncluded` beside
// this. The assistant has one because it ships inside two rungs; inventing the
// same shape here would suggest a rung grants this one day and quietly answer
// "no" for every plan there is.
func AdsEntitled(addons []string) bool {
	for _, a := range addons {
		if a == ModAds {
			return true
		}
	}
	return false
}
