package handlers

import (
	"testing"
	"time"

	"restaurant-backend/internal/models"
)

// ⚠️ **A date typed by somebody standing in the restaurant means that whole
// day.** An owner who sets a promotion to end on the 30th means it runs through
// the thirtieth; taking the date literally takes it off the wall as the doors
// open that morning, and the report of that arrives as "the screen is broken".
func TestASlideRunsThroughTheDayItEndsOn(t *testing.T) {
	start, end, err := tvSlideWindow("2026-09-01", "2026-09-30")
	if err != nil {
		t.Fatal(err)
	}
	if start == nil || end == nil {
		t.Fatal("both boundaries were given and one came back empty")
	}
	// Local, not UTC: parsed as UTC the boundary moves five hours in Tashkent —
	// the same clock the container's missing tzdata moved, the other way.
	if start.Location() != time.Local || end.Location() != time.Local {
		t.Error("a typed date is a local date")
	}
	noon := time.Date(2026, 9, 30, 12, 0, 0, 0, time.Local)
	evening := time.Date(2026, 9, 30, 23, 30, 0, 0, time.Local)
	slide := models.TVSlide{Active: true, StartsAt: start, EndsAt: end}
	for _, now := range []time.Time{noon, evening} {
		if !models.TVSlidePlayable(slide, now) {
			t.Errorf("the last day of the window is not playable at %s", now)
		}
	}
	next := time.Date(2026, 10, 1, 0, 30, 0, 0, time.Local)
	if models.TVSlidePlayable(slide, next) {
		t.Error("the promotion is still on the wall the morning after it ended")
	}
	before := time.Date(2026, 8, 31, 23, 0, 0, 0, time.Local)
	if models.TVSlidePlayable(slide, before) {
		t.Error("the promotion started the evening before it was meant to")
	}
}

// A window that can never be true is refused where somebody can still fix it —
// in the form, not on a wall that quietly plays nothing.
func TestAWindowThatEndsBeforeItStartsIsRefused(t *testing.T) {
	if _, _, err := tvSlideWindow("2026-09-30", "2026-09-01"); err == nil {
		t.Fatal("a backwards window was accepted")
	}
	// Both halves are optional, and neither is a boundary.
	start, end, err := tvSlideWindow("", "")
	if err != nil || start != nil || end != nil {
		t.Fatalf("an empty window should mean no boundary: %v %v %v", start, end, err)
	}
	if _, _, err := tvSlideWindow("30.09.2026", ""); err == nil {
		t.Fatal("a date in another format was accepted")
	}
}

// ⚠️ **Switched off is not the same as out of season**, and the television
// applies only the second. Off is a decision somebody made, filtered by the
// server; a window changes on its own, including while the screen is offline.
func TestASwitchedOffSlideNeverPlays(t *testing.T) {
	now := time.Now()
	if models.TVSlidePlayable(models.TVSlide{Active: false}, now) {
		t.Error("a slide switched off in the panel is on the wall")
	}
	if !models.TVSlidePlayable(models.TVSlide{Active: true}, now) {
		t.Error("a slide with no window at all should always play")
	}
}

// ⚠️ A "300" typed instead of "30" is a television that looks frozen, and the
// first thing anybody does about a frozen television is unplug it.
func TestAPictureStaysUpLongEnoughToReadAndNotLonger(t *testing.T) {
	if got := models.ClampTVSlideSeconds(0); got != models.TVSlideDefSeconds {
		t.Errorf("no duration should mean the default, got %d", got)
	}
	if got := models.ClampTVSlideSeconds(1); got != models.TVSlideMinSeconds {
		t.Errorf("a one-second slide is unreadable, got %d", got)
	}
	if got := models.ClampTVSlideSeconds(3600); got != models.TVSlideMaxSeconds {
		t.Errorf("an hour on one picture is a frozen screen, got %d", got)
	}
}

