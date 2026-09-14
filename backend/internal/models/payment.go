package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Online payment: Payme, Click, Uzum ----
//
// Three providers, one shape. Each of them works the same way underneath: the
// guest is sent to the provider's checkout, pays there, and the provider then
// calls *this* server to say so. The browser coming back is not evidence of
// anything — it is trivially faked and it also simply fails to happen when
// somebody pays and closes the tab. So an order is marked paid by exactly one
// thing: the provider's server-to-server call, verified.
//
// What that buys, and why the code below is shaped around it:
//
//   • the amount is always checked against the stored order, never taken from
//     the callback;
//   • every state change is idempotent, because all three providers retry and
//     Payme in particular will call PerformTransaction twice quite happily;
//   • the credentials live here, in their own collection, and are never
//     marshalled into anything the site returns.

// PaymentProvider ids, as stored on the order and the ledger.
const (
	ProviderCash = "cash"
	// ⚠️ **Not a way of paying — a way of not paying yet.** A regular who eats
	// today and settles on Friday is ordinary in this business; a sale that
	// leaves no record is not, and that is what the paper book by the till
	// produces. A check closed this way is `delivered` and `unpaid`: it is not
	// takings until the repayment is recorded, and on that day it becomes
	// takings with the method the money actually arrived in.
	MethodDebt    = "debt"
	ProviderPayme = "payme"
	ProviderClick = "click"
	ProviderUzum  = "uzum"
	ProviderAtmos = "atmos"
)

// PaymeSettings is what the Payme cabinet issues.
//
// The key is the merchant key from the cabinet: Payme signs every Merchant API
// call with it as HTTP Basic auth (login "Paycom"). Test and live are separate
// keys for the same merchant id, which is why both are kept — switching to the
// sandbox must not mean retyping the live key from a screenshot.
type PaymeSettings struct {
	Enabled    bool   `bson:"enabled" json:"enabled"`
	MerchantID string `bson:"merchantId" json:"merchantId"`
	Key        string `bson:"key" json:"-"`
	TestKey    string `bson:"testKey" json:"-"`
	// Sandbox mode: the checkout goes to the test host and the test key is the
	// one callbacks are checked against.
	TestMode bool `bson:"testMode" json:"testMode"`
	// The name of the field the cabinet was configured with — Payme calls it the
	// "account" and the merchant chooses what goes in it. Ours carries the order
	// number. Configurable because the cabinet is filled in by a human at the
	// bank, and "order_id" is a convention, not a rule.
	AccountField string `bson:"accountField" json:"accountField"`
}

// ClickSettings is the SHOP API scheme: Click calls Prepare and then Complete,
// both signed with an MD5 of the request plus the secret key.
type ClickSettings struct {
	Enabled    bool   `bson:"enabled" json:"enabled"`
	ServiceID  string `bson:"serviceId" json:"serviceId"`
	MerchantID string `bson:"merchantId" json:"merchantId"`
	// Needed only by the Click-API (invoices, card tokens); kept because the
	// cabinet issues it alongside the rest and it is easier to store than to
	// find again later.
	MerchantUserID string `bson:"merchantUserId" json:"merchantUserId"`
	SecretKey      string `bson:"secretKey" json:"-"`
}

// UzumSettings is Uzum Bank's merchant API: the bank issues a service id and a
// Basic auth pair, then calls check/create/confirm/reverse/status here.
type UzumSettings struct {
	Enabled   bool   `bson:"enabled" json:"enabled"`
	ServiceID string `bson:"serviceId" json:"serviceId"`
	Login     string `bson:"login" json:"login"`
	Password  string `bson:"password" json:"-"`
	// Same idea as Payme's account field: the bank's cabinet is configured with
	// the parameter names it will send inside `params`.
	AccountField string `bson:"accountField" json:"accountField"`
}

