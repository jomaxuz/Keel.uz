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

	// Where in the building this set hangs: "zal", "peshtaxta", "terrasa".
	//
	// ⚠️ **One zone per screen, and that asymmetry is deliberate** — a
	// television hangs on one wall. A slide, by contrast, targets several
	// (TVSlide.Zones), because "this promo goes to the dining room and the
	// terrace" is a real sentence and "this screen is in two rooms" is not.
	//
	// Empty means the screen has not been placed, and an unplaced screen plays
	// only what is meant for everybody — which is what every screen did before
	// zones existed, so nothing that is running today changes.
	Zone string `bson:"zone,omitempty" json:"zone,omitempty"`

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

// ---- What the screens play ----
//
// A branch's playlist: the pictures and the videos that loop on its
// televisions all day.
//
// ⚠️ **A playlist per branch, not per screen** — the same argument the mode
// field above makes from the other side. Two televisions in one room show the
// same restaurant's food; what differs between them is whether the order board
// is on, and that is a property of the wall. A playlist per screen would make
// somebody upload the same video four times and remember to change it four
// times, and the fourth one is the one that keeps showing last month's promo.

// TVSlide is one item in a branch's playlist.
type TVSlide struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId" json:"branchId"`

	// "image" or "video".
	//
	// ⚠️ **Stored rather than guessed from the extension.** The television has
	// to decide between an <Image> and a player before the file is opened, and
	// a URL that gains a query string, or a file served from somewhere else
	// later, would silently become a picture nobody can play.
	Kind string `bson:"kind" json:"kind"`

	// Where the file is. Ours, from the upload endpoints — see AdminTVUpload
	// for why a link to somebody else's server is not accepted.
	URL string `bson:"url" json:"url"`

	// What to call it in the panel. Never drawn on the television: a screen in
	// a dining room shows the picture, not our label for it.
	Name string `bson:"name" json:"name"`

	// How long a picture stays up, in seconds. Ignored for a video, which is as
	// long as it is — a clip cut off at ten seconds because somebody left the
	// default alone is the failure this field exists to avoid arguing about.
	Seconds int `bson:"seconds" json:"seconds"`

	// Where it sits in the loop. Contiguous is not required — reordering
	// rewrites all of them anyway — only the sort.
	Order int `bson:"order" json:"order"`

	// Off without being deleted: a seasonal promo comes back, and re-uploading
	// a video to show it again is how a restaurant ends up with four copies.
	Active bool `bson:"active" json:"active"`

	// Which zones this plays in.
	//
	// ⚠️ **Empty means everywhere, not nowhere**, and the whole design rests on
	// that being the default. Three things follow from it: every slide that
	// exists today keeps playing on every screen with no migration; "the same
	// loop on all the televisions" stays the thing you get without pressing
	// anything; and forgetting to set a zone shows a slide in *more* places
	// rather than fewer. That last one is why zones are a filter on a shared
	// playlist rather than a playlist per screen — the failure the per-screen
	// design was rejected for was a forgotten fourth list playing last month's
	// promotion, and a rule whose blank value means "everywhere" cannot fail
	// that way.
	//
	// ⚠️ Filtered on the **server**, beside Active and unlike the dates. The
	// split is not arbitrary: a screen's zone is a decision somebody made and
	// it does not change while the set is offline, whereas a date window closes
	// on its own at midnight with nobody watching.
	Zones []string `bson:"zones,omitempty" json:"zones,omitempty"`

	// The window this may play in, both optional.
	//
	// ⚠️ **The television filters by these itself**, against the server's clock
	// offset rather than its own — see the app. It is not the server trimming
	// the list, because a screen that has been offline since Friday must still
	// stop showing a promo that ended on Saturday. An expired offer on a wall
	// is worse than a blank one: a guest asks for it at the till.
	StartsAt *time.Time `bson:"startsAt,omitempty" json:"startsAt,omitempty"`
	EndsAt   *time.Time `bson:"endsAt,omitempty" json:"endsAt,omitempty"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// What a slide can be.
const (
	TVSlideImage = "image"
	TVSlideVideo = "video"
)

// TVSlidePlaysOn reports whether a slide belongs on a screen in this zone.
//
// ⚠️ **The one place the "empty means everywhere" rule is written.** It was a
// two-line condition at first and it existed in three files within a week — the
// handler, the panel and a test — which is three chances for one of them to
// read a blank list as "nowhere" and take a restaurant's whole playlist off its
// walls.
func TVSlidePlaysOn(zones []string, screenZone string) bool {
	if len(zones) == 0 {
		return true
	}
	return slices.Contains(zones, screenZone)
}

// ValidTVSlideKind reports whether this is something a television can draw.
func ValidTVSlideKind(kind string) bool {
	return kind == TVSlideImage || kind == TVSlideVideo
}

// How long a picture may be left up. The floor is what the eye needs to read a
// price; the ceiling is there because a "300" typed instead of "30" is a
// television that looks frozen, and the first thing anybody does about a frozen
// television is unplug it.
const (
	TVSlideMinSeconds = 3
	TVSlideMaxSeconds = 120
	TVSlideDefSeconds = 10
)

// ClampTVSlideSeconds keeps a picture's dwell time inside what a room can read.
//
// ⚠️ Zero means "not given" and becomes the default, not the floor: the panel
// sends no duration at all for a video.
func ClampTVSlideSeconds(n int) int {
	if n <= 0 {
		return TVSlideDefSeconds
	}
	if n < TVSlideMinSeconds {
		return TVSlideMinSeconds
	}
	if n > TVSlideMaxSeconds {
		return TVSlideMaxSeconds
	}
	return n
}

// TVSlidePlayable reports whether this slide may be on screen at `now`.
//
// ⚠️ **Pure, and the television runs it too.** The same rule has to give the
// same answer in the panel's preview, in the server's count and on a wall that
// has not reached the internet since Friday — and the only way three copies of
// a rule agree is for there to be one.
func TVSlidePlayable(s TVSlide, now time.Time) bool {
	if !s.Active {
		return false
	}
	if s.StartsAt != nil && now.Before(*s.StartsAt) {
		return false
	}
	if s.EndsAt != nil && now.After(*s.EndsAt) {
		return false
	}
	return true
}
