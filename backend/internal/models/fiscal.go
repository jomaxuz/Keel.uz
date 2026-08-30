package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Fiscalisation (ККМ / ОФД) ----
//
// Every sale taken in the hall has to be registered with the tax committee and
// come back with a QR the guest can check. That is not our rule and not a
// feature: it is the difference between our till being a till and our till
// being a second screen the restaurant keeps beside its real one, ringing every
// order twice.
//
// ⚠️ **We do not fiscalise anything ourselves and must never look as if we do.**
// The registered party is a virtual cash register in the tax committee's
// registry; we hand it a receipt and it files it. Same trade as the payment
// providers: they hold the card, we hold none of it — here they hold the
// certification, and the restaurant's contract is with them.
//
// The online half of this is already solved and not by us either: a card
// payment through ATMOS carries the basket to ATMOS, and ATMOS files the fiscal
// receipt (see handlers/payatmos.go). What has no such path is cash and card
// taken at the counter, which is exactly what the till screen introduced.

// FiscalSettings is one branch's connection to its virtual cash register.
//
// Per **branch**, for the same reason POSSettings is: a cash register is
// registered to a place, and a chain's second kitchen files its own takings.
// A receipt filed against the wrong branch is not a formatting mistake, it is a
// misreported location of a sale.
//
// In its own collection rather than on the branch document, for the same reason
// again — the branch goes to the site in full, and these are credentials.
type FiscalSettings struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	BranchID primitive.ObjectID `bson:"branchId" json:"branchId"`

	// "" | "multikassa" | "firstofd" | "epos" | "regos" | "hippo" | "simurg"
	Provider string `bson:"provider" json:"provider"`
	Enabled  bool   `bson:"enabled" json:"enabled"`

	// СТИР / ИНН — the taxpayer the receipts are filed under. Usually the same
	// number for every branch of one company, but stored per branch anyway: a
	// group that runs two legal entities is ordinary, and there is no screen on
	// which somebody would notice the receipts going out under the other one.
	TIN string `bson:"tin" json:"tin"`

	// The default VAT rate for this branch, as a percentage. A dish may override
	// it (MenuItem.VatPercent); almost none do.
	//
	// ⚠️ **A pointer, and enabling without it is refused.** Zero is a real,
	// common answer — plenty of restaurants are not VAT payers — which means an
	// int cannot tell "not a payer" from "nobody filled this in", and the two
	// produce the same receipt while meaning opposite things to an inspector.
	// The usual "zero value is today's behaviour" rule needs a today to fall
	// back to, and a tax rate has none. So the panel has to be told.
	VatPercent *int `bson:"vatPercent,omitempty" json:"vatPercent,omitempty"`

	// One drawer per provider, never a shared credentials field.
	//
	// ⚠️ Same rule as the map keys and the POS logins: an owner who tries one
	// provider and goes back to the previous one would otherwise hand the first
	// one the second one's password. There the symptom was a blank map; here it
	// is a receipt that does not get filed, which nobody sees until the receipt
	// is the one being asked about.
	//
	// ⚠️ The field names below are a **scaffold, not a transcription**. None of
	// these providers publishes its API; the shape is settled against the real
	// documentation when a contract is signed, and until then the drawer holds
	// what every one of them is certain to need. See internal/fiscal.
	Multikassa FiscalCreds `bson:"multikassa" json:"multikassa"`
	FirstOFD   FiscalCreds `bson:"firstofd" json:"firstofd"`
	EPOS       FiscalCreds `bson:"epos" json:"epos"`
	Regos      FiscalCreds `bson:"regos" json:"regos"`
	Hippo      FiscalCreds `bson:"hippo" json:"hippo"`
	Simurg     FiscalCreds `bson:"simurg" json:"simurg"`
	// ⚠️ Its own drawer, not Multikassa's: Rahmat's cloud register and the
	// program Rahmat resells for the till computer are two products with two
	// sets of credentials, and the panel's Multikassa row is the second one.
	Rahmat FiscalCreds `bson:"rahmat" json:"rahmat"`
	QPOS   FiscalCreds `bson:"qpos" json:"qpos"`
	Arca   FiscalCreds `bson:"arca" json:"arca"`

	// ---- The relay, for registers no browser can reach ----
	//
	// A small program on the register's own PC that connects **outwards** to us,
	// asks for filings and makes them against localhost. It exists because the
	// browser route depends on three things we do not control — mixed content,
	// private network access, and whether the register answers CORS at all —
	// and an outbound connection is subject to none of them.
	//
	// ⚠️ **The address in the token is the authentication**, exactly as it is
	// for the onlinePBX and Telegram webhooks: the agent has no account and no
	// user behind it. Rotatable, compared in constant time, and never returned
	// to the panel after it is first shown.
	AgentToken string `bson:"agentToken,omitempty" json:"-"`
	// ⚠️ **When the agent last asked for work — a timestamp, not "connected".**
	// The relay's whole failure mode is going quiet: the PC is rebooted for
	// Windows updates and nobody notices, because nothing on any screen changes.
	// A stored flag would still say "connected" a week later; an hour-old
	// timestamp says what is actually true. Same rule as lastEventAt.
	AgentSeenAt *time.Time `bson:"agentSeenAt,omitempty" json:"agentSeenAt,omitempty"`

	// ---- Ending the register's day ----
	//
	// ⚠️ **A request, not an action**, and that is forced by where things are:
	// the cash shift is closed from the panel, which is very often a laptop
	// somewhere else, while the Z-report has to be filed by something standing
	// on the restaurant's network. So closing the drawer *asks* for the day to
	// end, and whoever can reach the register — the relay, or the till screen
	// next time it is open — carries it out.
	//
	// One nullable timestamp rather than a queue, for the same reason there is
	// no job collection: the state is small, there is at most one of it, and a
	// second record of the same fact is a second thing that can disagree.
	CloseDayRequestedAt *time.Time `bson:"closeDayRequestedAt,omitempty" json:"closeDayRequestedAt,omitempty"`
	// Who asked, so the Z-report carries a name — the register prints one.
	CloseDayBy string `bson:"closeDayBy,omitempty" json:"closeDayBy,omitempty"`
	// The cash shift the resulting Z-report belongs to, remembered because by
	// the time the register answers, that shift is closed and no longer the
	// "open" one anything would find by searching.
	CloseDayShiftID primitive.ObjectID `bson:"closeDayShiftId,omitempty" json:"-"`

	// What the last connection check said, so the panel can show it without
	// dialling the provider on every page load.
	LastCheckAt *time.Time `bson:"lastCheckAt,omitempty" json:"lastCheckAt,omitempty"`
	LastCheckOK bool       `bson:"lastCheckOk" json:"lastCheckOk"`
	LastCheck   string     `bson:"lastCheck" json:"lastCheck"`

	// ⚠️ **The most useful line on the page**, and deliberately a timestamp
	// rather than a flag: when a receipt was last filed. The check button proves
	// the credentials worked when it was pressed; this answers "are sales being
	// registered right now", and a stored flag goes stale the moment the hour
	// moves past it. Same lesson as lastEventAt and lastUpdateAt.
	LastReceiptAt *time.Time `bson:"lastReceiptAt,omitempty" json:"lastReceiptAt,omitempty"`

	// The most recent real failure, which is a different and more important
	// question than the test button's: it is about a guest standing at the
	// counter, not about us checking. Cleared by a success.
	LastErrorAt *time.Time `bson:"lastErrorAt,omitempty" json:"lastErrorAt,omitempty"`
	LastError   string     `bson:"lastError" json:"lastError"`

	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// FiscalCreds is one provider's drawer.
