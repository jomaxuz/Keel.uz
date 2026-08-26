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
	// ⚠️ **No `omitempty`, and that word cost every receipt this queue ever
	// held.** With it, a fresh job writes no `tries` field at all — and
	// MongoDB's `$lt` against a number does not match a missing field, so the
	// query that hands work to the agent could never see a new job. Proven
	// against a real database: `{tries: {$lt: 3}}` returns the document with
	// `tries: 1` and not the one without the field.
	//
	// The symptom was perfect: jobs queued, the relay ran and asked, the server
	// answered "nothing to do", nothing printed and nothing was logged.
	Tries int `bson:"tries" json:"tries,omitempty"`
}

// MaxPrintTries is how often a job is offered before it is given up on.
//
// ⚠️ Three, and then it is left failed **with its reason**: a kitchen ticket
// that cannot print is a plate nobody is making, and the useful thing is for
// somebody to be told, not for the queue to keep trying quietly.
const MaxPrintTries = 3

// MaxPrintAge is how long a queued job is still worth printing.
//
// ⚠️ **A ticket has a lifetime, and the queue had none.** A job waited for an
// agent forever — so a restaurant that printed all afternoon with the till
// switched off, then opened it in the evening, got the entire afternoon out of
// the printer in one burst. Reported exactly that way: "chek chiqmadi… keyin
// hamma chek bittada chiqdi".
//
// That is worse than the paper. A kitchen ticket for an order served five hours
// ago is not a late ticket, it is an **instruction to cook it again** — and it
// arrives at the pass looking exactly like a new one. A guest's receipt reaches
// a counter the guest left before lunch.
//
// ⚠️ **Thirty minutes, chosen between two failures.** Shorter and a printer
// briefly unplugged loses a real order the kitchen never sees. Longer and a
// service's worth of dead paper is still waiting. Half an hour survives a
// jam, a paper change, or somebody rebooting the monoblock, and does not
// survive a shift.
//
// ⚠️ **Applied when the job is handed out, not when it is created.** The
// difference matters: a job created while the till was off is perfectly valid
// for the next thirty minutes, and refusing to queue it would lose the ticket
// for a till that comes back in two.
const MaxPrintAge = 30 * time.Minute