// ⚠️ **The extension is a claim.** This endpoint writes into a directory the
// whole internet reads, under our own domain — an HTML file stored as
// `promo.mp4` is a page we host and did not write.
func TestOnlyRealVideoBytesAreStored(t *testing.T) {
	mp4 := append([]byte{0, 0, 0, 0x18}, []byte("ftypmp42")...)
	webm := []byte{0x1A, 0x45, 0xDF, 0xA3, 0x01, 0x00, 0x00, 0x00}
	if !tvVideoSniff(mp4, ".mp4") || !tvVideoSniff(webm, ".webm") {
		t.Fatal("a real video was refused")
	}
	html := []byte("<!doctype html><script>alert(1)</script>")
	if tvVideoSniff(html, ".mp4") || tvVideoSniff(html, ".webm") {
		t.Fatal("a page uploaded as a video was accepted")
	}
	// A truncated file must not index past its own bytes.
	if tvVideoSniff([]byte("ft"), ".mp4") || tvVideoSniff(nil, ".webm") {
		t.Fatal("a short read was read as a video")
	}
	if tvVideoSniff(mp4, ".mov") {
		t.Fatal("an extension nothing plays in a dining room was accepted")
	}
}

// ⚠️ **The one place in this codebase that deletes an upload**, so the name it
// derives has to come out of a stored string safely: a path that walks out of
// the upload directory would remove something else entirely.
func TestADeletedVideoOnlyEverRemovesItsOwnFile(t *testing.T) {
	if got := uploadNameOf("https://osh.keel.uz/uploads/a1b2c3.mp4"); got != "a1b2c3.mp4" {
		t.Errorf("uploadNameOf = %q", got)
	}
	for _, bad := range []string{
		"https://osh.keel.uz/uploads/../../etc/passwd",
		"https://osh.keel.uz/uploads/",
		"https://osh.keel.uz/uploads/.thumb/600/x.webp",
		"https://cdn.example.com/promo.mp4",
		"",
	} {
		if got := uploadNameOf(bad); got != "" {
			t.Errorf("uploadNameOf(%q) = %q, want empty", bad, got)
		}
	}
}

// ---- Zones: which screen a slide belongs on ----

// ⚠️ **The default is "everywhere", and every other property of this feature
// rests on it.** A blank list read as "nowhere" would take every playlist in
// production off every wall the moment this shipped — there is no migration,
// and every slide that exists today has no zones.
func TestASlideWithNoZonesPlaysOnEveryScreen(t *testing.T) {
	for _, screenZone := range []string{"", "zal", "peshtaxta", "terrasa"} {
		if !models.TVSlidePlaysOn(nil, screenZone) {
			t.Fatalf("an untargeted slide did not play on %q", screenZone)
		}
		if !models.TVSlidePlaysOn([]string{}, screenZone) {
			t.Fatalf("an empty zone list did not play on %q", screenZone)
		}
	}
}

// A targeted slide plays where it was sent, and nowhere else.
func TestATargetedSlidePlaysOnlyInItsZones(t *testing.T) {
	zones := []string{"zal", "terrasa"}
	for _, on := range []string{"zal", "terrasa"} {
		if !models.TVSlidePlaysOn(zones, on) {
			t.Errorf("slide should play in %q", on)
		}
	}
	for _, off := range []string{"peshtaxta", "oshxona", ""} {
		if models.TVSlidePlaysOn(zones, off) {
			t.Errorf("slide should not play in %q", off)
		}
	}
}

// ⚠️ **A screen that has not been placed is not a wildcard.** It plays what is
// meant for everybody — which is every slide, on a branch that never uses zones
// — but a promotion aimed at the terrace must not appear on a set nobody has
// told us where to find.
func TestAnUnplacedScreenGetsOnlyTheUntargeted(t *testing.T) {
	if !models.TVSlidePlaysOn(nil, "") {
		t.Error("an unplaced screen must still get the shared playlist")
	}
	if models.TVSlidePlaysOn([]string{"zal"}, "") {
		t.Error("an unplaced screen must not receive targeted content")
	}
}

// ⚠️ **Case, and it is the whole reason cleanZones exists.** "Zal" typed on a
// slide against "zal" chosen on a screen is a promotion that plays nowhere,
// with every field filled in, no error anywhere, and the only symptom a dark
// television in a room somebody is standing in.
func TestZonesAreNormalisedSoTheyCanMatch(t *testing.T) {
	got := cleanZones([]string{"  Zal ", "TERRASA", "zal", "", "   "})
	want := []string{"zal", "terrasa"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// The list is capped, for the same reason the playlist is: a field that reaches
// the database from a form should not be able to grow without bound.
func TestTheZoneListIsCapped(t *testing.T) {
	in := make([]string, 60)
	for i := range in {
		in[i] = "zona" + itoa(i)
	}
	if n := len(cleanZones(in)); n > 20 {
		t.Fatalf("kept %d zones, cap is 20", n)
	}
}