//
// The same shape for all six because at this level they genuinely are the same
// shape — an account, a secret, the cash register this branch files under, and
// a base URL so a sandbox can be pointed at. What differs between them is how
// those are named on the wire, and that belongs in the adapter rather than in
// six near-identical structs here.
//
// ⚠️ Secrets carry `json:"-"`. The settings response says only whether a
// secret is present; an empty secret on save means "keep the stored one", never
// "delete it" — the rule every credentials screen in this codebase follows,
// because an owner fixing a typo in the TIN must not silently unfiscalise the
// restaurant.
type FiscalCreds struct {
	// Login, client id, merchant id — whatever the provider calls the account.
	Login    string `bson:"login" json:"login"`
	Password string `bson:"password" json:"-"`
	// A pre-issued token, for providers that hand one out instead of taking a
	// login. Both are kept: which one a provider uses is not something the
	// owner should have to know before choosing them from the list.
	Token string `bson:"token" json:"-"`
	// The cash register / terminal this branch's receipts are filed under.
	RegisterID string `bson:"registerId" json:"registerId"`
	// Overridable so a sandbox can be pointed at before real money moves.
	BaseURL string `bson:"baseUrl" json:"baseUrl"`
}

// HasSecret reports whether any secret is stored, for the panel's flag.
func (c FiscalCreds) HasSecret() bool { return c.Password != "" || c.Token != "" }

