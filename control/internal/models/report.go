package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Crash reports: what broke, before the restaurant tells us ----
//
// ⚠️ **Grouped, never a stream.** One render loop in one panel is ten thousand
// identical errors in an afternoon, and a list with ten thousand rows in it
// answers no question at all. What somebody fixing this needs is *what is
// broken*, with how often and where attached — so identical errors collapse
// into one group and the group carries the count.
//
// ⚠️ **The tenant is taken from the link token, never from the report.** The
// panel posts to its own server, which forwards with the per-tenant credential
// it already holds for the briefing and the domain link. A slug in the body
// would let any browser file a crash against another restaurant, and this list
// is read as evidence.
//
// ⚠️ **This does not replace the down check.** A report can only arrive if the
// tenant's own container is up to forward it, so the one failure it cannot
// report is that container being gone. That is what `attention: "down"` reads
// live from Docker for, and the two are deliberately different mechanisms: a
// pipe cannot carry news of its own absence.

// Which program the report came from. Free-form on the wire — a native app we
// have not written yet must be able to report before this file knows its name —
// but these are the ones that exist.
const (
	ReportAppPanel   = "panel"   // the restaurant dashboard, in a browser
	ReportAppSite    = "site"    // the public restaurant site, a guest's browser
	ReportAppTill    = "till"    // the till and floor screens
	ReportAppWaiter  = "waiter"  // the waiter's phone
	ReportAppKitchen = "kitchen" // the kitchen screen
	ReportAppCourier = "courier" // the courier PWA
	ReportAppServer  = "server"  // the tenant's own Go process
)

// ErrorGroup is one distinct fault, however many times it happened.
type ErrorGroup struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`

	// Which restaurant. Denormalised like the support thread's, and for the
	// same reason: the list is drawn and searched without joining, and a group
	// still reads correctly after a tenant is renamed or deleted.
	Slug       string `bson:"slug" json:"slug"`
	Restaurant string `bson:"restaurant" json:"restaurant"`

	// Which program, and which build of it. ⚠️ The version is on the group
	// rather than only on the samples because "is this still happening after
	// the deploy" is the first question asked of every one of these, and a
	// version buried in a sample is a version nobody looks at.
	App            string `bson:"app" json:"app"`
	FirstVersion   string `bson:"firstVersion,omitempty" json:"firstVersion,omitempty"`
	LatestVersion  string `bson:"latestVersion,omitempty" json:"latestVersion,omitempty"`
	Platform       string `bson:"platform,omitempty" json:"platform,omitempty"`
	LatestPlatform string `bson:"latestPlatform,omitempty" json:"latestPlatform,omitempty"`

	// What identical means. See Fingerprint.
	Key string `bson:"key" json:"key"`

	// The fault, in the words the program used.
	Message string `bson:"message" json:"message"`
	// Where it happened — a route, a screen name, an endpoint.
	Where string `bson:"where,omitempty" json:"where,omitempty"`

	Count int `bson:"count" json:"count"`
	// ⚠️ **Counted per day as well as in total**, because "forty times" means
	// two entirely different things over an afternoon and over three months,
	// and the list is sorted by what is happening *now*.
	Today   int    `bson:"today" json:"today"`
	TodayOn string `bson:"todayOn,omitempty" json:"todayOn,omitempty"`

	// ⚠️ How many people, not just how many times. One cashier with a broken
	// tablet and every cashier in the chain are the same count and completely
	// different mornings.
	Sessions []string `bson:"sessions,omitempty" json:"-"`
	Users    int      `bson:"users" json:"users"`

	FirstAt time.Time `bson:"firstAt" json:"firstAt"`
	LastAt  time.Time `bson:"lastAt" json:"lastAt"`

	// A handful of the actual occurrences, newest last. Capped — see
	// MaxReportSamples. The stack is here rather than on the group because two
	// occurrences of one fault can have different stacks, and a single stored
	// stack quietly becomes the stack of whichever one arrived first.
	Samples []ErrorSample `bson:"samples,omitempty" json:"samples,omitempty"`

	// Set when somebody has dealt with it. ⚠️ Resolving never deletes and never
	// stops collection: the group keeps counting, and a resolved group that
	// starts counting again is the single most useful row on this screen —
	// it is a fix that did not hold.
	Resolved   bool       `bson:"resolved" json:"resolved"`
	ResolvedAt *time.Time `bson:"resolvedAt,omitempty" json:"resolvedAt,omitempty"`
	ResolvedBy string     `bson:"resolvedBy,omitempty" json:"resolvedBy,omitempty"`
	// What was done. One line, for the next person who sees it come back.
	Note string `bson:"note,omitempty" json:"note,omitempty"`
	// The count when it was resolved, so a regression is arithmetic rather
	// than memory.
	ResolvedCount int `bson:"resolvedCount,omitempty" json:"resolvedCount,omitempty"`
}

// ErrorSample is one occurrence.
type ErrorSample struct {
	At    time.Time `bson:"at" json:"at"`
	Stack string    `bson:"stack,omitempty" json:"stack,omitempty"`
	// What the program was doing. Deliberately a short free-form line rather
	// than a structure: every app has different things worth saying, and a
	// schema would mean the useful one is the field nobody added.
	Context string `bson:"context,omitempty" json:"context,omitempty"`
	// Version and platform of this particular occurrence.
	Version  string `bson:"version,omitempty" json:"version,omitempty"`
	Platform string `bson:"platform,omitempty" json:"platform,omitempty"`
	// Which branch and which role, where the app knew. ⚠️ A role, never a name
	// and never an id: "kassir" is what makes a report reproducible, and the
	// cashier's identity is not ours to collect from somebody else's staff.
	Branch string `bson:"branch,omitempty" json:"branch,omitempty"`
	Role   string `bson:"role,omitempty" json:"role,omitempty"`
}

// MaxReportSamples is how many occurrences of one fault are kept.
//
// ⚠️ Small on purpose. The fifth stack of the same crash has never told anybody
// anything the first told them, and the storage is the platform's — one broken
// loop at one restaurant must not be able to grow without bound.
const MaxReportSamples = 5
