package handlers

import (
	"testing"

	"restaurant-backend/internal/models"
)

// ⚠️ **The palette a design was drawn with was stored and read by nobody**,
// and the second half is what made it serious: publishing a design also sets
// `designLocked`, which switches the owner's own theme editor off. So a
// customer with a drawn design had nobody at all who could change its colours —
// the console could not, because nothing read what it wrote, and the owner
// could not, because the lock is the point. This is the read.
func TestADrawnPaletteReachesTheSite(t *testing.T) {
	brand := models.SiteTheme{Brand: "#e2590d", Font: "modern"}
	drawn := models.SiteTheme{Brand: "#f04e23", Accent: "#ffd400"}

	got := mergeTheme(brand, drawn)
	if got.Brand != "#f04e23" {
		t.Errorf("the drawn accent did not reach the site: %q", got.Brand)
	}
	if got.Accent != "#ffd400" {
		t.Errorf("the second colour was lost: %q", got.Accent)
	}
	// ⚠️ And what the design said nothing about is left alone. A layout that
	// only rearranges bands carries an empty theme, and assigning it wholesale
	// would strip the font pairing a restaurant picked by hand — a change
	// nobody asked for, made by a document about something else.
	if got.Font != "modern" {
		t.Errorf("a field the design never set was overwritten: %q", got.Font)
	}
}

func TestADesignWithNoPaletteRepaintsNothing(t *testing.T) {
	brand := models.SiteTheme{
		Brand: "#e2590d", BrandDark: "#ff8a4c", Accent: "#00a3a3",
		Font: "soft", Background: "warm", Shadow: "strong",
		ButtonShape: "pill", ButtonStyle: "outline",
	}
	if got := mergeTheme(brand, models.SiteTheme{}); got != brand {
		t.Errorf("an empty theme repainted the site:\n got %+v\nwant %+v", got, brand)
	}
}

// ⚠️ **Zero is a real answer for both of these**, which is why they are
// pointers in the model: a radius of 0 is square corners — the whole look of
// half the shop references — and "unset" has to stay distinguishable from it.
// A plain int here would make "square corners" unexpressible.
func TestSquareCornersAreSettableAndUnsetIsNot(t *testing.T) {
	zero := 0
	base := models.SiteTheme{Radius: nil}

	got := mergeTheme(base, models.SiteTheme{Radius: &zero})
	if got.Radius == nil || *got.Radius != 0 {
		t.Errorf("square corners could not be chosen: %v", got.Radius)
	}

	sixteen := 16
	kept := mergeTheme(models.SiteTheme{Radius: &sixteen}, models.SiteTheme{})
	if kept.Radius == nil || *kept.Radius != 16 {
		t.Errorf("a radius the design never mentioned was cleared: %v", kept.Radius)
	}
}