// AtmosSettings is the ATMOS gateway.
//
// ⚠️ **Deliberately the hosted invoice, not ATMOS's card API.**
//
// ATMOS offers both. Its `/merchant/pay/*` endpoints take the card number and
// expiry directly, which would put the guest's PAN on this server — and, in a
// platform where every restaurant runs its own container, would put *every
// tenant* in PCI DSS scope. `/checkout/invoice/create` returns a
// checkout.atmos.uz URL instead: the guest types their card on ATMOS's page,
// exactly as with Payme, Click and Uzum, and the card never reaches us.
//
// Anybody moving this to the direct API is not making an optimisation; they
// are changing what this system is legally responsible for.
type AtmosSettings struct {
	Enabled bool `bson:"enabled" json:"enabled"`
	// The store this restaurant was issued in the ATMOS cabinet.
	StoreID string `bson:"storeId" json:"storeId"`
	// OAuth2 client credentials for apigw.atmos.uz.
	ConsumerKey    string `bson:"consumerKey" json:"-"`
	ConsumerSecret string `bson:"consumerSecret" json:"-"`
	// The key ATMOS signs its callbacks with. Separate from the OAuth pair:
	// one authenticates us to them, this one authenticates them to us, and
	// conflating the two is how a signature check ends up verifying nothing.
	APIKey string `bson:"apiKey" json:"-"`
	// Overridable for the sandbox; empty = the production gateway.
	BaseURL string `bson:"baseUrl" json:"baseUrl"`
}

// InStoreSettings is the counter rails, one drawer per provider.
//
// ⚠️ **A map, not a field each, and that decision is already written down.**
// The fiscal settings were built the other way first — a struct per provider,
// a switch per read — and adding the seventh provider meant editing four
// separate lists, where forgetting one produced a screen that saved
// credentials nothing ever read. See docs/DECISIONS.md → "Fiskal provayderlar".
// The same shape here means a new rail is a row in instore.Providers() and an
// adapter, and nothing else.
type InStoreSettings struct {
	// Which rails are switched on, by provider id. ⚠️ Separate from whether
	// credentials exist: an owner mid-migration has both providers' keys typed
	// in and wants exactly one of them offered at the counter.
	Enabled map[string]bool `bson:"enabled" json:"enabled"`
	// One drawer per provider id.
	Creds map[string]InStoreCreds `bson:"creds" json:"creds"`
}

// InStoreCreds is one provider's drawer.
//
// The same four boxes for every rail because at this level they are the same
// four: the service the money is filed against, the till inside it, the secret
// the requests are signed with, and a host so a sandbox can be pointed at.
// What each provider *calls* them is the adapter's business.
//
// ⚠️ The secret carries `json:"-"` and an empty secret on save means "keep the
// stored one" — the rule every credentials screen here follows, because an
// owner correcting a service id must not silently stop the till taking cards.
type InStoreCreds struct {
	ServiceID string `bson:"serviceId" json:"serviceId"`
	UserID    string `bson:"userId" json:"userId"`
	SecretKey string `bson:"secretKey" json:"-"`
	BaseURL   string `bson:"baseUrl" json:"baseUrl"`
}

// HasSecret reports whether a secret is stored, for the panel's flag.
func (c InStoreCreds) HasSecret() bool { return c.SecretKey != "" }

// InStoreCreds returns one rail's drawer, and whether it is switched on.
//
// ⚠️ **Both answers from one call.** Asked separately, the two questions drift:
// the offer list checks `Enabled`, the charge checks the credentials, and a
// rail enabled with half its keys typed in becomes a button that fails in front
// of a guest — which is the exact failure `Configured` was written against for
// the website.
func (s *PaymentSettings) InStoreCreds(provider string) (InStoreCreds, bool) {
	c := s.InStore.Creds[provider]
	on := s.InStore.Enabled[provider] &&
		c.ServiceID != "" && c.UserID != "" && c.SecretKey != ""
	return c, on
}

