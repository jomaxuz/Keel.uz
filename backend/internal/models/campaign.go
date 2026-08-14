package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Campaign statuses. Stored, so never renamed.
const (
	CampaignSending = "sending"
	CampaignDone    = "done"
	CampaignFailed  = "failed"
)

// Campaign is one message sent to one segment.
//
// A record rather than a counter, for the same reason a staff payment is: weeks
// later somebody asks "did we message the sleeping customers in July, and what
// did we say" — usually because a guest is annoyed, or because the owner wants
// to repeat something that worked. A tally answers neither.
//
// The audience itself is deliberately **not** stored. It is computed from the
// order history at send time (see handlers/campaigns.go), and a saved list would
// be wrong by the next order — the customer who ordered yesterday is no longer
// asleep. What is worth keeping is the shape of the send: which segment, how
// many, how many arrived.
// The two ways a campaign reaches somebody. Stored, so never renamed.
const (
	CampaignSMS      = "sms"
	CampaignTelegram = "telegram"
	// A browser notification, to the guests who allowed them on the site.
	//
	// ⚠️ A third audience, not a fallback for the other two. It is the only
	// channel that reaches somebody at a desktop with no phone in the
	// conversation, and it misses everybody who declined the permission — so
	// "we sent it by push" and "we sent it by SMS" reach overlapping but
	// different halves of the customer base, and the campaign record has to
	// say which.
	CampaignPush = "push"
)

type Campaign struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Segment string             `bson:"segment" json:"segment"`
	Text    string             `bson:"text" json:"text"`
	Status  string             `bson:"status" json:"status"`
	// "sms" or "telegram". ⚠️ Recorded rather than derived: the two reach different
	// people (a phone number versus a bot the guest opened), cost different money,
	// and a later "why did only 40 get it?" is answered by this field alone.
	Channel string `bson:"channel,omitempty" json:"channel,omitempty"`
	// An optional photograph, sent as the message's own image. Telegram only —
	// there is no such thing in an SMS.
	Image string `bson:"image,omitempty" json:"image,omitempty"`

	// How many were messaged, and the two reasons some were not. Both kept
	// because the difference is actionable: "no phone" is a data problem, "opted
	// out" is a decision the guest made and nobody should try to fix.
	Total    int `bson:"total" json:"total"`
	OptedOut int `bson:"optedOut" json:"optedOut"`
	NoPhone  int `bson:"noPhone" json:"noPhone"`
	// Excluded because they have never opened the bot. The Telegram counterpart of
	// NoPhone, and kept apart for the same reason: it is a different problem with a
	// different fix — invite them to the bot rather than collect a number.
	NoTelegram int `bson:"noTelegram,omitempty" json:"noTelegram,omitempty"`
	// Excluded because no browser of theirs is subscribed. The push counterpart of
	// NoPhone and NoTelegram, kept apart for the same reason: the fix is different
	// again — ask them on the site, where they already are.
	NoPush int `bson:"noPush,omitempty" json:"noPush,omitempty"`

	// Progress, written while the send runs.
	Sent   int `bson:"sent" json:"sent"`
	Failed int `bson:"failed" json:"failed"`
	// The first gateway error, verbatim. "84 failed" with no reason leaves the
	// owner guessing between no credit, an unapproved sender name and our bug —
	// three different next steps.
	Error string `bson:"error,omitempty" json:"error,omitempty"`

	// SMS parts per message, frozen at send time: the price depends on it, and
	// the alphabet rule that decides it is invisible on screen.
	Parts    int    `bson:"parts" json:"parts"`
	Provider string `bson:"provider" json:"provider"`

	CreatedBy  string     `bson:"createdBy" json:"createdBy"`
	CreatedAt  time.Time  `bson:"createdAt" json:"createdAt"`
	StartedAt  *time.Time `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	FinishedAt *time.Time `bson:"finishedAt,omitempty" json:"finishedAt,omitempty"`
}
