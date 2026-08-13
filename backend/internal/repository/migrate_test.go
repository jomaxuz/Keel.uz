package repository

import (
	"testing"

	"restaurant-backend/internal/models"
)

func bands(types ...string) []models.DesignSection {
	out := make([]models.DesignSection, 0, len(types))
	for _, t := range types {
		out = append(out, models.DesignSection{Type: t, Span: 12})
	}
	return out
}

func typesOf(sections []models.DesignSection) []string {
	out := make([]string, 0, len(sections))
	for _, s := range sections {
		out = append(out, s.Type)
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Where the band lands, and — more importantly — when it does not land at all.
func TestWithReviewsBand(t *testing.T) {
	cases := []struct {
		name  string
		in    []string
		want  []string
		added bool
	}{
		{
			name:  "before the address, as on the site's own page",
			in:    []string{models.BlockHero, models.BlockMenuGrid, models.BlockHoursAddress},
			want:  []string{models.BlockHero, models.BlockMenuGrid, models.BlockReviews, models.BlockHoursAddress},
			added: true,
		},
		{
			name:  "above the footer when there is no address band",
			in:    []string{models.BlockNavbar, models.BlockCanvas, models.BlockFooter},
			want:  []string{models.BlockNavbar, models.BlockCanvas, models.BlockReviews, models.BlockFooter},
			added: true,
		},
		{
			name:  "at the end when there is neither",
			in:    []string{models.BlockHero, models.BlockMenuGrid},
			want:  []string{models.BlockHero, models.BlockMenuGrid, models.BlockReviews},
			added: true,
		},
		{
			name:  "the real kfc layout",
			in:    []string{models.BlockNavbar, models.BlockCanvas, models.BlockMenuGrid, models.BlockAbout, models.BlockFooter},
			want:  []string{models.BlockNavbar, models.BlockCanvas, models.BlockMenuGrid, models.BlockAbout, models.BlockReviews, models.BlockFooter},
			added: true,
		},
		{
			name:  "a design that already has one is left alone",
			in:    []string{models.BlockHero, models.BlockReviews, models.BlockHoursAddress},
			want:  []string{models.BlockHero, models.BlockReviews, models.BlockHoursAddress},
			added: false,
		},
		{
			name:  "an undrawn design stays undrawn",
			in:    nil,
			want:  nil,
			added: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, added := WithReviewsBand(bands(c.in...))
			if added != c.added {
				t.Fatalf("added=%v, want %v", added, c.added)
			}
			if !equal(typesOf(got), c.want) {
				t.Fatalf("got %v, want %v", typesOf(got), c.want)
			}
		})
	}
}

// The band the migration inserts has to survive the tenant's own sanitiser —
// one that gets dropped on read is a migration that appears to have worked and
// changed nothing.
func TestInsertedBandSurvivesSanitize(t *testing.T) {
	sections, added := WithReviewsBand(bands(models.BlockHero, models.BlockHoursAddress))
	if !added {
		t.Fatal("band was not added")
	}
	d := &models.PageDesign{Status: models.DesignPublished, Sections: sections}
	d.Sanitize()
	if indexOfBand(d.Sections, models.BlockReviews) < 0 {
		t.Fatalf("the sanitiser dropped the band: %v", typesOf(d.Sections))
	}
}
