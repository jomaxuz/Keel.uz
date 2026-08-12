package handlers

import (
	"net/http"
	"strings"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Which kitchen's stop list the public menu is drawn against.
//
// ⚠️ **The honest answer depends on a question the guest has not been asked
// yet.** They are browsing; the address that decides which branch cooks comes
// later, at the checkout. So on a multi-branch install "is lag'mon available?"
// has no single true answer while the menu is on screen.
//
// It used to be answered with the **default** branch's list — whichever branch
// happened to sort first. That is not optimistic or pessimistic, it is
// arbitrary, and it was wrong in both directions at once: it hid dishes the
// company could deliver, and it offered dishes the guest's own branch had run
// out of.
//
// The rule now:
//
//   - **One branch, or a branch already chosen** (a table QR, a pickup branch,
//     the site's branch cookie): that branch's list, exactly as before. This is
//     every single-branch install, which is most of them — nothing changes.
//   - **Several branches and none chosen**: sold out only where **every** branch
//     has it stopped. That is the one statement that stays true whichever
//     kitchen ends up serving: "you cannot get this from us right now".
//
// The optimistic side of that is deliberate and is only affordable because the
// checkout now catches the per-branch case by name (see quote's `soldOut`).
// Without that this rule would move the disappointment later; with it, the menu
// stops hiding food the restaurant can actually sell.

// soldOutLens answers "can this dish be ordered right now". Nil means the
// caller does not know which kitchen and does not want the check at all.
type soldOutLens func(primitive.ObjectID) bool

// publicSoldOut builds the lens for a guest browsing the menu.
//
// Also returns the branch when one was actually resolved, because callers that
// have other per-branch business (the site profile, the floor plan) still need
// it; it is nil precisely when the lens is the across-all-branches one.
func (h *Handler) publicSoldOut(
	r *http.Request, brandID primitive.ObjectID,
) (soldOutLens, *models.Branch) {
	raw := strings.TrimSpace(r.URL.Query().Get("branchId"))
	branch, err := h.bookingBranch(r, raw)
	if err != nil {
		return nil, nil
	}
	// The guest named a branch: they are at its table, collecting from it, or
	// the site is being served in its name. No guessing needed.
	if raw != "" {
		return branch.IsSoldOut, branch
	}

	filter := bson.M{"isActive": true}
	if !brandID.IsZero() {
		filter["brandId"] = brandID
	}
	branches, err := h.listBranchesFiltered(r, filter)
	// One branch — the ordinary restaurant — is the same answer either way, and
	// keeping it on this path means the common case never depends on the rule
	// below being right.
	if err != nil || len(branches) <= 1 {
		return branch.IsSoldOut, branch
	}

	// Several kitchens, none chosen. A dish is only unavailable when nowhere
	// has it: anything else would hide food the company can deliver, from a
	// guest whose own branch has plenty.
	stopped := everywhereSoldOut(branches)
	return func(id primitive.ObjectID) bool { return stopped[id] }, nil
}

// everywhereSoldOut is the set of dishes stopped at every one of these
// branches. Built by intersection rather than by counting hits, so a dish that
// only one branch has ever heard of cannot end up in it.
func everywhereSoldOut(branches []models.Branch) map[primitive.ObjectID]bool {
	if len(branches) == 0 {
		return nil
	}
	// Start from the first branch's list and keep only what survives in all
	// the others.
	stopped := map[primitive.ObjectID]bool{}
	for _, id := range branches[0].SoldOut {
		stopped[id] = true
	}
	for _, id := range branches[0].POSSoldOut {
		stopped[id] = true
	}
	for i := 1; i < len(branches); i++ {
		b := &branches[i]
		for id := range stopped {
			if !b.IsSoldOut(id) {
				delete(stopped, id)
			}
		}
		if len(stopped) == 0 {
			break
		}
	}
	return stopped
}
