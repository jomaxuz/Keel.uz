package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- A phone that can be told something ----
//
// ⚠️ **A device belongs to a person, not to a branch.** A waiter's phone is
// theirs; the point of a notification is that it reaches the one person whose
// table is affected, and a branch-wide broadcast is how a restaurant teaches
// its staff to swipe notifications away without reading them.
//
// ⚠️ **Signing out drops it, and that is not tidiness.** Phones get handed over
// — a shift ends, somebody borrows one, a device is reassigned — and a token
// left behind sends the next evening's tables to the person who went home. The
// same reasoning that keeps the till session in memory rather than on disk.
type StaffDevice struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	StaffID primitive.ObjectID `bson:"staffId" json:"staffId"`
	// The branch is denormalised so a send can be scoped without a second read.
	// ⚠️ Refreshed on every registration: an employee moved between branches
	// keeps the same phone.
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	// The Expo push token ("ExponentPushToken[...]").
	//
	// ⚠️ **Unique.** The same phone re-registers on every launch — the token is
	// re-read from the operating system and can be re-issued after a reinstall
	// — and without the index one device accumulates a row per launch, so every
	// notification is delivered as many times as the app has been opened.
	Token string `bson:"token" json:"token"`
	// "ios" or "android". Kept for reading the logs, not for behaviour: Expo's
	// service takes the same call for both.
	Platform string `bson:"platform,omitempty" json:"platform,omitempty"`
	// ⚠️ **The language this phone reads, stored with the token.** A
	// notification is written by the server, so it is the one piece of text on
	// these apps the device cannot translate for itself — and the choice
	// belongs to the person holding the phone, not to the restaurant's panel.
	// Empty is Uzbek, which is what every row written before this said.
	Lang      string    `bson:"lang,omitempty" json:"lang,omitempty"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}