// PaymentSettings is the singleton holding every provider's credentials.
//
// Deliberately its own collection rather than a field on `restaurant`: the
// restaurant profile is returned in full to every visitor of the site, and a
// secret that lives one forgotten `json:"-"` away from a public response is a
// secret waiting to leak. Nothing here is ever part of a public payload.
type PaymentSettings struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	// Where the provider sends the guest back to. Empty = the site's own
	// tracking page, which is what a restaurant wants and never has to type.
	ReturnURL string        `bson:"returnUrl" json:"returnUrl"`
	Payme     PaymeSettings `bson:"payme" json:"payme"`
	Click     ClickSettings `bson:"click" json:"click"`
	Uzum      UzumSettings  `bson:"uzum" json:"uzum"`
	Atmos     AtmosSettings `bson:"atmos" json:"atmos"`
	// The counter rails: a cashier scanning the guest's code. Separate from the
	// four above even where the merchant is the same company, because the
	// credentials genuinely are different — CLICK Pass signs with the Merchant
	// API pair, not the SHOP API secret the website uses, and Uzum FastPay is
	// issued against a different service entirely. See internal/instore.
	InStore InStoreSettings `bson:"inStore" json:"inStore"`
	// The marketplaces the restaurant sells through — Yandex Eats, Uzum Tezkor.
	//
	// ⚠️ **Here rather than in `delivery_provider`**, which answers a different
	// question: that one is "who carries the food", this is "who holds the
	// money". Yandex can be both at once, and one record for both would give a
	// restaurant that merely hires the courier fleet an unpaid balance it does
	// not have. See models/payout.go.
	Aggregators []AggregatorAccount `bson:"aggregators,omitempty" json:"aggregators"`
	// The buttons the till offers for money taken at the counter — cash, the
	// terminal, a bank transfer, under whatever names the owner gives them. See
	// TillMethod. Empty means the three defaults.
	TillMethods []TillMethod `bson:"tillMethods,omitempty" json:"tillMethods"`
	UpdatedAt   time.Time    `bson:"updatedAt" json:"updatedAt"`
}

// Configured reports whether a provider can actually take money, which is what
// the checkout offers the guest — an enabled provider with half its
// credentials typed in is worse than a hidden one, because the guest only
// finds out at the bank.
// EnabledAggregators is the marketplaces a till may take an order for.
//
// ⚠️ Empty slice, never nil: this is marshalled straight into a settings
// response and `null.map` is a blank screen.
func (s *PaymentSettings) EnabledAggregators() []AggregatorAccount {
	out := []AggregatorAccount{}
	for _, a := range s.Aggregators {
		if a.Enabled && a.ID != "" {
			out = append(out, a)
		}
	}
	return out
}

func (s *PaymentSettings) Configured(provider string) bool {
	switch provider {
	case ProviderPayme:
		return s.Payme.Enabled && s.Payme.MerchantID != "" && s.PaymeKey() != ""
	case ProviderClick:
		return s.Click.Enabled && s.Click.ServiceID != "" &&
			s.Click.MerchantID != "" && s.Click.SecretKey != ""
	case ProviderUzum:
		return s.Uzum.Enabled && s.Uzum.ServiceID != "" &&
			s.Uzum.Login != "" && s.Uzum.Password != ""
	case ProviderAtmos:
		// The callback key counts: without it ATMOS's confirmation cannot be
		// verified, and an unverifiable confirmation is one we must refuse —
		// so a gateway configured without it can never actually take money.
		return s.Atmos.Enabled && s.Atmos.StoreID != "" &&
			s.Atmos.ConsumerKey != "" && s.Atmos.ConsumerSecret != "" &&
			s.Atmos.APIKey != ""
	case ProviderCash:
		// Cash needs no configuring and can never be switched off: somebody has
		// to be able to order when the card rails are down.
		return true
	}
	return false
}

// PaymeKey is the key callbacks are checked against, which depends on the mode.
func (s *PaymentSettings) PaymeKey() string {
	if s.Payme.TestMode {
		return s.Payme.TestKey
	}
	return s.Payme.Key
}

// ---- The order's side of it ----

// Payment states as they appear on an order.
const (
	// Cash, and anything else settled off-line. Not a failure.
	PayUnpaid = "unpaid"
	// The guest was sent to a provider and has not come back paid yet.
	PayPending = "pending"
	PayPaid    = "paid"
	// The money went back: a cancelled order, or the provider reversing.
	PayRefunded = "refunded"
)

// Ledger states for one provider transaction. The numbers are Payme's own
// (1 created, 2 performed, -1 cancelled while pending, -2 cancelled after
// performing) because Payme is the only provider that requires them to be
// echoed back exactly; the other two are mapped onto the same set so there is
// one state machine to reason about rather than three.
const (
	TxnCreated        = 1
	TxnPerformed      = 2
	TxnCancelledSetup = -1
	TxnCancelledPaid  = -2
)

