package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- The open API: other programs reading this restaurant ----
//
// See docs/DECISIONS.md → "Ochiq API: kalitlar va webhook'lar". Two halves:
// keys a program presents when it asks (`api_key`), and addresses we call when
// something happens (`webhook_endpoint`, with its queue in `webhook_delivery`).

// API key scopes. ⚠️ Part of the published contract: a key is issued with
// these strings in it, and renaming one silently locks out every program that
// holds such a key.
const (
	ScopeMenuRead   = "menu:read"
	ScopeOrdersRead = "orders:read"
	// Every movement of money, classified — for an accounting service the
	// restaurant contracts with. ⚠️ **Carries no customer data**: an
	// accountant needs amounts, not guests, and this is the scope an owner can
	// hand out without handing over the customer list.
	ScopeFinanceRead = "finance:read"
)

// APIScopes is every scope, in the order the panel lists them.
var APIScopes = []string{ScopeMenuRead, ScopeOrdersRead, ScopeFinanceRead}

// APIKey is a credential the owner hands to another program.
//
// ⚠️ **Only the hash is stored**, as with the Uzum Tezkor secret: the key is
// shown once, when it is made, and "make a new one" is the answer to "we lost
// it". A key the panel could show again is in every screenshot from then on.
type APIKey struct {
	ID   primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name string             `bson:"name" json:"name"`
	// The first characters of the key, so the owner can tell which of theirs a
	// log line or a partner's email is about. Not enough to use.
	Prefix string `bson:"prefix" json:"prefix"`
	// SHA-256 of the whole key, unique. A random 256-bit key needs no bcrypt:
	// there is no dictionary to try, and the lookup happens on every request.
	Hash   string   `bson:"hash" json:"-"`
	Scopes []string `bson:"scopes" json:"scopes"`

	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	CreatedBy   string             `bson:"createdBy,omitempty" json:"createdBy,omitempty"`
	CreatedByID primitive.ObjectID `bson:"createdById,omitempty" json:"-"`
	// Written at most once every few minutes (see handlers/openapi.go) — a key
	// in a polling loop must not turn every read into a write.
	LastUsedAt *time.Time `bson:"lastUsedAt,omitempty" json:"lastUsedAt,omitempty"`
	// ⚠️ **Revoked, not deleted**: the row is what answers "which program was
	// that, and who let it in" after the fact.
	RevokedAt *time.Time `bson:"revokedAt,omitempty" json:"revokedAt,omitempty"`
}

// Has reports whether the key carries scope.
func (k *APIKey) Has(scope string) bool {
	for _, s := range k.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// Webhook events. ⚠️ Published strings, like the scopes.
const (
	EventOrderCreated       = "order.created"
	EventOrderStatusChanged = "order.status_changed"
)

// WebhookEvents is every event, in the order the panel lists them.
var WebhookEvents = []string{EventOrderCreated, EventOrderStatusChanged}

// WebhookEndpoint is an address we call when something happens.
type WebhookEndpoint struct {
	ID     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	URL    string             `bson:"url" json:"url"`
	Events []string           `bson:"events" json:"events"`
	// ⚠️ **Stored in clear, unlike an API key's hash**, because we sign with it
	// on every delivery. Never sent back to the browser after the answer that
	// created or rotated it — `json:"-"`, and this document is only ever read
	// by the panel through its own response shape.
	Secret  string `bson:"secret" json:"-"`
	Enabled bool   `bson:"enabled" json:"enabled"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
	// The last outcome, for the panel's one-line health. ⚠️ Failures are
	// counted, never acted on by switching the endpoint off: a receiver down
	// for a night is not a decision to stop, and the owner is who decides.
	LastSuccessAt       *time.Time `bson:"lastSuccessAt,omitempty" json:"lastSuccessAt,omitempty"`
	LastFailureAt       *time.Time `bson:"lastFailureAt,omitempty" json:"lastFailureAt,omitempty"`
	LastError           string     `bson:"lastError,omitempty" json:"lastError,omitempty"`
	ConsecutiveFailures int        `bson:"consecutiveFailures" json:"consecutiveFailures"`
}

// Subscribes reports whether the endpoint wants event.
func (e *WebhookEndpoint) Subscribes(event string) bool {
	for _, s := range e.Events {
		if s == event {
			return true
		}
	}
	return false
}

// Delivery states.
const (
	DeliveryPending   = "pending"
	DeliveryDelivered = "delivered"
	DeliveryFailed    = "failed"
	// The endpoint was switched off while this waited. Kept, not sent: turning
	// a receiver back on must not release a day's backlog into it.
	DeliverySkipped = "skipped"
)

// WebhookDelivery is one event on its way to one endpoint — the queue and,
// once sent, its record.
//
// ⚠️ **(endpointId, eventId) is unique.** The same event can be raised twice —
// a till resending a sale it could not confirm, two status changes read back
// in one go — and a receiver told "order 42 was delivered" twice has no way to
// know it was one delivery. The event id is derived from the order and its
// position in statusHistory, so a repeat is the same id and the insert fails.
type WebhookDelivery struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	EndpointID primitive.ObjectID `bson:"endpointId" json:"endpointId"`
	EventID    string             `bson:"eventId" json:"eventId"`
	Event      string             `bson:"event" json:"event"`
	// The body exactly as it will be signed and sent — a snapshot at the moment
	// of the event, so a retry two hours later says what was true then.
	Payload string `bson:"payload" json:"-"`
	// What the event is about, for the panel's list.
	OrderNumber string `bson:"orderNumber,omitempty" json:"orderNumber,omitempty"`

	Status        string     `bson:"status" json:"status"`
	Attempts      int        `bson:"attempts" json:"attempts"`
	NextAttemptAt time.Time  `bson:"nextAttemptAt" json:"nextAttemptAt"`
	LastStatus    int        `bson:"lastStatus,omitempty" json:"lastStatus,omitempty"`
	LastError     string     `bson:"lastError,omitempty" json:"lastError,omitempty"`
	CreatedAt     time.Time  `bson:"createdAt" json:"createdAt"`
	DeliveredAt   *time.Time `bson:"deliveredAt,omitempty" json:"deliveredAt,omitempty"`
}
