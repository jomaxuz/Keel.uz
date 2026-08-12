package handlers

import (
	"testing"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func candidate(name string, dist float64, full bool) branchCandidate {
	return branchCandidate{
		Branch: &models.Branch{Name: name},
		Dist:   dist,
		Full:   full,
	}
}

// The rule the whole multi-branch routing rests on, and the one a live install
// found the hard way: a kitchen that can cook the basket beats a nearer one
// that cannot. Before this, one finished dish at the closest branch refused an
// order the company could perfectly well fill three kilometres away.
func TestBestBranchPrefersOneThatCanCookTheBasket(t *testing.T) {
	got := bestBranch([]branchCandidate{
		candidate("near-but-out", 2, false),
		candidate("further-but-stocked", 5, true),
		candidate("furthest-stocked", 9, true),
	})
	if got == nil || got.Branch.Name != "further-but-stocked" {
		t.Fatalf("chose %v, want the nearest branch that has everything", got)
	}
}

// Among branches that can all fill it, nothing changed: nearest wins. A shorter
// run is hotter food, and every candidate already agreed to carry this address.
func TestBestBranchNearestAmongEquals(t *testing.T) {
	got := bestBranch([]branchCandidate{
		candidate("far", 12, true),
		candidate("near", 3, true),
		candidate("middle", 7, true),
	})
	if got == nil || got.Branch.Name != "near" {
		t.Fatalf("chose %v, want the nearest", got)
	}
}

// ⚠️ The half that is easy to lose. When **nobody** has everything the answer is
// still the nearest branch, not "no branch" — that branch's name and stop list
// are what produce the honest "lag'mon bugun tugadi". Returning nothing would
// turn a refusal the guest can act on into "we don't deliver here", which is a
// different and untrue statement.
func TestBestBranchFallsBackWhenNobodyHasEverything(t *testing.T) {
	got := bestBranch([]branchCandidate{
		candidate("far", 12, false),
		candidate("near", 3, false),
	})
	if got == nil || got.Branch.Name != "near" {
		t.Fatalf("chose %v, want the nearest so the refusal can name the dish", got)
	}
}

// Nothing covers the address at all: that is the genuine "we don't deliver
// here", and it has to stay distinguishable from the case above.
func TestBestBranchNilWhenNothingCovers(t *testing.T) {
	if got := bestBranch(nil); got != nil {
		t.Fatalf("chose %v from no candidates", got)
	}
}

// A basket is only sellable where every dish in it is — including the courses
// inside a combo, which is why basketDishes expands them.
func TestAnySoldOutAsksTheBranchForEveryDish(t *testing.T) {
	stopped := primitive.NewObjectID()
	fine := primitive.NewObjectID()
	// One list is the counter's, the other is mirrored from the till. Either
	// makes a dish unsellable, and routing has to respect both.
	counter := &models.Branch{SoldOut: []primitive.ObjectID{stopped}}
	till := &models.Branch{POSSoldOut: []primitive.ObjectID{stopped}}

	for name, b := range map[string]*models.Branch{"counter": counter, "till": till} {
		if !anySoldOut(b, []primitive.ObjectID{fine, stopped}) {
			t.Errorf("%s: a stopped dish was not noticed", name)
		}
		if anySoldOut(b, []primitive.ObjectID{fine}) {
			t.Errorf("%s: a stocked basket was reported sold out", name)
		}
	}
	// No basket in hand is the plain "what would delivery cost here" question,
	// and it must never exclude a branch.
	if anySoldOut(counter, nil) {
		t.Error("an empty basket excluded a branch")
	}
}
