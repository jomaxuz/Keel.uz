package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Browser push: the third way a message reaches a customer.
//
// The other two already exist and neither covers this ground. SMS reaches
// everybody and costs money on every message; Telegram is free and reaches only
// the guests who opened the bot. Web push is free, and it is the only channel
// that works for somebody sitting at a desktop browser with the restaurant's
// site open in a tab — which, for an office lunch order, is most of them.

// PushSettings is the restaurant's VAPID identity: one document, generated
// once.
//
// ⚠️ **Generated, never configured.** Unlike the payment or SMS credentials
// there is nothing for an owner to sign up for and nothing to paste in — the
// keys are just a keypair, and asking a restaurant owner to produce one would
// mean the feature is never switched on anywhere. It is created on first use
// and then left alone.
//
// ⚠️ **And never rotated.** Every push service ties a subscription to the
// public key it was created under, so regenerating these silently invalidates
// every subscription the restaurant has ever collected. Nothing reports it: the
// notifications simply stop arriving, and the panel goes on saying they were
// sent.
type PushSettings struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	// Base64url, unpadded. The public key is handed to every visitor's browser
	// and is public by design — the same category as the map key
	// (§ "Xarita kaliti sir emas"), and the exact opposite of the field below.
	PublicKey string `bson:"publicKey" json:"publicKey"`
	// ⚠️ `json:"-"` is load-bearing. This document is small enough to be
	// returned by accident from a settings endpoint, and one forgotten tag is
	// one leaked signing key — which lets anybody send notifications in the
	// restaurant's name to every guest who ever subscribed.
	PrivateKey string `bson:"privateKey" json:"-"`
	// Contact address the push services require, so they have somebody to
	// write to about a misbehaving sender.
	Subject   string    `bson:"subject" json:"subject"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

// PushSubscription is one browser that agreed to be notified.
//
// One document per browser, not per person: the same customer on a phone and a
// laptop is two subscriptions, and both should ring. Deduplication happens at
// send time, by account, for the same reason the SMS audience deduplicates by
// phone — one message per person, however many devices they own.
type PushSubscription struct {
	ID     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID primitive.ObjectID `bson:"userId" json:"userId"`
	// ⚠️ **The unique key.** The endpoint URL is both the address and the
	// identity of a subscription — it is unguessable and issued once per
	// browser. Without a unique index every page load that re-subscribes (which
	// browsers do, silently, after an update) adds a duplicate, and the guest
	// starts receiving each campaign three times.
	Endpoint string `bson:"endpoint" json:"endpoint"`
	// The browser's public key and shared secret, base64url. Stored verbatim:
	// none of it is ours to interpret.
	P256dh string `bson:"p256dh" json:"-"`
	Auth   string `bson:"auth" json:"-"`
	// Which language to write to them in, frozen at subscribe time so a
	// background send does not have to read the account again.
	Lang string `bson:"lang,omitempty" json:"lang,omitempty"`
	// What the browser called itself. Only so the guest can tell two devices
	// apart when they revoke one.
	Device    string    `bson:"device,omitempty" json:"device,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	// Refreshed whenever the browser re-registers, so a subscription that has
	// not been seen in a year can be told from one that is simply quiet.
	SeenAt time.Time `bson:"seenAt" json:"seenAt"`
}
