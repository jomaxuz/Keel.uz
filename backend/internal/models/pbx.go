package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Phone system ----
//
// One PBX account per company: the restaurant has one phone number that rings,
// whatever else it has two of. Its own collection rather than a field on the
// restaurant profile, for the reason every other credential here lives apart —
// the profile is returned in full to every visitor of the site.

// PBX providers.
const PBXOnlinePBX = "onlinepbx"

type PBXSettings struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	Provider string             `bson:"provider" json:"provider"`
	Enabled  bool               `bson:"enabled" json:"enabled"`

	// The account's own subdomain, e.g. "example.onpbx.ru".
	Domain string `bson:"domain" json:"domain"`
	APIKey string `bson:"apiKey" json:"-"`

	// The secret in the webhook URL.
	//
	// onlinePBX sends no credentials with its webhooks — it posts to whatever
	// address is configured in its panel and nothing more. So the address
	// itself has to be the secret, or anyone who guesses the path can invent
	// calls in the restaurant's log. Generated here, never typed by a human,
	// and rotatable.
	WebhookToken string `bson:"webhookToken" json:"-"`

	// Which extension the panel dials *from* when an operator has none of
	// their own. Empty means click-to-call is only offered to operators who
	// have an extension set on their account.
	DefaultExtension string `bson:"defaultExtension" json:"defaultExtension"`

	LastCheckAt *time.Time `bson:"lastCheckAt,omitempty" json:"lastCheckAt,omitempty"`
	LastCheckOK bool       `bson:"lastCheckOk" json:"lastCheckOk"`
	LastCheck   string     `bson:"lastCheck" json:"lastCheck"`
	// When an event last arrived. The single most useful line on the settings
	// page: credentials can be perfect and the webhook URL still never pasted
	// into onlinePBX, and nothing else distinguishes those two states.
	LastEventAt *time.Time `bson:"lastEventAt,omitempty" json:"lastEventAt,omitempty"`

	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}
