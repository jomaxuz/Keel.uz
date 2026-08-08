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
type Campaign struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Segment string             `bson:"segment" json:"segment"`
	Text    string             `bson:"text" json:"text"`
	Status  string             `bson:"status" json:"status"`

	// How many were messaged, and the two reasons some were not. Both kept
	// because the difference is actionable: "no phone" is a data problem, "opted
	// out" is a decision the guest made and nobody should try to fix.
	Total    int `bson:"total" json:"total"`
	OptedOut int `bson:"optedOut" json:"optedOut"`
	NoPhone  int `bson:"noPhone" json:"noPhone"`

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
