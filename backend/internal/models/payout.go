package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Money the restaurant has earned but not yet been given ----
//
// ⚠️ **The gap between "the guest paid" and "we have it" is weeks, and nothing
// in the system knew that gap existed.** An aggregator — Yandex Eats, Uzum
// Tezkor — takes the guest's money at the moment of the order and transfers it
// to the restaurant once a month, minus its commission. Card terminals and the
// online rails (Click, Payme, Uzum, ATMOS) work the same way on a shorter
// clock. So a February that sold twelve million through Uzum Tezkor is a
// February in which the restaurant *has* nothing yet, and a bank statement in
// March that reads nine and a half million is either correct or short by two
// hundred thousand — and until this document existed, no screen in the system
// could tell those two apart.
//
// ⚠️ **What arrives is NOT revenue.** The sale was counted the day the guest
// paid; the transfer is that same money changing location, exactly like cash
// carried from the drawer to the safe. Counting the arrival as income would
// book every aggregator sale twice — and the doubled figure looks entirely
// plausible, which is what makes it dangerous. Only the **commission** is a
// cost, and only the commission reaches the financial report.
//
// ⚠️ **All three numbers are stored as the statement says them**, and net is
// never computed as gross − commission. When the three disagree the difference
// is the most valuable thing on the page: a refund the aggregator clawed back,
// a penalty, a correction from last month. Deriving one of them would erase
// exactly the discrepancy this document exists to surface.
type Payout struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`

	// Who transferred it: a payment-method id ("click", "payme", "uzum",
	// "atmos", "card") or an aggregator's ("yandex_eats", "uzum_tezkor").
	//
	// ⚠️ Matching a payment method by the same id is what lets the screen say
	// "sold 12 000 000 through this rail, received 9 500 000" — the one
	// sentence that catches an underpayment.
	Provider string `bson:"provider" json:"provider"`
	// A frozen label, so a payout keeps its name even if the account is later
	// renamed or removed. The same rule a delivery's supplier name follows.
	ProviderName string `bson:"providerName,omitempty" json:"providerName,omitempty"`

	// The window the statement covers, local "YYYY-MM-DD".
	//
	// ⚠️ **Not the same as when it arrived**, and both are kept. The period is
	// what decides which sales are now settled; the arrival date is what the
	// bank statement shows and what the commission is costed in. A restaurant
	// paid on the 5th for March needs each of those to answer a different
	// question.
	PeriodFrom string `bson:"periodFrom" json:"periodFrom"`
	PeriodTo   string `bson:"periodTo" json:"periodTo"`

	// What the provider says it collected on our behalf.
	Gross int `bson:"gross" json:"gross"`
	// What it kept.
	Commission int `bson:"commission" json:"commission"`
	// What actually landed in the account.
	Net int `bson:"net" json:"net"`

	ReceivedAt time.Time `bson:"receivedAt" json:"receivedAt"`
	// Which account, in the restaurant's own words. Free text: this is read by
	// the person reconciling a bank statement, not by any code.
	Account string `bson:"account,omitempty" json:"account,omitempty"`
	Note    string `bson:"note,omitempty" json:"note,omitempty"`

	CreatedBy string    `bson:"createdBy,omitempty" json:"createdBy,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

// Diff is gross − commission − net: zero when the statement adds up.
//
// ⚠️ **Shown rather than corrected.** A non-zero difference is a real event —
// a refund charged back, a penalty, an adjustment carried over — and the screen
// that hides it turns a question the owner should ask into a number nobody can
// explain three months later.
func (p Payout) Diff() int { return p.Gross - p.Commission - p.Net }

// PayoutBalance is one rail: what was sold through it, what has been settled,
// and what is therefore still with the provider.
type PayoutBalance struct {
	Provider string `json:"provider"`
	Name     string `json:"name"`
	// Sales through this rail that no payout covers yet.
	Sold  int `json:"sold"`
	Count int `json:"count"`
	// ⚠️ **Where the counting starts.** Everything up to the last settled
	// period is closed; only sales after it are owed. Empty when no payout has
	// ever been recorded — and then `Sold` is every sale ever made through the
	// rail, which is a fact and not yet a debt. The screen says which of the two
	// it is looking at rather than presenting a frightening number as an
	// unpaid balance.
	SettledThrough string `json:"settledThrough,omitempty"`
	// Totals of what has actually arrived, all time.
	Received   int        `json:"received"`
	Commission int        `json:"commission"`
	LastAt     *time.Time `json:"lastAt,omitempty"`
}

// AggregatorAccount is a marketplace the restaurant sells through.
//
// ⚠️ **A payment method, not a delivery service.** `delivery_provider` answers
// "who carries the food"; this answers "who holds the money". Yandex Eats can
// be both at once, and folding them into one record would mean a restaurant
// that only uses the courier fleet suddenly has an unpaid balance.
type AggregatorAccount struct {
	// "yandex_eats", "uzum_tezkor", or a slug the restaurant types.
	ID      string `bson:"id" json:"id"`
	Name    string `bson:"name" json:"name"`
	Enabled bool   `bson:"enabled" json:"enabled"`
	// The usual rate, used only to prefill the payout form.
	//
	// ⚠️ **Never used to compute money.** The statement is the document; a
	// commission worked out from a stored percentage would be a confident
	// number about somebody else's arithmetic, and the whole point of this
	// screen is to notice when the two disagree.
	CommissionPercent float64 `bson:"commissionPercent,omitempty" json:"commissionPercent,omitempty"`
}

const (
	// ProviderYandexEats and ProviderUzumTezkor are the two marketplaces that
	// matter here today. Known by id rather than left to free text because
	// their sales have to be attributable on the till, and a rail spelled two
	// ways is two rails with half a balance each.
	ProviderYandexEats = "yandex_eats"
	ProviderUzumTezkor = "uzum_tezkor"
	// MethodCard is a bank terminal at the counter: the guest's card is charged
	// now and the acquirer settles later, minus its own commission — the same
	// shape as an aggregator on a shorter clock.
	MethodCard = "card"
	// MethodTransfer is a bank transfer straight into the account. No third
	// party holds it, so it settles nothing and appears on no payout.
	MethodTransfer = "transfer"
)

// KnownAggregators are the ids offered before anybody types one.
func KnownAggregators() []AggregatorAccount {
	return []AggregatorAccount{
		{ID: ProviderYandexEats, Name: "Yandex Eats"},
		{ID: ProviderUzumTezkor, Name: "Uzum Tezkor"},
	}
}
