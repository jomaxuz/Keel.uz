package repository

import (
	"os"
	"strings"
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

// ⚠️ **Every warehouse read is "this branch, this period", and two of them run
// on screens people keep open.** The flow report walks the period's deliveries
// and write-offs; the ingredient list computes what should be on the shelf on
// every load, which reads the last count and everything after it. Without these
// indexes that is a scan of collections which only grow — on a **shared**
// mongod, which makes one busy restaurant everybody else's problem. The order
// history already taught this lesson once (COLLSCAN, measured on a live
// tenant), which is why it is a test and not a comment.
func TestWarehouseQueriesHaveIndexes(t *testing.T) {
	src := readMigrateSource(t)
	for _, want := range []string{
		`{s.Purchases, bson.D{{Key: "branchId", Value: 1}, {Key: "at", Value: -1}}}`,
		`{s.WriteOffs, bson.D{{Key: "branchId", Value: 1}, {Key: "at", Value: -1}}}`,
		`{s.Transfers, bson.D{{Key: "branchId", Value: 1}, {Key: "at", Value: -1}}}`,
		`{s.Stocktakes, bson.D{{Key: "branchId", Value: 1}, {Key: "at", Value: -1}}}`,
		`{s.Menu, bson.D{{Key: "recipe.ingredientId", Value: 1}}}`,
		`{s.PrintJobs, bson.D{{Key: "branchId", Value: 1}, {Key: "createdAt", Value: 1}}}`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("missing index: %s", want)
		}
	}
}

func readMigrateSource(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("migrate.go")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
