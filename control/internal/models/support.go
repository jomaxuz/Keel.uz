package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Support: the restaurant asking us something ----
//
// ⚠️ **The conversation lives here, in the control plane, not in the tenant's
// own database.** An operator answers thirty restaurants in a morning; a thread
// stored on each customer's server would mean the console reaching into thirty
// databases to draw one list, and a restaurant whose container is down — which
// is exactly when they write to us — would be a restaurant that cannot ask for
// help.
//
// The restaurant's side of the wire is its container, which already holds a
// per-tenant token for the briefing and the domain link. The panel never talks
// to this service directly: it talks to its own server, which forwards. That is
// the same trust boundary everything else here uses, and it means a browser on
// a customer's domain never carries a platform credential.

// SupportStatus is where a thread stands. ⚠️ Three states, and the middle one
// is the whole point: "waiting" is the queue an operator works, "open" is a
// conversation somebody is already in.
type SupportStatus string

const (
	// The restaurant has written and nobody has answered yet.
	SupportWaiting SupportStatus = "waiting"
	// An operator has replied; the ball may be on either side.
	SupportOpen SupportStatus = "open"
	// Done. ⚠️ Closing never deletes: the next question from the same
	// restaurant is usually about the last answer.
	SupportClosed SupportStatus = "closed"
)

// SupportThread is one question and everything said about it.
type SupportThread struct {
	ID   primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Slug string             `bson:"slug" json:"slug"`
	// Denormalised so the queue can be drawn and searched without joining
	// thirty tenant records — and so a thread still reads correctly after a
	// restaurant is renamed or deleted.
	Restaurant string `bson:"restaurant" json:"restaurant"`

	// What the owner typed first, trimmed. ⚠️ The subject is never asked for
	// separately: a support widget that opens with a "subject" field is a
	// widget people close.
	Subject string        `bson:"subject" json:"subject"`
	Status  SupportStatus `bson:"status" json:"status"`

	// Who asked, on the restaurant's side. A name and a role, never a
	// credential — the operator needs to know whether they are talking to the
	// owner or to a cashier, and nothing more.
	AskedBy   string `bson:"askedBy" json:"askedBy"`
	AskedRole string `bson:"askedRole,omitempty" json:"askedRole,omitempty"`

	// Which operator picked it up, by console user id and name.
	OperatorID   primitive.ObjectID `bson:"operatorId,omitempty" json:"-"`
	OperatorName string             `bson:"operatorName,omitempty" json:"operatorName,omitempty"`

	// ⚠️ **Counted per side rather than a single "unread" flag.** One flag
	// cannot answer both "does this restaurant have an answer waiting" and
	// "does this queue have work in it", and the two are read by different
	// people on different screens.
	UnreadForUs    int `bson:"unreadForUs" json:"unreadForUs"`
	UnreadForOwner int `bson:"unreadForOwner" json:"unreadForOwner"`

	// The last line, for the list. Saves loading every thread's messages to
	// draw a queue.
	LastText string    `bson:"lastText" json:"lastText"`
	LastFrom string    `bson:"lastFrom" json:"lastFrom"`
	LastAt   time.Time `bson:"lastAt" json:"lastAt"`

	CreatedAt time.Time  `bson:"createdAt" json:"createdAt"`
	ClosedAt  *time.Time `bson:"closedAt,omitempty" json:"closedAt,omitempty"`
}

// SupportFrom is who said a line.
type SupportFrom string

const (
	FromOwner    SupportFrom = "owner"
	FromOperator SupportFrom = "operator"
	// ⚠️ **The assistant's answers are stored as messages too**, and marked.
	// An owner scrolling back has to be able to tell what a person told them
	// from what a model did — and an operator picking the thread up needs to
	// read what was already said before they repeat it.
	FromAssistant SupportFrom = "assistant"
)

// SupportMessage is one line of one thread.
type SupportMessage struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ThreadID primitive.ObjectID `bson:"threadId" json:"threadId"`
	From     SupportFrom        `bson:"from" json:"from"`
	Author   string             `bson:"author" json:"author"`
	Text     string             `bson:"text" json:"text"`
	At       time.Time          `bson:"at" json:"at"`
}

// SupportMaxText is how long one message may be.
//
// ⚠️ Generous, because the useful support message is a paste of what went
// wrong. Capped anyway: this arrives from a browser on a customer's domain, and
// an unbounded field is a way to fill our disk with somebody else's problem.
const SupportMaxText = 8000
