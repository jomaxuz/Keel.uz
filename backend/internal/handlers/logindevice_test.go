package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// ⚠️ **Two ways in, because two kinds of request carry it.** A login has a body
// and can put the claim in it; `me` has none and has to use headers — and it is
// the call every app makes on every launch, which is what keeps the "last seen"
// line and the IP current. One shape, read from both.
func TestDeviceClaimIsReadFromBodyThenHeaders(t *testing.T) {
	r := httptest.NewRequest("POST", "/", nil)
	r.Header.Set("X-Keel-Device", "from-header")
	r.Header.Set("X-Keel-App", "waiter")
	r.Header.Set("X-Keel-Platform", "android")
	r.Header.Set("X-Keel-Device-Name", "SM-A155F")

	// The body wins where it says anything: it is the more specific answer, and
	// an app that sends both is sending the same value twice.
	got := deviceFrom(r, deviceClaim{DeviceID: "from-body"})
	if got.DeviceID != "from-body" {
		t.Fatalf("body id was overwritten by the header: %q", got.DeviceID)
	}
	if got.App != "waiter" || got.Platform != "android" || got.Name != "SM-A155F" {
		t.Fatalf("headers were not read: %+v", got)
	}

	// And with nothing in the body, the headers answer the whole claim.
	got = deviceFrom(r, deviceClaim{})
	if got.DeviceID != "from-header" || got.App != "waiter" {
		t.Fatalf("header-only claim: %+v", got)
	}
}

// ⚠️ **A browser sends none of this, and that is what keeps the panel
// openable.** The lock is about the four apps; binding the panel to one laptop
// would be the opposite of what a panel is for, and the way that stays true is
// that an empty claim is simply not a claim.
func TestABrowserMakesNoClaim(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	got := deviceFrom(r, deviceClaim{})
	if got.DeviceID != "" || got.App != "" {
		t.Fatalf("a plain request produced a claim: %+v", got)
	}
}

// ⚠️ Whitespace and length are cut here rather than at the database: these
// strings are written into a row somebody reads in the panel, and an id padded
// with a newline would look identical to the one on the phone and never match.
func TestDeviceClaimIsTrimmedAndBounded(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	got := deviceFrom(r, deviceClaim{
		DeviceID: "  spaced  ",
		App:      " waiter ",
		Name:     strings.Repeat("x", 200),
	})
	if got.DeviceID != "spaced" || got.App != "waiter" {
		t.Fatalf("not trimmed: %+v", got)
	}
	if len(got.Name) > 60 {
		t.Fatalf("name was not bounded: %d chars", len(got.Name))
	}
}

// ⚠️ **A phone that is already signed in never signs in again**, and until this
// was fixed that meant it never appeared in the panel at all.
//
// Binding happens at the sign-in. A handset signed in before this feature
// existed — or before the build that sends the claim — keeps its token, calls
// `me` on every launch, and was answered by an update that matched nothing.
// The owner's own phone was the first report: "the device is not showing". The
// only cure was to sign out and back in, which nothing on any screen suggests
// and nobody has a reason to do.
//
// So `touchDevice` adopts an install with no row — and the care is in *when*:
// blindly writing one would let the second phone of a shared account take the
// binding on a launch, silently, which is precisely what the lock is for.
func TestAnAlreadySignedInPhoneIsAdopted(t *testing.T) {
	src := readSource(t, "logindevice.go")
	touch := between(t, src, "func (h *Handler) touchDevice", "\n}\n")
	if !strings.Contains(touch, "h.adoptDevice(") {
		t.Fatal("a phone that never signs in again is invisible to the panel again")
	}
	// ⚠️ Only when the update matched nothing. Adopting on every call would
	// insert beside a live row and lose to the unique index every launch.
	if !strings.Contains(touch, "res.MatchedCount > 0") {
		t.Fatal("adoption is no longer conditional on there being no row")
	}

	adopt := between(t, src, "func (h *Handler) adoptDevice", "\n}\n")
	// Both of bindDevice's questions, asked again: does this account already
	// hold a phone for this app, and is this install already somebody else's.
	if !strings.Contains(adopt, `"kind": kind, "subjectId": subject, "app": d.App`) {
		t.Fatal("a second phone can take the binding without signing in")
	}
	if !strings.Contains(adopt, `"app": d.App, "deviceId": d.DeviceID`) {
		t.Fatal("an install already bound to somebody else can be adopted")
	}
}
