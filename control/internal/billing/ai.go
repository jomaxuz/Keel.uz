package billing

// ---- The assistant: who has it, and how much of it ----
//
// ⚠️ **An add-on, not a rung — for the reason written at the top of plans.go.**
// Making the assistant a plan feature would mean a restaurant that buys a Start
// till discovers its briefing has switched off, because a customer with no
// subscription at all has no plan to fail the check against. Paying us more
// would take a feature away, which is the exact defect that kept analysis and
// the CRM off the ladder. Sold on top instead, and *included* at the two rungs
// that already price it in.
//
// ⚠️ **Nothing existing is withdrawn by this file.** The assistant is new — no
// restaurant is using it today — so gating it takes nothing from anybody. That
// is the only condition under which a paid feature may be introduced at all,
// and it is worth saying out loud because it will not be true the second time.

// AddonAI is the assistant on a plan that does not include it.
const AddonAI = "ai"

// AIMonthly is what it costs per month, in whole so'm.
//
// ⚠️ Per restaurant, not per branch — unlike the till rungs. The briefing is
// about a business, and a chain reading one briefing per branch would be
// charged four times for what is arguably one answer.
const AIMonthly = 250_000

// AIIncluded is whether this plan already pays for the assistant.
func AIIncluded(plan string) bool {
	switch plan {
	case PlanPro, PlanEnterprise:
		return true
	}
	return false
}

// AIEntitled is whether this restaurant may use the assistant at all.
func AIEntitled(plan string, addons []string) bool {
	if AIIncluded(plan) {
		return true
	}
	for _, a := range addons {
		if a == AddonAI {
			return true
		}
	}
	return false
}

// AIDailyCap is how many assistant requests one restaurant may make in a day.
//
// ⚠️ **A runaway guard, not a budget anybody is expected to spend.** Ordinary
// use is the morning briefing plus a campaign text or two — two or three
// requests. These numbers only ever bind when something is wrong: a dashboard
// left open on a refresh loop, a script, a bug of ours. Sized so the restaurant
// that is genuinely using the feature never meets them, and the one that is
// looping meets them within the hour rather than at the end of the month, on an
// invoice, in our accounts.
//
// ⚠️ **They step with the rung because the rungs mean different restaurants.**
// A single-till Start is one manager on one phone; an Enterprise chain is four
// managers, four branches and a head office, and one cap across all of them
// would be four times tighter for the customer paying most.
func AIDailyCap(plan string) int {
	switch plan {
	case PlanEnterprise:
		return 50
	case PlanPro:
		return 20
	case PlanStandard:
		return 10
	default:
		// Start, and a restaurant with no till subscription at all — the
		// website-only customer who bought the assistant on its own. Same rung
		// because it is the same size of business.
		return 5
	}
}

// AIExtraBlock is what one bought increment of daily allowance is.
//
// ⚠️ **Ten a day, priced monthly.** The limit is a *daily* number because that
// is what protects us from a runaway — a monthly pool would be spent in an
// afternoon by a stuck browser tab and then the restaurant has nothing for
// three weeks. But it is *sold* monthly because that is how a restaurant thinks
// about a bill.
const (
	AIExtraBlock   = 10
	AIExtraMonthly = 100_000
)

// AIDailyCapWith is the cap once bought blocks are counted.
//
// ⚠️ **Added to the plan's cap, never replacing it.** A Pro restaurant that buys
// one block gets thirty a day, not ten — otherwise buying more would be a
// downgrade for anybody above the smallest rung, which is the sort of thing
// nobody notices until a customer does.
func AIDailyCapWith(plan string, extraBlocks int) int {
	if extraBlocks < 0 {
		extraBlocks = 0
	}
	return AIDailyCap(plan) + extraBlocks*AIExtraBlock
}

// AIExtraMonthlyFor is what those blocks cost per month.
func AIExtraMonthlyFor(extraBlocks int) int {
	if extraBlocks <= 0 {
		return 0
	}
	return extraBlocks * AIExtraMonthly
}
