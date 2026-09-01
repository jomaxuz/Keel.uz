package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- A courier's phone, and why it is not a StaffDevice ----
//
// ⚠️ **Three roles, three sessions, three device tables.** A courier is not an
// employee of the till: they sign in with `courier_token`, their id comes from
// the `courier` collection, and nothing in the staff tables refers to them.
// Reusing `staff_device` would mean a `staffId` holding a courier's id — an
// identifier that is valid in one collection and points at a different person
// in another, which is the kind of mistake that stays hidden until two
// notifications go to the wrong phone.
//
// ⚠️ **Signing out drops the row.** A courier's phone is the one most likely to
// be sold, lost or handed to the next rider, and a token left behind sends
// tomorrow's addresses — customer names, phone numbers and all — to whoever now
// holds it, with no way for them to turn it off from their side.
type CourierDevice struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CourierID primitive.ObjectID `bson:"courierId" json:"courierId"`
	// The Expo push token ("ExponentPushToken[...]").
	//
	// ⚠️ **Unique.** The app re-registers on every launch — the token is
	// re-read from the operating system and can be re-issued after a reinstall
	// — and without the index one phone accumulates a row per launch, so every
	// notification arrives as many times as the app has been opened.
	Token string `bson:"token" json:"token"`
	// "ios" or "android". Kept for reading the logs, not for behaviour: Expo's
	// service takes the same call for both.
	Platform string `bson:"platform,omitempty" json:"platform,omitempty"`
	// ⚠️ The courier's chosen language, stored with the phone rather than with
	// the account. A notification is written by the server, so it is the only
	// text in this app the device cannot translate for itself — and the choice
	// belongs to the person holding the phone, who may ride for a restaurant
	// whose panel is in another language.
	Lang      string    `bson:"lang,omitempty" json:"lang,omitempty"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}
