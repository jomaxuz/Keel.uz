package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- A guest's phone ----
//
// ⚠️ **Its own collection, and this is the fifth of them.** `staff_device`,
// `courier_device`, `admin_device`, `login_device` and now this: the ids come
// from different collections, and one field holding two kinds of id is how a
// notification about somebody's dinner reaches a cook. The reasoning is written
// out at length in models/admindevice.go and it has not changed.
//
// ⚠️ **A guest is not staff, and the difference decides the content.** An
// employee's phone is told about work — a table, a shift, a wage — and can be
// told a lot of it, because they are being paid to read it. A guest is told one
// thing: what happened to the food they are waiting for. Anything else is an
// app they turn notifications off for, and the switch is one switch: the
// campaign we wanted to send next month goes with it.
type UserDevice struct {
	ID     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID primitive.ObjectID `bson:"userId" json:"userId"`

	// The FCM registration token this phone holds.
	//
	// ⚠️ **Unique.** The app re-registers on every launch — the token is re-read
	// from the operating system and can be re-issued after a reinstall — and
	// without the index one phone accumulates a row per launch, so every order
	// update is delivered as many times as the app has been opened. The web
	// subscription learned this first (`push_subscription.endpoint`), and it is
	// the same failure with a different key.
	Token string `bson:"token" json:"token"`

	// ⚠️ **The language this phone reads, stored with the token.** The text is
	// written by the server (`internal/i18n`), so it is the one thing on this
	// device that cannot be translated where it is displayed — and the choice
	// belongs to the guest holding the phone, not to the restaurant's panel.
	// Empty is Uzbek, which is what every row written before this said.
	Lang string `bson:"lang,omitempty" json:"lang,omitempty"`

	// "android". Kept for reading the logs rather than for behaviour — the
	// transport is chosen by the token's own shape (push/send.go).
	Platform  string    `bson:"platform,omitempty" json:"platform,omitempty"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}
