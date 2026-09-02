package handlers

import (
	"strings"
	"testing"

	"restaurant-backend/internal/models"
)

// ⚠️ **A pairing code is read off a wall in a public room**, so every property
// that keeps it safe is a property of the code itself rather than of who sees
// it: short-lived, replaced constantly, single-use, and never enough on its own
// to collect the token.
func TestAPairingCodeIsReadableAndShortLived(t *testing.T) {
	// Nothing a person misreads across a dining room. O/0 and I/1 are the pair
	// that cost a support call every time; U and V are the pair that cost one
	// on a low-resolution screen.
	for _, bad := range []string{"O", "0", "I", "1", "U", "V"} {
		if strings.Contains(tvCodeAlphabet, bad) {
			t.Errorf("the code alphabet contains %q, which is misread", bad)
		}
	}
	if tvCodeLength < 6 {
		t.Fatalf("a %d-character code is guessable", tvCodeLength)
	}
	if tvCodeTTL.Minutes() > 3 {
		t.Fatal("a code that lives this long is a code somebody photographs")
	}

	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		code, err := newTVCode()
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != tvCodeLength {
			t.Fatalf("code %q is the wrong length", code)
		}
		if seen[code] {
			t.Fatalf("code %q came out twice in 200 draws", code)
		}
		seen[code] = true
	}
}

// ⚠️ **Typed by a person, from a screen that groups the characters.** Somebody
// reading "K7P — 4RM" types it the way they read it, and refusing that is a
// support call about a code that is on the screen in front of them.
func TestATypedCodeIsReadTheWayPeopleTypeIt(t *testing.T) {
	for _, in := range []string{"k7p4rm", "K7P 4RM", "k7p-4rm", "  K7P—4RM  "} {
		if got := normalizeTVCode(in); got != "K7P4RM" {
			t.Errorf("normalizeTVCode(%q) = %q", in, got)
		}
	}
}

// ⚠️ **The poll is answered by a secret, never by the code.** The code is
// visible to every guest in the room; if it were enough to ask "have I been
// claimed yet", anybody with a phone could collect the token meant for the wall
// by polling faster than the television does.
func TestTheCodeIsNotEnoughToCollectTheToken(t *testing.T) {
	src := readSource(t, "tvpair.go")
	fn := between(t, src, "func (h *Handler) TVPairStatus", "\n}\n")

	if !strings.Contains(fn, "pollSecret") {
		t.Fatal("the status poll no longer asks for the poll secret")
	}
	if strings.Contains(fn, `Query().Get("code")`) {
		t.Fatal("the status poll accepts the code that is shown on the wall")
	}
	// Constant time, and the same reply for a wrong secret as for an expired
	// code: a different answer tells a guessing client it found a live screen.
	if !strings.Contains(fn, "subtle.ConstantTimeCompare") {
		t.Fatal("the poll secret is compared byte by byte")
	}
	// Handed over once. A second caller — a late retry, or somebody who learned
	// the secret — must get nothing.
	if !strings.Contains(fn, "DeleteOne") {
		t.Fatal("the token can be collected twice")
	}
}

// ⚠️ **Asking for a new code kills the old one.** The television asks every ten
// seconds; if each call added a row, one screen would have a dozen working
// codes — including the one somebody photographed a minute ago.
func TestANewCodeReplacesTheOldOne(t *testing.T) {
	fn := between(t, readSource(t, "tvpair.go"),
		"func (h *Handler) TVPairStart", "\n}\n")

	if !strings.Contains(fn, "UpdateOne") || !strings.Contains(fn, "SetUpsert(true)") {
		t.Fatal("a new code is added beside the old one instead of replacing it")
	}
	if !strings.Contains(fn, `"installId": install`) {
		t.Fatal("the pairing is no longer keyed to one television")
	}
	// ⚠️ And the token from a previous pairing is cleared: a set that was
	// claimed and asks for a new code is being paired again, and a stale token
	// left in the row would be handed to whoever claims it next.
	if !strings.Contains(fn, `"token":    ""`) {
		t.Fatal("a stale token survives into the new pairing")
	}
}

// ⚠️ **A token that outlives its screen is the whole risk of a year-long
// token.** "Unpair this television" has to mean the next request fails — not
// the next year — so the version is compared on every request, against both the
// screen's row and the branch's counter.
func TestAnUnpairedScreenStopsAtOnce(t *testing.T) {
	fn := between(t, readSource(t, "tvpair.go"),
		"func (h *Handler) tvScreen(", "\n}\n")

	if !strings.Contains(fn, "claims.Ver != screen.Version") {
		t.Fatal("a token issued before this screen was re-paired still works")
	}
	if !strings.Contains(fn, "screen.Version != branch.TVVersion") {
		t.Fatal("revoking a branch's screens no longer stops them")
	}
}

// The cap is the number the module is billed by, so the two answers that must
// never be got backwards are argued with here rather than in a restaurant.
//
// ⚠️ **No subscription means no cap**, exactly as it does for registers: a
// limit that switches itself on for customers who were never sold one takes a
// working feature away on a deploy.
func TestScreenCapCountsOnlyWhatWasSold(t *testing.T) {
	if tvCapExceeded(nil, 99) {
		t.Fatal("an install with no subscription was capped")
	}
	if tvCapExceeded(&models.Subscription{Enabled: false, Screens: 1}, 5) {
		t.Fatal("a disabled subscription was read as a cap")
	}
	if tvCapExceeded(&models.Subscription{Enabled: true, Screens: 0}, 99) {
		t.Fatal("0 screens was read as a limit of nothing rather than no limit")
	}
	sub := &models.Subscription{Enabled: true, Screens: 2}
	if tvCapExceeded(sub, 1) {
		t.Fatal("the second of two screens was refused")
	}
	if !tvCapExceeded(sub, 2) {
		t.Fatal("a third screen was allowed on a plan that sells two")
	}
}

// ⚠️ **Claiming a code is what spends a paid slot and puts a screen in a room**
// — so it needs somebody with access to that branch. Anybody in the dining room
// can read the code off the wall; only the panel can say which branch it is.
func TestClaimingACodeNeedsTheBranch(t *testing.T) {
	fn := between(t, readSource(t, "tvscreens.go"),
		"func (h *Handler) AdminTVClaim", "\n}\n")

	if !strings.Contains(fn, "h.requireBranchAccess(r, branchID)") {
		t.Fatal("a code can be claimed into a branch the admin cannot reach")
	}
	// ⚠️ The token goes to the pairing row, not back to the browser: it belongs
	// to the wall, and a browser that has held it once is a browser it can leak
	// from.
	if strings.Contains(fn, `"token": token,`) &&
		!strings.Contains(fn, "TVPairings.UpdateByID") {
		t.Fatal("the screen's token is being returned to the panel")
	}
	// The same set paired twice must not spend two slots.
	if !strings.Contains(fn, `bson.M{"installId": pairing.InstallID}`) {
		t.Fatal("a re-paired television opens a second row and a second slot")
	}
}