// Filing states. "pending" is set before the register is asked, so a filing
// whose answer never came back is a visible unfinished one rather than an order
// that looks as if it was never meant to have a receipt.
const (
	FiscalPending = "pending"
	FiscalFiled   = "filed"
	FiscalFailed  = "failed"
)

// FiscalDay is the register's own account of a day, stored on the cash shift it
// belongs to.
//
// ⚠️ **Kept because it is a second, independent count of the same takings.**
// The shift's `expected` figure is built from the orders we recorded; these
// numbers come from the machine that filed them with the state. When a drawer
// is short, the first useful question is which of the two the cash agrees with
// — and without this the question cannot be asked, only argued about.
//
// ⚠️ It does **not** replace or correct `expected`. That figure is frozen at
// closing time on purpose (see AdminCloseCashShift), and a shortfall that
// rewrote itself when a second source arrived would be a shortfall nobody could
// investigate.
type FiscalDay struct {
	// The Z-report's sequence number — what an inspector asks for.
	Number string `bson:"number,omitempty" json:"number,omitempty"`
	// Whole so'm, as the register totalled them.
	SaleCash  int `bson:"saleCash" json:"saleCash"`
	SaleCard  int `bson:"saleCard" json:"saleCard"`
	SaleTotal int `bson:"saleTotal" json:"saleTotal"`
	SaleCount int `bson:"saleCount" json:"saleCount"`
	// Separate from sales: a day with heavy refunds that happens to balance is
	// a different story from a quiet one, and netting them hides it.
	RefundTotal int `bson:"refundTotal" json:"refundTotal"`

	ClosedAt time.Time `bson:"closedAt" json:"closedAt"`
	// Why the day could not be ended, when it could not. Kept on the shift
	// rather than only in a log because "the Z-report was not filed" is a fact
	// about *this* day that somebody will ask about later.
	Error string `bson:"error,omitempty" json:"error,omitempty"`
}

// FiscalReceipt is what came back from the provider, recorded on the order.
//
// ⚠️ **Its own field, not folded into the payment status.** They are different
// facts that move at different times: the guest pays at the counter in a
// second, and the receipt is filed by somebody else's server over a network
// that can be down. One field for both would either lie about the money or lose
// the record of the filing — the same reasoning that keeps `pos.till` separate
// from `pos.status`.
type FiscalReceipt struct {
	// "" | "pending" | "filed" | "failed"
	Status string `bson:"status" json:"status"`
	// Which provider filed it. Stored per receipt rather than read from the
	// settings, because a restaurant that switches providers still has to be
	// able to say who filed last March's receipts.
	Provider string `bson:"provider" json:"provider"`

	// The fiscal sign and the QR the guest checks. Both come from the provider
	// and neither is ours to compute.
	FiscalSign string `bson:"fiscalSign,omitempty" json:"fiscalSign,omitempty"`
	QRText     string `bson:"qrText,omitempty" json:"qrText,omitempty"`
	ReceiptID  string `bson:"receiptId,omitempty" json:"receiptId,omitempty"`

	FiledAt *time.Time `bson:"filedAt,omitempty" json:"filedAt,omitempty"`
	// Why it failed, in the provider's words — kept for the owner, never shown
	// to a guest (the SMS gateway lesson: a stranger at the counter can do
	// nothing with "STORE_NOT_FOUND", and it tells them about our contract).
	Error string `bson:"error,omitempty" json:"error,omitempty"`
	// How many times we have tried, so a retry loop cannot run forever against
	// a receipt that will never be accepted.
	Attempts int `bson:"attempts,omitempty" json:"attempts,omitempty"`
}
