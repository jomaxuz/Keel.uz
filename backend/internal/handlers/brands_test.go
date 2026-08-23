package handlers

import (
	"testing"

	"restaurant-backend/internal/models"
)

// ⚠️ **A floor colour reaches an SVG `fill` on the public booking page.**
// Accepting whatever the form sent would make the plan editor a way to put
// arbitrary content onto every guest's screen, so the value is a name from a
// closed list — checked on the way in, where one save can be corrected, rather
// than in three renderers that each have to remember.
func TestFloorColoursAreCheckedOnSave(t *testing.T) {
	in := models.BookingSettings{
		Shapes: []models.FloorShape{
			{Color: "amber"},
			{Color: `" onload="alert(1)`},
			{Color: "puce"},
		},
		Tables: []models.FloorTable{{Color: "green"}, {Color: "#bada55"}},
	}
	out := clampFloorColors(in)

	if out.Shapes[0].Color != "amber" || out.Tables[0].Color != "green" {
		t.Error("a colour from the palette was thrown away")
	}
	// ⚠️ Emptied rather than refused: a colour is decoration, and rejecting a
	// whole floor plan over one is a save the owner cannot complete.
	for _, got := range []string{out.Shapes[1].Color, out.Shapes[2].Color, out.Tables[1].Color} {
		if got != "" {
			t.Errorf("an unknown colour survived: %q", got)
		}
	}
}
