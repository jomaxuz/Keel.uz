package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- An owner's phone ----
//
// ⚠️ **The fourth device table, and the reason is the same as the other
// three.** A panel account, an employee and a courier are three different
// identities in three different collections; one field holding two kinds of id
// is how a notification reaches the wrong person, and here the wrong person
// would be reading somebody else's takings.
//
// ⚠️ **What this channel carries did not exist on a phone before.** The loss
// alerts — a discount after the bill was printed, a till short at the close,
// a dish written off — went to Telegram and nowhere else, so a restaurant that
// never linked a chat was told none of them. That is the gap this fills, and it
// is why the alert path sends here as well rather than instead.
type AdminDevice struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	AdminID primitive.ObjectID `bson:"adminId" json:"adminId"`
	// The branch this account is tied to, copied so a send can be scoped
	// without a second read. Empty means the whole company, which is what an
	// owner has.
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	// "owner" or "manager", copied for the same reason: some events are for
	// owners only (see sendToOwners — a manager is one of the people the loss
	// alerts are *about*).
	Role string `bson:"role,omitempty" json:"role,omitempty"`
	// The Expo push token ("ExponentPushToken[...]").
	//
	// ⚠️ **Unique.** The app re-registers on every launch, and without the
	// index one phone accumulates a row per launch — so every notification is
	// delivered as many times as the app has been opened.
	Token    string `bson:"token" json:"token"`
	Platform string `bson:"platform,omitempty" json:"platform,omitempty"`
	// The language this phone reads. Notifications are written by the server,
	// so it is the one text in the app the device cannot translate itself.
	Lang      string    `bson:"lang,omitempty" json:"lang,omitempty"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}
