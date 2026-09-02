package models

import (
	"slices"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- The screens on the wall ----
//
// A television in the dining room, running our Android TV app: it plays what
// the panel tells it to and, in a fast-food room, shows which order numbers are
// cooking and which are ready.
//
// ⚠️ **A row per screen, like a till device and for the same reasons.** The
// module is sold per screen, so "how many televisions does this restaurant
// run" has to be a fact rather than a sentence in a price list; and a screen
// that walks out of the building has to be revocable **on its own**, without
// darkening every other television in the chain.

// TVScreen is one paired television.
type TVScreen struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId" json:"branchId"`

	// What to call it in the panel: "Zal TV", "Kassa oldidagi ekran".
	//
	// ⚠️ Free text and never read as anything but a label — the same rule as
	// Staff.Position. Its only job is to let a manager pick the right row when
	// one television has to be unpaired, and a list of identical rows is a list
	// nobody can act on.
	Name string `bson:"name" json:"name"`

	// The branch's TV revocation counter at the moment this was paired. A token
	// whose version is behind the branch's is dead — that is how "unpair every
	// screen in this branch" works in one act.
	Version int `bson:"version" json:"version"`

	// What this screen is for.
	//
	// ⚠️ **A property of the screen, not of the playlist.** One television by
	// the counter shows order numbers all day while the one in the dining room
	// plays the menu; the same restaurant, the same playlist library, two
	// answers — and the answer belongs to the wall the screen is hanging on.
	Mode string `bson:"mode" json:"mode"`

	// The device's own random id, generated on first launch and kept on the
	// television. ⚠️ Stored so a re-pair of the *same* set replaces its row
	// rather than adding a second one — otherwise a television reset by a
	// curious guest quietly eats another paid slot.
	InstallID string `bson:"installId,omitempty" json:"-"`

	// ⚠️ **When it last spoke to us**, which is the only honest answer to "is
	// this screen working?". A paired row and a television playing all evening
	// look identical otherwise — and the difference is the whole reason a
	// manager opens this page.
	LastSeenAt *time.Time `bson:"lastSeenAt,omitempty" json:"lastSeenAt,omitempty"`
	// What the app reports about itself, so "it shows nothing" can be answered
	// without driving to the restaurant.
	AppVersion string `bson:"appVersion,omitempty" json:"appVersion,omitempty"`

	// Who paired it, so a screen nobody recognises has somebody to ask.
	PairedBy  string    `bson:"pairedBy,omitempty" json:"pairedBy,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

// What a screen may be showing. Deliberately three, and the third exists
// because a fast-food room wants both at once: a video with a strip of numbers
// under it, rather than a television that has to choose.
const (
	TVModeContent = "content" // playlist only
	TVModeBoard   = "board"   // order numbers only
	TVModeSplit   = "split"   // playlist, with the numbers along the bottom
)

// TVModes is every valid mode, in the order the panel offers them.
var TVModes = []string{TVModeContent, TVModeBoard, TVModeSplit}

// ValidTVMode reports whether this is a mode we understand.
//
// ⚠️ An unknown mode is **not** silently corrected to content: a screen set to
// a word the server does not know is a bug somewhere else, and quietly showing
// the wrong thing on a wall is how it stays hidden.
func ValidTVMode(mode string) bool {
	return slices.Contains(TVModes, mode)
}

// TVPairing is a code shown on a television, waiting for somebody to type it
// into the panel.
//
// ⚠️ **Stored, unlike the kiosk's code, and it has to be.** The kiosk derives
// its code from the branch's secret because the screen already knows which
// branch it belongs to. A television being paired knows nothing about anybody —
// that is the whole point of pairing — so the code cannot be derived from
// anything, and the server has to remember what it handed out.
//
// ⚠️ **One live code per television.** The app asks for a fresh code every few
// seconds and the previous one dies at that moment, so a code photographed off
// a screen is worthless before the photographer has put their phone away. The
// TTL index is the second half of that: an abandoned pairing removes itself.
type TVPairing struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"-"`

	// What the television draws, and what somebody types into the panel.
	// Uppercase, and deliberately without the characters people misread: no
	// O/0, no I/1. A code read aloud across a dining room has to survive it.
	Code string `bson:"code" json:"code"`

	// The television asking. One pairing row per install: asking for a new code
	// replaces the old row rather than leaving a trail of live codes behind.
	InstallID string `bson:"installId" json:"-"`

	// The secret the television polls with. ⚠️ **Not the code**: the code is on
	// a screen in a public room, so anybody who can read it could otherwise ask
	// "has this been claimed yet?" and collect the token meant for the wall.
	PollSecret string `bson:"pollSecret" json:"-"`

	// Filled in when an owner or a manager claims the code. Until then the
	// television polls and gets "still waiting".
	ScreenID primitive.ObjectID `bson:"screenId,omitempty" json:"-"`
	Token    string             `bson:"token,omitempty" json:"-"`

	// ⚠️ A TTL index removes the row when this passes — see EnsureIndexes. An
	// expired code that lingers is a code that still works.
	ExpiresAt time.Time `bson:"expiresAt" json:"-"`
	CreatedAt time.Time `bson:"createdAt" json:"-"`
}