// Payment is one provider transaction against one order.
//
// It is a ledger, not a status field: a provider may create a transaction,
// cancel it, and have the guest try again with another card, and "what
// actually happened to order 4218" has to stay answerable afterwards.
type Payment struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrderID     primitive.ObjectID `bson:"orderId" json:"orderId"`
	OrderNumber string             `bson:"orderNumber" json:"orderNumber"`
	BranchID    primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	Provider    string             `bson:"provider" json:"provider"`
	// The provider's own id for this transaction: Payme's `id`, Click's
	// click_trans_id, Uzum's transId. Unique per provider — the guard against
	// processing a retried callback twice.
	ProviderTxnID string `bson:"providerTxnId" json:"providerTxnId"`
	// In so'm, copied from the order at creation. The callback's amount is
	// checked against this and never replaces it.
	Amount int `bson:"amount" json:"amount"`
	State  int `bson:"state" json:"state"`
	// Why it was cancelled, in the provider's own vocabulary.
	Reason int `bson:"reason,omitempty" json:"reason,omitempty"`

	// Payme requires the exact millisecond timestamps it was given to be echoed
	// back on every later CheckTransaction, so they are stored rather than
	// derived from the Go times below.
	CreateTimeMs  int64 `bson:"createTimeMs" json:"createTimeMs"`
	PerformTimeMs int64 `bson:"performTimeMs" json:"performTimeMs"`
	CancelTimeMs  int64 `bson:"cancelTimeMs" json:"cancelTimeMs"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// ---- The till's own payment buttons ----

// The three kinds of money a counter takes, and the only values a till button
// may have as its kind.
const (
	TillKindCash     = "cash"
	TillKindCard     = "card"
	TillKindTransfer = "transfer"
)

// TillMethod is one way of paying the owner offers at the counter: "Naqd",
// "Humo terminal", "Beznal (hisob raqamga)".
//
// ⚠️ **A name over a kind, never a new kind.** The drawer counts `cash`, the
// shift report splits `cash` / `card` / everything else, and the payouts screen
// reads `card` as the terminal's settlement. A button with its own id in
// `order.paymentMethod` would fall out of all three — a "Naqd (dollar)" sale
// missing from the drawer it went into. So the order keeps the kind, and the
// button's id and name travel beside it (Order.PaymentOptionID).
type TillMethod struct {
	ID string `bson:"id" json:"id"`
	// Empty on the three defaults, which the till names in its own language.
	Name    string `bson:"name" json:"name"`
	Kind    string `bson:"kind" json:"kind"`
	Enabled bool   `bson:"enabled" json:"enabled"`
}

// IsTillKind reports whether k is one of the three kinds a till button may be.
func IsTillKind(k string) bool {
	return k == TillKindCash || k == TillKindCard || k == TillKindTransfer
}

// DefaultTillMethods is what a till offers before the owner has set anything:
// the three buttons it always had, with ids equal to their kinds so sales made
// before this existed read the same.
func DefaultTillMethods() []TillMethod {
	return []TillMethod{
		{ID: TillKindCash, Kind: TillKindCash, Enabled: true},
		{ID: TillKindCard, Kind: TillKindCard, Enabled: true},
		{ID: TillKindTransfer, Kind: TillKindTransfer, Enabled: true},
	}
}

// TillMethodList is every till button, switched on or not.
func (s *PaymentSettings) TillMethodList() []TillMethod {
	if len(s.TillMethods) == 0 {
		return DefaultTillMethods()
	}
	return s.TillMethods
}

// EnabledTillMethods is what the till draws. ⚠️ Never nil.
func (s *PaymentSettings) EnabledTillMethods() []TillMethod {
	out := []TillMethod{}
	for _, m := range s.TillMethodList() {
		if m.Enabled && IsTillKind(m.Kind) {
			out = append(out, m)
		}
	}
	return out
}

// TillMethodByID finds a button by id — ⚠️ **switched off or not**. A sale
// rung up offline an hour ago on a button the owner has since hidden is still a
// sale on that button, and refusing it would lose the name, not the money.
func (s *PaymentSettings) TillMethodByID(id string) (TillMethod, bool) {
	for _, m := range s.TillMethodList() {
		if m.ID == id && IsTillKind(m.Kind) {
			return m, true
		}
	}
	return TillMethod{}, false
}
