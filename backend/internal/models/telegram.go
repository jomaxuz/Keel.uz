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

	// The secret half of the webhook address, generated rather than typed.
	//
	// ⚠️ Telegram sends no password of its own with a delivery, so the URL is the
	// credential — the same shape as the onlinePBX webhook. Kept out of every
	// response (`json:"-"`) and rotatable: a leaked address is replaced by
	// generating a new one and re-registering.
	WebhookToken string `bson:"webhookToken" json:"-"`
	// Registered with Telegram at this moment.
	WebhookAt time.Time `bson:"webhookAt" json:"webhookAt"`
	// ⚠️ **Which registration this is**, and it exists because of a real failure.
	//
	// `setWebhook` does not only say *where* to deliver: it says *what* to
	// deliver (`allowed_updates`). The first version asked for messages only.
	// Adding the language buttons made the bot depend on `callback_query` — an
	// update type Telegram had never been asked for, and therefore never sent.
	//
	// The result was the worst shape a bug can take: the greeting arrived, the
	// buttons were drawn, and tapping one did **nothing at all**. Nothing failed,
	// nothing was logged, and the settings page showed a healthy bot — because
	// from our side everything *was* healthy. The stale thing was a registration
	// held by Telegram, which no amount of looking at our own state can reveal.
	//
	// So the registration carries a version, and a stored version behind the code
	// is re-registered automatically (see registerTelegramWebhook). A field
	// somebody has to remember to refresh is a field that will be stale again the
	// next time the shape changes.
	WebhookVersion int `bson:"webhookVersion,omitempty" json:"webhookVersion,omitempty"`
	// ⚠️ The most useful line on the settings page, for the same reason
	// `lastEventAt` is for onlinePBX: the token can be perfect and the bot still
	// silent, and a connection check **cannot show that** — it proves we can
	// reach Telegram, not that Telegram can reach us. This is the only field that
	// answers "did a guest's Start actually arrive here?".
	LastUpdateAt time.Time `bson:"lastUpdateAt" json:"lastUpdateAt"`

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
