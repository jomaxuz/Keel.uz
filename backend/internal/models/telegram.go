package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TelegramSettings is the restaurant's own bot.
//
// ⚠️ **Its own collection, like the payment keys and the SMS password**, and for
// the same reason: the `restaurant` document is returned in full to every
// visitor, and one forgotten `json:"-"` there is a leaked bot token. A leaked bot
// token is not "somebody else's credit" — it is the ability to write to every
// guest who ever opened this restaurant's mini app, under the restaurant's name.
//
// Per restaurant, not per platform. The bot is the owner's: their name, their
// contract with Telegram, their brand on the mini app — the same reasoning that
// put the SMS gateway and the payment providers in the owner's hands.
type TelegramSettings struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	Enabled bool               `bson:"enabled" json:"enabled"`
	// The token from @BotFather. Never returned to a browser — the panel only
	// ever learns whether one is stored.
	BotToken string `bson:"botToken" json:"-"`
	// The bot's @username, filled in by the check button rather than typed.
	//
	// Public on purpose: the mini app's deep link is built from it
	// (`t.me/<username>/app`), so it has to reach the page. It also removes a
	// transcription error from the one place it would be invisible — a link with
	// a mistyped bot name opens somebody else's bot.
	BotUsername string `bson:"botUsername" json:"botUsername"`

	// What the last check said. Kept for the reason every other integration here
	// keeps it: a token that has stopped working says nothing on its own, and the
	// restaurant finds out when a guest cannot sign in.
	LastCheckAt time.Time `bson:"lastCheckAt" json:"lastCheckAt"`
	LastCheckOk bool      `bson:"lastCheckOk" json:"lastCheckOk"`
	LastCheck   string    `bson:"lastCheck" json:"lastCheck"`

	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// Usable reports whether a Telegram login can be completed at all.
//
// Both halves matter: a token with the switch off is a restaurant that connected
// a bot and then turned it off, and honouring the switch is what makes turning it
// off mean something.
func (t *TelegramSettings) Usable() bool {
	return t != nil && t.Enabled && t.BotToken != ""
}
