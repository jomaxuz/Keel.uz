package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PrintJob is one receipt waiting for one printer.
//
// ⚠️ **It is a queue because the printer is not ours to reach.** The server is
// in a data centre and the printer is on a shelf in a kitchen; the only thing
// that can talk to both is the agent on the restaurant's own PC, which asks for
// work. So printing is not a call, it is a job — and that is also what makes it
// survive the case a restaurant actually hits: the printer is out of paper at
// eight o'clock, somebody loads a roll, and the ticket is still there.
//
// ⚠️ **The bytes are built on the server.** The agent carries them one hop and
// writes them; it holds no layout, no code page and no business rule — the same
// division the fiscal agent is built on, for the same reason: nobody will ever
// read the log of a program running unattended in a restaurant.
type PrintJob struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId" json:"-"`

	// Which printer, and what it is being sent.
	PrinterID   string `bson:"printerId" json:"printerId"`
	PrinterName string `bson:"printerName,omitempty" json:"printerName,omitempty"`
	Target      string `bson:"target" json:"target"`
	// kitchen · till · customer · precheck — for the panel, and for the log.
	Kind string `bson:"kind" json:"kind"`
	// The order this belongs to, so a failure can be traced to a table.
	OrderID primitive.ObjectID `bson:"orderId,omitempty" json:"-"`
	Number  string             `bson:"number,omitempty" json:"number,omitempty"`

	// The ESC/POS bytes, ready to write.
	Payload []byte `bson:"payload" json:"-"`

	CreatedAt time.Time  `bson:"createdAt" json:"createdAt"`
	TakenAt   *time.Time `bson:"takenAt,omitempty" json:"takenAt,omitempty"`
	DoneAt    *time.Time `bson:"doneAt,omitempty" json:"doneAt,omitempty"`
	// What went wrong at the printer, in its own words.
	Error string `bson:"error,omitempty" json:"error,omitempty"`
	// How many times it has been handed out. ⚠️ Bounded: a job that kills the
	// agent every time it is tried would otherwise be handed out forever and
	// nothing else in the queue would ever print.
	Tries int `bson:"tries,omitempty" json:"tries,omitempty"`
}

// MaxPrintTries is how often a job is offered before it is given up on.
//
// ⚠️ Three, and then it is left failed **with its reason**: a kitchen ticket
// that cannot print is a plate nobody is making, and the useful thing is for
// somebody to be told, not for the queue to keep trying quietly.
const MaxPrintTries = 3
