package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- POS integration ----
//
// The till the restaurant already runs. Settings hang off a **branch**, not the
// company: a chain with two kitchens has two iiko terminal groups, and an order
// printed at the wrong one is worse than an order not printed at all.
//
// Like the payment credentials, these live in their own collection rather than
// on the branch document — a branch is returned to the site in full, and a
// merchant password one forgotten `json:"-"` away from a public response is a
// password waiting to leak.

// POSSettings is one branch's till connection.
type POSSettings struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	BranchID primitive.ObjectID `bson:"branchId" json:"branchId"`

	// "" | "iiko" | "syrve" | "clopos" | "poster" | "rkeeper"
	Provider string `bson:"provider" json:"provider"`
	Enabled  bool   `bson:"enabled" json:"enabled"`
	// Send the order by itself the moment it is confirmed. Off means the
	// operator presses "send" — which is what a restaurant that is still
	// checking the integration wants.
	AutoSend bool `bson:"autoSend" json:"autoSend"`

	Iiko struct {
		APILogin       string `bson:"apiLogin" json:"-"`
		OrganizationID string `bson:"organizationId" json:"organizationId"`
		TerminalGroup  string `bson:"terminalGroup" json:"terminalGroup"`
		OrderTypeID    string `bson:"orderTypeId" json:"orderTypeId"`
		PaymentTypeID  string `bson:"paymentTypeId" json:"paymentTypeId"`
		BaseURL        string `bson:"baseUrl" json:"baseUrl"`
	} `bson:"iiko" json:"iiko"`

	// Syrve is iiko's international edition and speaks the same API, but its
	// credentials are stored separately on purpose: a restaurant that tries
	// iiko, then switches to Syrve, must not end up dialling one with the
	// other's apiLogin. Same shape, different drawer.
	Syrve struct {
		APILogin       string `bson:"apiLogin" json:"-"`
		OrganizationID string `bson:"organizationId" json:"organizationId"`
		TerminalGroup  string `bson:"terminalGroup" json:"terminalGroup"`
		OrderTypeID    string `bson:"orderTypeId" json:"orderTypeId"`
		PaymentTypeID  string `bson:"paymentTypeId" json:"paymentTypeId"`
		BaseURL        string `bson:"baseUrl" json:"baseUrl"`
	} `bson:"syrve" json:"syrve"`

	Poster struct {
		Token string `bson:"token" json:"-"`
		// Poster prices and files orders per sales point, so this is not
		// optional the way a "default branch" would be.
		SpotID  int    `bson:"spotId" json:"spotId"`
		BaseURL string `bson:"baseUrl" json:"baseUrl"`
	} `bson:"poster" json:"poster"`

	Clopos struct {
		ClientID     string `bson:"clientId" json:"clientId"`
		ClientSecret string `bson:"clientSecret" json:"-"`
		Brand        string `bson:"brand" json:"brand"`
		IntegratorID string `bson:"integratorId" json:"integratorId"`
		VenueID      int    `bson:"venueId" json:"venueId"`
		SaleTypeID   int    `bson:"saleTypeId" json:"saleTypeId"`
		BaseURL      string `bson:"baseUrl" json:"baseUrl"`
	} `bson:"clopos" json:"clopos"`

	RKeeper struct {
		URL      string `bson:"url" json:"url"`
		Login    string `bson:"login" json:"login"`
		Password string `bson:"password" json:"-"`
		Station  string `bson:"station" json:"station"`
		Anchor   string `bson:"anchor" json:"anchor"`
		Token    string `bson:"token" json:"-"`
	} `bson:"rkeeper" json:"rkeeper"`

	// What the last connection check said, so the panel can show it without
	// hitting the till on every page load.
	LastCheckAt *time.Time `bson:"lastCheckAt,omitempty" json:"lastCheckAt,omitempty"`
	LastCheckOK bool       `bson:"lastCheckOk" json:"lastCheckOk"`
	LastCheck   string     `bson:"lastCheck" json:"lastCheck"`

	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// POSMapping ties one of our dishes to its id in one branch's till.
