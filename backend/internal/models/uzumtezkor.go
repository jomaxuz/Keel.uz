package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Uzum Tezkor: a marketplace that sends us its orders ----
//
// Uzum Tezkor's partner API works the other way round from every payment
// provider here: **they call us**. They fetch the menu and the stock, POST each
// order to this server, and ask for its status once a minute. We are the
// server; they are the client. See docs/vendor/uzum-tezkor-retail.md.
//
// ⚠️ **Their courier, always.** Uzum Tezkor delivers only with its own fleet, so
// an order from it has no address, no delivery fee and no courier of ours — the
// restaurant cooks it and hands it over. That is why it is its own order type
// rather than a "delivery" with a flag: every screen that reasons about our
// couriers, our zones or money a courier brings back must leave it alone.

// OrderTypeUzumTezkor is `order.type` for a Uzum Tezkor order. Same value as
// the payment method, on purpose: it is one marketplace, and a report grouping
// by either field must find the same orders.
const OrderTypeUzumTezkor = ProviderUzumTezkor

// OrderAggregator is what a marketplace told us about an order it placed.
type OrderAggregator struct {
	// "uzum_tezkor".
	Provider string `bson:"provider" json:"provider"`
	// The marketplace's own id (`eatsId`, `DDDDDD-DDDDDD`). ⚠️ **The retry key**:
	// Uzum resends an order when our answer did not arrive, and the resend must
	// return the first order rather than create a second dinner. Unique with
	// Provider — see repository/migrate.go.
	ExternalID string `bson:"externalId" json:"externalId"`
	// When their courier is due at the counter. The one delivery fact the
	// kitchen needs, because it decides when the food has to be ready.
	CourierAt    *time.Time `bson:"courierAt,omitempty" json:"courierAt,omitempty"`
	CourierPhone string     `bson:"courierPhone,omitempty" json:"courierPhone,omitempty"`
	// How many sets of cutlery.
	Persons int    `bson:"persons,omitempty" json:"persons,omitempty"`
	Comment string `bson:"comment,omitempty" json:"comment,omitempty"`
	// How the guest paid Uzum ("CARD" | "CASH") and what Uzum says the food
	// costs. ⚠️ **Neither is money the restaurant collects** — the guest paid the
	// marketplace, which settles later (see payout). Kept as told, never used to
	// price the order.
	PaymentType string `bson:"paymentType,omitempty" json:"paymentType,omitempty"`
	ItemsCost   int    `bson:"itemsCost,omitempty" json:"itemsCost,omitempty"`
	// The order exactly as it arrived.
	//
	// ⚠️ **What `GET /order/{id}` answers with, byte for byte.** Uzum reads a
	// different composition in that answer as the restaurant *changing* the
	// order — a push and a transaction on the guest's phone. Rebuilding the body
	// from our own lines would differ in a rounding, an omitted field or a
	// modifier name, and every one of those would be an update nobody made.
	// Never sent to a browser.
	Payload string `bson:"payload,omitempty" json:"-"`
}

// UzumTezkorSettings is the credential pair we hand Uzum Tezkor, and whether
// the integration is on. A singleton.
//
// ⚠️ **Its own collection**, for the reason payment_settings is: the restaurant
// profile goes to every visitor in full, and a forgotten `json:"-"` there would
// be a published key.
type UzumTezkorSettings struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	Enabled  bool               `bson:"enabled" json:"enabled"`
	ClientID string             `bson:"clientId,omitempty" json:"clientId"`
	// SHA-256 of the secret, never the secret. ⚠️ Shown once when it is minted
	// and then unrecoverable — a secret the panel can show again is a secret
	// anybody with the owner's laptop can copy.
	SecretHash string `bson:"secretHash,omitempty" json:"-"`
	// Bumped on every new secret, and carried in each access token: a rotated
	// secret has to end the tokens issued under the old one, not wait an hour
	// for them to expire.
	TokenVersion int        `bson:"tokenVersion" json:"-"`
	RotatedAt    *time.Time `bson:"rotatedAt,omitempty" json:"rotatedAt,omitempty"`
	UpdatedAt    time.Time  `bson:"updatedAt" json:"updatedAt"`
}
