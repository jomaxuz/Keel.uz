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
	ProviderCash  = "cash"
	ProviderPayme = "payme"
	ProviderClick = "click"
	ProviderUzum  = "uzum"
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
	UpdatedAt time.Time     `bson:"updatedAt" json:"updatedAt"`
}

// Configured reports whether a provider can actually take money, which is what
// the checkout offers the guest — an enabled provider with half its
// credentials typed in is worse than a hidden one, because the guest only
// finds out at the bank.
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