//
// Its own collection rather than a field on the dish, for one reason: the menu
// belongs to a **brand** and the till belongs to a **branch**. A brand served by
// two branches on two separate iiko accounts has two different ids for the same
// lag'mon, and a single field on the dish can only hold one of them.
type POSMapping struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID   primitive.ObjectID `bson:"branchId" json:"branchId"`
	MenuItemID primitive.ObjectID `bson:"menuItemId" json:"menuItemId"`
	// The id in the till, and its name there — copied so the mapping screen
	// reads without a round trip to the POS, and still reads when the till is
	// unreachable.
	POSProductID   string    `bson:"posProductId" json:"posProductId"`
	POSProductName string    `bson:"posProductName" json:"posProductName"`
	UpdatedAt      time.Time `bson:"updatedAt" json:"updatedAt"`
}

// POS states on an order.
//
// ⚠️ These describe **our handover**, not the till's opinion of the order. The
// two are genuinely different facts and conflating them is the mistake this
// bridge is most likely to make: Poster files an order perfectly and leaves it
// at `status: 0` until a cashier presses accept, so "we sent it" and "the
// kitchen has it" can be hours apart. The till's own answer lives in
// OrderPOS.TillState.
const (
	// Never sent — the usual state for a restaurant with no till connected.
	POSIdle = ""
	// Handed over successfully. Says nothing about whether the till accepted.
	POSSent = "sent"
	// The till refused it, or could not be reached. The reason is kept; this
	// is the state an operator has to be able to see and retry from.
	POSFailed = "failed"
	// Sent, but the till had not confirmed by the time we stopped waiting.
	POSPending = "pending"
)

// Till-side states, as reported by the POS itself (pos.Status.State), plus one
// of ours for tills that cannot be asked.
const (
	// Filed, but nobody at the till has taken it yet. Poster's `status: 0`.
	TillWaiting = "waiting"
	// A real till transaction now.
	TillAccepted = "accepted"
	// Withdrawn at the till, by someone standing there.
	TillCancelled = "cancelled"
	// This POS has no way to answer. r_keeper's order lifecycle is a till
	// session, not a queryable document. Recorded once so the poller stops
	// asking and the panel stops implying an answer is coming.
	TillUnsupported = "unsupported"
)

// OrderTill is what the till says became of an order we handed it.
//
// Separate from the send outcome above because it answers a different question
// and changes on a different clock: the handover is over in a second, the
// acceptance happens whenever somebody at the counter looks up.
type OrderTill struct {
	// One of the Till* constants. Empty means never asked.
	State string `bson:"state,omitempty" json:"state,omitempty"`
	// The till's own wording, for the operator who wants the detail — usually
	// the check number once accepted.
	Raw string `bson:"raw,omitempty" json:"raw,omitempty"`
	// When we last asked. ⚠️ The most useful field on the whole struct: a
	// state without a time silently ages into a lie, exactly like the
	// `lastEventAt` rule on the PBX and Telegram settings pages.
	CheckedAt *time.Time `bson:"checkedAt,omitempty" json:"checkedAt,omitempty"`
	// When it became a real transaction. Set once, never cleared.
	AcceptedAt *time.Time `bson:"acceptedAt,omitempty" json:"acceptedAt,omitempty"`
	// Why the last question failed, when it did. Not an order failure: the
	// order is at the till either way.
	Error string `bson:"error,omitempty" json:"error,omitempty"`
}

// OrderPOS is what happened when this order was pushed to the till.
//
// Kept on the order rather than in a side table because it is read on exactly
// one screen — the receipt — and the question it answers is always about one
// order: "did the kitchen get this?"
type OrderPOS struct {
	Provider string `bson:"provider" json:"provider"`
	Status   string `bson:"status" json:"status"`
	// The till's own id, needed to ask after it later.
	POSOrderID string `bson:"posOrderId,omitempty" json:"posOrderId,omitempty"`
	// Anything worth showing: the till's order number, or why it refused.
	Note  string `bson:"note,omitempty" json:"note,omitempty"`
	Error string `bson:"error,omitempty" json:"error,omitempty"`
	// How many times sending has been attempted, so a retry loop is visible
	// rather than silent.
	Attempts int        `bson:"attempts" json:"attempts"`
	SentAt   *time.Time `bson:"sentAt,omitempty" json:"sentAt,omitempty"`
	// What the till itself reports, once we have asked it. Absent on orders
	// sent before this existed, and on tills that cannot answer.
	Till *OrderTill `bson:"till,omitempty" json:"till,omitempty"`
}
