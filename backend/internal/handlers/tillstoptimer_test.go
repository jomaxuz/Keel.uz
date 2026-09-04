package handlers

import (
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
)

// ⚠️ **The list alone stopped being the answer, and everything that reads it
// has to go through the method.** A dish whose hour is up is still in
// `soldOut` — nothing removes it, deliberately — so a caller testing the array
// would keep a dish off the menu the timer had already released. That is
// exactly what a room reads as "the timer does not work".
func TestATimedStopLiftsItselfWhenTheHourIsUp(t *testing.T) {
	dish := primitive.NewObjectID()
	gone := time.Now().Add(-time.Minute)
	b := &models.Branch{
		SoldOut:      []primitive.ObjectID{dish},
		SoldOutUntil: []models.SoldOutTimer{{MenuItemID: dish, Until: gone}},
	}

	if b.IsManualSoldOut(dish) {
		t.Fatal("the deadline passed and the dish is still stopped")
	}
	// ⚠️ And through the door every other screen uses, not only the manual one.
	if b.IsSoldOut(dish) {
		t.Fatal("IsSoldOut still holds it — the site and the basket would refuse it")
	}
}

func TestATimedStopHoldsUntilItsTime(t *testing.T) {
	dish := primitive.NewObjectID()
	soon := time.Now().Add(time.Hour)
	b := &models.Branch{
		SoldOut:      []primitive.ObjectID{dish},
		SoldOutUntil: []models.SoldOutTimer{{MenuItemID: dish, Until: soon}},
	}

	if !b.IsManualSoldOut(dish) {
		t.Fatal("a stop with an hour left is not holding")
	}
}

// ⚠️ **A stop with no deadline is the default and must stay open-ended.** This
// is what the button did before timers existed, and every dish stopped by every
// restaurant until now has no entry in the parallel list.
func TestAStopWithNoDeadlineNeverLiftsOnItsOwn(t *testing.T) {
	dish := primitive.NewObjectID()
	b := &models.Branch{SoldOut: []primitive.ObjectID{dish}}

	if !b.IsManualSoldOut(dish) {
		t.Fatal("a stop with no timer lifted itself — every existing stop just came back on the menu")
	}
	if b.SoldOutUntilFor(dish) != nil {
		t.Fatal("a dish with no timer reports one")
	}
}

// ⚠️ **The server owns the clock.** The screen sends a duration; a till whose
// CMOS battery has died reports 2010 after a power cut — the reason offline
// check times are clamped — and a deadline computed there would either lift
// instantly or never.
func TestADurationIsMeasuredFromTheServersClock(t *testing.T) {
	now := time.Date(2026, 9, 4, 19, 0, 0, 0, time.Local)
	got := stopUntil(models.Branch{}, 120, false, now)

	if got == nil || !got.Equal(now.Add(2*time.Hour)) {
		t.Fatalf("until=%v, wanted two hours after 19:00", got)
	}
	// Zero is "until somebody says otherwise", which is what the button meant
	// before this existed.
	if stopUntil(models.Branch{}, 0, false, now) != nil {
		t.Fatal("no duration produced a deadline")
	}
	// A day is the ceiling: past that, open-ended is the honest setting.
	long := stopUntil(models.Branch{}, 99999, false, now)
	if long == nil || !long.Equal(now.Add(24*time.Hour)) {
		t.Fatalf("until=%v, wanted the 24h ceiling", long)
	}
}

// ⚠️ **A room that shuts at two in the morning shuts tomorrow.** Read as today,
// the deadline is already behind and the stop lifts on the next read — during
// the very evening it was meant to cover.
func TestUntilClosingCrossesMidnightForALateRoom(t *testing.T) {
	// A Friday evening at 21:00, in a place open until 02:00.
	now := time.Date(2026, 9, 4, 21, 0, 0, 0, time.Local)
	b := models.Branch{WorkingHours: []models.WorkingHour{
		{Day: int(now.Weekday()), Open: "10:00", Close: "02:00"},
	}}

	got := stopUntil(b, 0, true, now)
	if got == nil {
		t.Fatal("no deadline for 'until closing'")
	}
	want := time.Date(2026, 9, 5, 2, 0, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Fatalf("until=%v, wanted %v", got, want)
	}
}

func TestUntilClosingUsesTonightWhereTheRoomShutsTonight(t *testing.T) {
	now := time.Date(2026, 9, 4, 14, 0, 0, 0, time.Local)
	b := models.Branch{WorkingHours: []models.WorkingHour{
		{Day: int(now.Weekday()), Open: "10:00", Close: "23:00"},
	}}

	got := stopUntil(b, 0, true, now)
	want := time.Date(2026, 9, 4, 23, 0, 0, 0, time.Local)
	if got == nil || !got.Equal(want) {
		t.Fatalf("until=%v, wanted %v", got, want)
	}
}

// ⚠️ **A branch with no hours filled in is most of them**, and the fallback has
// to hold rather than lift: one that expired immediately would make the button
// look broken on exactly those installs. Same reading as an empty mapProvider.
func TestUntilClosingFallsBackToTheEndOfTheDay(t *testing.T) {
	now := time.Date(2026, 9, 4, 19, 30, 0, 0, time.Local)
	got := stopUntil(models.Branch{}, 0, true, now)

	want := time.Date(2026, 9, 5, 0, 0, 0, 0, time.Local)
	if got == nil || !got.Equal(want) {
		t.Fatalf("until=%v, wanted midnight %v", got, want)
	}
	if got.Before(now) {
		t.Fatal("the fallback deadline is already in the past — the stop would never hold")
	}
}

// ⚠️ **Every screen has to ask the same way.** The sales grid greys dishes from
// `soldOutIDs`, and while it read the array directly a timed stop stayed grey
// on the till after the stop-list screen had already shown it as available —
// two screens, one branch, opposite answers.
func TestTheSalesGridAsksThroughTheMethodToo(t *testing.T) {
	src := readSource(t, "till.go")
	fn := between(t, src, "func (h *Handler) soldOutIDs", "\n}\n")

	if !strings.Contains(fn, "branch.IsManualSoldOut(id)") {
		t.Fatal("the sales grid reads the manual list raw — a timed stop would not lift there")
	}
}
