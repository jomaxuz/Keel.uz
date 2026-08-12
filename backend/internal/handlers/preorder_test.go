package handlers

import (
	"testing"
	"time"

	"restaurant-backend/internal/models"
)

// A branch that takes pre-orders, open 10:00–22:00 every day.
func preorderBranch() *models.Branch {
	hours := make([]models.WorkingHour, 0, 7)
	for d := range 7 {
		hours = append(hours, models.WorkingHour{Day: d, Open: "10:00", Close: "22:00"})
	}
	return &models.Branch{
		WorkingHours: hours,
		Preorder: models.PreorderSettings{
			Enabled: true, LeadMinutes: 60, MinMinutes: 60, MaxDays: 3, SlotMinutes: 30,
		},
	}
}

func at(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.ParseInLocation("2006-01-02 15:04", s, time.Local)
	if err != nil {
		t.Fatalf("bad time %q: %v", s, err)
	}
	return v
}

func rfc(v time.Time) string { return v.Format(time.RFC3339) }

// The bell is the whole feature, and this is the arithmetic behind it: the
// kitchen hears about a pre-order one lead time before it is due, not when it
// was placed. Getting this backwards is invisible in the panel — the order
// looks perfectly normal — and shows up only as food cooked at the wrong hour.
func TestPreorderQueueAtSubtractsTheLead(t *testing.T) {
	b := preorderBranch()
	placed := at(t, "2026-08-12 12:00")
	scheduled := at(t, "2026-08-12 19:00")

	got := preorderQueueAt(b, scheduled, placed)
	if want := at(t, "2026-08-12 18:00"); !got.Equal(want) {
		t.Fatalf("queue at = %v, want %v", got, want)
	}

	// ⚠️ Never before the order existed. An operator taking one for "in ten
	// minutes" has a lead longer than the notice, and the naive subtraction
	// would put the order in the queue before it was written — it would read as
	// having waited an hour the moment it was placed, and jump the pass ahead of
	// tickets that really have.
	soon := at(t, "2026-08-12 12:10")
	if got := preorderQueueAt(b, soon, placed); !got.Equal(placed) {
		t.Fatalf("near-term pre-order queued at %v, want the placement time %v", got, placed)
	}
}

func TestPreorderRejectedWhenBranchDoesNotTakeThem(t *testing.T) {
	b := preorderBranch()
	b.Preorder.Enabled = false
	now := at(t, "2026-08-12 12:00")

	if _, err := resolvePreorder(b, rfc(at(t, "2026-08-12 19:00")), false, now); err == nil {
		t.Fatal("a branch with pre-orders off accepted one")
	}
	// ⚠️ And the operator is not exempt from this one. The other rules are form
	// validation; this is the owner's decision about their own kitchen, and an
	// operator quietly overriding it means the switch does not mean anything.
	if _, err := resolvePreorder(b, rfc(at(t, "2026-08-12 19:00")), true, now); err == nil {
		t.Fatal("the operator bypassed the branch's own switch")
	}
}

// Nothing scheduled is the ordinary order, and it must stay ordinary: no time,
// no error, no pre-order behaviour anywhere downstream.
func TestPreorderEmptyMeansOrdinaryOrder(t *testing.T) {
	got, err := resolvePreorder(preorderBranch(), "  ", false, at(t, "2026-08-12 12:00"))
	if err != nil || got != nil {
		t.Fatalf("empty scheduledAt = (%v, %v), want (nil, nil)", got, err)
	}
}

func TestPreorderGuestRules(t *testing.T) {
	b := preorderBranch()
	now := at(t, "2026-08-12 12:00")

	cases := []struct {
		name string
		want string // "" = accepted
		time time.Time
	}{
		{"too soon", "rejected", at(t, "2026-08-12 12:30")},
		{"just inside the notice", "", at(t, "2026-08-12 13:30")},
		{"after closing", "rejected", at(t, "2026-08-12 23:00")},
		{"before opening", "rejected", at(t, "2026-08-13 09:00")},
		{"last allowed day", "", at(t, "2026-08-15 20:00")},
		{"past the horizon", "rejected", at(t, "2026-08-16 20:00")},
		{"in the past", "rejected", at(t, "2026-08-12 09:00")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := resolvePreorder(b, rfc(c.time), false, now)
			if c.want == "rejected" {
				if err == nil {
					t.Fatalf("%v was accepted", c.time)
				}
				return
			}
			if err != nil {
				t.Fatalf("%v was rejected: %v", c.time, err)
			}
			if !got.Equal(c.time) {
				t.Fatalf("stored %v, want %v", got, c.time)
			}
		})
	}
}

// ⚠️ An operator is exempt from the timing rules, deliberately: "in twenty
// minutes" and "for the wedding in six weeks" are both normal things to say on
// the phone, and a rule written for a web form must not stop a restaurant
// taking an order it is willing to cook.
func TestPreorderOperatorIsExemptFromTimingButNotFromThePast(t *testing.T) {
	b := preorderBranch()
	now := at(t, "2026-08-12 12:00")

	for _, when := range []time.Time{
		at(t, "2026-08-12 12:15"), // inside the notice
		at(t, "2026-08-12 23:30"), // after closing
		at(t, "2026-10-01 19:00"), // past the horizon
	} {
		if _, err := resolvePreorder(b, rfc(when), true, now); err != nil {
			t.Fatalf("operator was refused %v: %v", when, err)
		}
	}
	if _, err := resolvePreorder(b, rfc(at(t, "2026-08-12 11:00")), true, now); err == nil {
		t.Fatal("an order dated into the past was accepted")
	}
}

// A settings form is one mistyped zero away from a restaurant nobody can order
// from: a lead of 5000 minutes makes every slot a guest is allowed to pick one
// the kitchen is already late for.
func TestClampPreorderKeepsSettingsUsable(t *testing.T) {
	got := clampPreorder(models.PreorderSettings{
		Enabled: true, LeadMinutes: 5000, MinMinutes: 9000, MaxDays: 900, SlotMinutes: 0,
	})
	if got.LeadMinutes != maxPreorderLeadMinutes || got.MinMinutes != maxPreorderLeadMinutes {
		t.Fatalf("lead/notice not clamped: %+v", got)
	}
	if got.MaxDays != maxPreorderDays {
		t.Fatalf("maxDays not clamped: %+v", got)
	}
	if got.SlotMinutes != 30 {
		t.Fatalf("slot minutes = %d, want the 30-minute default", got.SlotMinutes)
	}
	// ⚠️ The switch itself is never rewritten. Defaults fill in blanks; they do
	// not turn a feature on for a restaurant that did not ask for it.
	if off := clampPreorder(models.PreorderSettings{}); off.Enabled {
		t.Fatal("clamping switched pre-orders on")
	}
}
