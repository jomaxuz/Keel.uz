package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Which phone an account is allowed to be on ----
//
// ⚠️ **A password is not an identity in a restaurant.** Four people share a
// staff room, a courier's login is written on a card in the office, and a
// waiter who wants an evening off can hand their username to a colleague — none
// of which the server can see, because every one of those logins is correct.
// What it can see is that the phone changed.
//
// So a sign-in from an app binds the account to that install, and the binding
// is refused two ways:
//
//	one account, one phone   — signing in on a second device is refused
//	one phone, one account   — a second account on the same install is refused
//
// ⚠️ **Per app, not globally.** The same person is legitimately a waiter on
// Keel Waiter and an employee on Keel Team, and an owner may hold Owner and
// Waiter on one handset. Binding across apps would refuse the ordinary case.
//
// ⚠️ **The escape hatch is the whole feature, not a concession to it.** A
// reinstall mints a new id, a lost phone never comes back, and somebody's
// screen breaks on a Friday night — so every binding is visible in the panel
// with a button that deletes it. A lock nobody can lift is worse than the
// sharing it prevents: this codebase has written that down once already, about
// the check hold that expires by itself.
type LoginDevice struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	// Which table `SubjectID` points into: "admin", "staff" or "courier".
	//
	// ⚠️ **Three id spaces, and the kind is what keeps them apart.** A courier
	// id and a staff id can collide in the way any two ObjectIDs can — never in
	// practice, but the field costs nothing and the alternative is three
	// collections of an identical shape.
	Kind      string             `bson:"kind" json:"kind"`
	SubjectID primitive.ObjectID `bson:"subjectId" json:"subjectId"`
	// Which app: "owner", "waiter", "courier", "team".
	App string `bson:"app" json:"app"`
	// The install's own id, minted by the app on first run and kept in the
	// device's secure store.
	//
	// ⚠️ **Not a hardware serial**, and that is deliberate as well as
	// unavoidable: Android has not handed out a stable device id to ordinary
	// apps for years, and asking for one would be asking for an identifier we
	// have no business keeping. What this identifies is an *installation* —
	// which is exactly the thing being bound.
	DeviceID string `bson:"deviceId" json:"deviceId"`
	Platform string `bson:"platform,omitempty" json:"platform,omitempty"`
	// A name somebody can recognise in the panel ("SM-A155F"). Sent by the app;
	// empty is ordinary and the screen falls back to the id.
	Name string `bson:"name,omitempty" json:"name,omitempty"`
	// Where it was last seen from. ⚠️ Kept because it is the one thing that
	// answers "is this really them" when a device *is* legitimately shared: a
	// courier logging in from the restaurant's wifi every morning and then from
	// another city is a question worth asking.
	IP         string    `bson:"ip,omitempty" json:"ip,omitempty"`
	CreatedAt  time.Time `bson:"createdAt" json:"createdAt"`
	LastSeenAt time.Time `bson:"lastSeenAt" json:"lastSeenAt"`
}
