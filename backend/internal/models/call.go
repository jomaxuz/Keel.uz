package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Call centre ----
//
// One record per conversation an operator had on the phone.
//
// The restaurant has no PBX here: nothing observes the line, so a call exists
// only because the operator wrote it down. That shapes the model — every field
// has to be worth typing during a live call, or it will be left empty and the
// log becomes a lie. What is worth typing: who rang, what came of it, and
// whether somebody still has to ring back.
//
// The outcome is the point of the record. "How many calls did we take today"
// is a number nobody acts on; "how many turned into orders, and who is still
// waiting for a call back" is two numbers a manager runs the shift from.

// Call outcomes. Stored as ids and translated in the panel, the same rule the
// audit log follows: renaming a label must never rewrite history.
const (
	// The call produced an order — the one outcome the shift is measured by.
	CallOutcomeOrder = "order"
	// A table was held.
	CallOutcomeBooking = "booking"
	// A question answered: opening hours, where an order has got to, the price
	// of something. No money, but the call was worth taking.
	CallOutcomeInfo = "info"
	// Something went wrong and the guest said so. Kept apart from `info`
	// because a complaint is owed an answer, not just a reply.
	CallOutcomeComplaint = "complaint"
	// The guest wants ringing back, or the operator promised to. The only
	// outcome that leaves work behind, which is why CallbackAt goes with it.
	CallOutcomeCallback = "callback"
	// They asked, they heard the price or the wait, they hung up. Worth
	// recording: a run of these is the restaurant losing orders it could see.
	CallOutcomeRefused = "refused"
	// Nobody picked up / the line dropped. Usually paired with a callback.
	CallOutcomeMissed = "missed"
	CallOutcomeSpam   = "spam"
)

// CallOutcomes is every valid outcome, used to validate what the panel sends.
var CallOutcomes = []string{
	CallOutcomeOrder, CallOutcomeBooking, CallOutcomeInfo, CallOutcomeComplaint,
	CallOutcomeCallback, CallOutcomeRefused, CallOutcomeMissed, CallOutcomeSpam,
}

// CallOutcomePending is a call the PBX told us about that nobody has written
// up yet. It is not a failure state — it is the normal state of a call for the
// minute between the phone going down and the operator choosing an outcome —
// but it is the one worth filtering for at the end of a shift.
const CallOutcomePending = ""

// Call is one logged conversation.
type Call struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`

	// "in" (the guest rang) or "out" (the operator rang them).
	Direction string `bson:"direction" json:"direction"`
	// The number as dialled, normalised to 998XXXXXXXXX where possible.
	Phone string `bson:"phone" json:"phone"`
	// The customer this number belongs to, when it belongs to one. A first-time
	// caller has no account yet, so this is empty and the name is whatever the
	// operator heard.
	UserID primitive.ObjectID `bson:"userId,omitempty" json:"userId,omitempty"`
	// Copied at the time of the call: the account may be renamed or deleted,
	// the log still has to read.
	CustomerName string `bson:"customerName" json:"customerName"`

	Outcome string `bson:"outcome" json:"outcome"`
	Note    string `bson:"note" json:"note"`

	// What the call produced, so the log links to it. Numbers are denormalised
	// for the same reason names are — the row must be readable on its own.
	OrderID           primitive.ObjectID `bson:"orderId,omitempty" json:"orderId,omitempty"`
	OrderNumber       string             `bson:"orderNumber,omitempty" json:"orderNumber,omitempty"`
	ReservationID     primitive.ObjectID `bson:"reservationId,omitempty" json:"reservationId,omitempty"`
	ReservationNumber string             `bson:"reservationNumber,omitempty" json:"reservationNumber,omitempty"`

	// The promise: ring this person back at this time. Stored as a pointer so
	// "no callback" stays absent from the JSON rather than arriving as the
	// year 1 — which would sort to the top of every overdue list.
	CallbackAt   *time.Time `bson:"callbackAt,omitempty" json:"callbackAt,omitempty"`
	CallbackDone bool       `bson:"callbackDone" json:"callbackDone"`

	// Who took it. Name denormalised, like the audit log.
	OperatorID   primitive.ObjectID `bson:"operatorId,omitempty" json:"operatorId,omitempty"`
	OperatorName string             `bson:"operatorName" json:"operatorName"`

	// How long the operator was on the line. From the PBX when there is one,
	// and otherwise counted by the panel from the moment the card was opened;
	// 0 means neither happened.
	Seconds int `bson:"seconds" json:"seconds"`

	// ---- What the phone system said, when there is one ----
	//
	// Where this row came from: "manual" (an operator typed it) or "pbx" (the
	// exchange announced it). Empty on rows written before a PBX existed,
	// which reads as manual — which is what they were.
	Source string `bson:"source,omitempty" json:"source,omitempty"`
	// The exchange's own id for the call. Unique, and the thing that makes the
	// webhooks idempotent: five events arrive per call and all five have to
	// land on one row.
	PBXCallID string `bson:"pbxCallId,omitempty" json:"pbxCallId,omitempty"`
	// Which internal extension took it. Kept as well as OperatorID because a
	// call may be answered by somebody with no panel account at all.
	Extension string `bson:"extension,omitempty" json:"extension,omitempty"`
	// Whether a recording exists. The link itself is **not** stored: onlinePBX
	// signs its download URLs and a saved one quietly stops working, so the
	// panel asks for a fresh one when somebody presses play.
	HasRecording bool `bson:"hasRecording,omitempty" json:"hasRecording,omitempty"`
	// Why the line dropped, in the exchange's words. Worth keeping: it is how
	// "the customer hung up" is told apart from "our line died".
	HangupCause string `bson:"hangupCause,omitempty" json:"hangupCause,omitempty"`
	// Set while the call is live, cleared when it ends. This is what the
	// operator's screen watches to open the caller's card by itself.
	Ringing bool `bson:"ringing,omitempty" json:"ringing,omitempty"`
	// When the phone started ringing, and when it was answered.
	RingingAt  *time.Time `bson:"ringingAt,omitempty" json:"ringingAt,omitempty"`
	AnsweredAt *time.Time `bson:"answeredAt,omitempty" json:"answeredAt,omitempty"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}
