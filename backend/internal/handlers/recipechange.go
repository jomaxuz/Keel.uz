package handlers

// ---- What changed on a tech card ----
//
// ⚠️ **The one channel in this product that steals without touching money.**
// A dish's card says 200g of beef; the kitchen puts in 150g. Every portion
// leaves 50g unaccounted for, the stock figures agree with the books perfectly
// — because the books were changed to agree with the theft — and a count finds
// nothing, because nothing is missing against a card that expects it to be
// gone. The only moment it is visible is the moment the card is edited.
//
// The audit log recorded that moment as `menu.update · Lag'mon` and nothing
// else. A whole-document replace, one line in the journal, and the numbers that
// changed nowhere at all.

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// recipeChange is one norm that moved.
type recipeChange struct {
	Name string
	Unit string
	From float64
	To   float64
}

// Up reports whether a dish now consumes more than it did.
//
// ⚠️ **Direction matters and cheapness does not.** A norm going *down* is a
// kitchen being told to use less, which shows up as a surplus at the next count
// and embarrasses nobody. A norm going up is the shape of the loss — and it is
// the same shape whether the ingredient is saffron or salt, because what walks
// out is the difference, not the line.
func (c recipeChange) Up() bool { return c.To > c.From }

// recipeDiff is what an edit did to a dish's card.
//
// ⚠️ **Read before the replace, because after it there is nothing to compare
// to.** `UpdateMenuItem` replaces the whole document, so the previous recipe
// exists only in the moment between the two.
func (h *Handler) recipeDiff(
	ctx context.Context, id primitive.ObjectID, next []models.RecipeLine,
) []recipeChange {
	var prev models.MenuItem
	if err := h.Store.Menu.FindOne(ctx, bson.M{"_id": id}).Decode(&prev); err != nil {
		// ⚠️ A dish being created has no previous card, and its whole recipe is
		// not a change — it is the first version. Reporting it as forty
		// increases would bury the one edit that matters under every new dish
		// the restaurant ever adds.
		return nil
	}
	was := map[primitive.ObjectID]float64{}
	for _, l := range prev.Recipe {
		was[l.IngredientID] = l.Qty
	}
	now := map[primitive.ObjectID]float64{}
	for _, l := range next {
		now[l.IngredientID] = l.Qty
	}

	ids := map[primitive.ObjectID]bool{}
	for k := range was {
		ids[k] = true
	}
	for k := range now {
		ids[k] = true
	}

	names := h.ingredientNames(ctx, ids)
	var out []recipeChange
	for id := range ids {
		from, to := was[id], now[id]
		// Floating point: two decimal grams are the same norm.
		if diffTiny(from, to) {
			continue
		}
		n := names[id]
		out = append(out, recipeChange{Name: n.name, Unit: n.unit, From: from, To: to})
	}
	// ⚠️ Sorted, because this string ends up in a journal a person reads. An
	// entry that reshuffles between two identical edits is one nobody can
	// compare against last month's.
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func diffTiny(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 0.001
}

type ingName struct{ name, unit string }

func (h *Handler) ingredientNames(
	ctx context.Context, ids map[primitive.ObjectID]bool,
) map[primitive.ObjectID]ingName {
	out := map[primitive.ObjectID]ingName{}
	list := make([]primitive.ObjectID, 0, len(ids))
	for id := range ids {
		list = append(list, id)
	}
	if len(list) == 0 {
		return out
	}
	cur, err := h.Store.Ingredients.Find(ctx, bson.M{"_id": bson.M{"$in": list}})
	if err != nil {
		return out
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var in models.Ingredient
		if cur.Decode(&in) == nil {
			out[in.ID] = ingName{name: in.Name, unit: in.Unit}
		}
	}
	return out
}

// describeRecipeDiff is the sentence the journal keeps.
//
// ⚠️ **Named ingredients and both numbers.** "Retsept o'zgardi" is the entry
// that made this invisible in the first place: it is true, it is in the
// journal, and it answers nothing. What a person opening the log a month later
// needs is which ingredient and from what to what — that is the whole
// difference between a record and a receipt for one.
func describeRecipeDiff(changes []recipeChange) string {
	if len(changes) == 0 {
		return ""
	}
	parts := make([]string, 0, len(changes))
	for _, c := range changes {
		switch {
		case c.From == 0:
			parts = append(parts, fmt.Sprintf("+%s %s%s", c.Name, num(c.To), c.Unit))
		case c.To == 0:
			parts = append(parts, fmt.Sprintf("−%s (%s%s edi)", c.Name, num(c.From), c.Unit))
		default:
			parts = append(parts, fmt.Sprintf("%s %s→%s%s",
				c.Name, num(c.From), num(c.To), c.Unit))
		}
	}
	// Bounded: an edit that rewrote a forty-line card should not put forty
	// lines into a journal row nobody can scan.
	const most = 8
	if len(parts) > most {
		return strings.Join(parts[:most], ", ") +
			fmt.Sprintf(" (+%d ta)", len(parts)-most)
	}
	return strings.Join(parts, ", ")
}

func num(f float64) string {
	s := fmt.Sprintf("%.3f", f)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

// biggestIncrease is the norm that went up the most, in proportion.
//
// ⚠️ **Proportion, not absolute quantity**, because the units are not
// comparable: 50 of anything is enormous in saffron and nothing in flour, and a
// threshold in grams would alert on every bread recipe and never on the one
// that matters.
//
// Returns nothing when the previous norm was zero: an ingredient added to a
// card is not an increase, it is a new line, and its arrival has ordinary
// causes — a garnish, a sauce, a dish being reworked.
func biggestIncrease(changes []recipeChange) (recipeChange, int, bool) {
	var best recipeChange
	bestPct := 0
	for _, c := range changes {
		if !c.Up() || c.From <= 0 {
			continue
		}
		pct := int((c.To - c.From) / c.From * 100)
		if pct > bestPct {
			best, bestPct = c, pct
		}
	}
	return best, bestPct, bestPct > 0
}

// RecipeIncreaseAlertPct is how much a norm has to grow to be worth a message.
//
// ⚠️ **A fifth, and it is high on purpose.** Cards get corrected: a chef who
// discovers the card says 180g and the ladle holds 200g is doing the right
// thing by fixing it, and that is a 11% edit nobody should be messaged about.
// A norm going up by a fifth is a different size of decision.
const RecipeIncreaseAlertPct = 20

// alertOnRecipeIncrease tells the owner when a dish starts consuming more.
//
// ⚠️ **Raised for every branch, because the card is the brand's.** A recipe is
// not owned by a room — the chain's three kitchens cook from one catalogue —
// so there is no single branch to attribute this to. It goes out under the
// company, which is also who it costs.
func (h *Handler) alertOnRecipeIncrease(
	r *http.Request, m models.MenuItem, changes []recipeChange,
) {
	c, pct, ok := biggestIncrease(changes)
	if !ok || pct < RecipeIncreaseAlertPct {
		return
	}
	branch := h.alertBranchFor(r)
	set := h.alertSettingsOf(r.Context(), branch)
	if !set.Enabled {
		return
	}
	name := ""
	if admin, err := h.adminUser(r); err == nil {
		name = admin.Name
		if name == "" {
			name = admin.Username
		}
	}
	h.raiseAlert(models.LossAlert{
		BranchID: branch,
		Kind:     models.AlertRecipeUp,
		By:       name,
		// ⚠️ Zero, and deliberately: what this costs depends on how many
		// portions get sold, which nobody knows yet. A figure invented here
		// would be the one number in the message that was made up.
		Amount:  0,
		Subject: m.Name + ": " + c.Name + " " + num(c.From) + "→" + num(c.To) + c.Unit,
		Reason:  fmt.Sprintf("+%d%%", pct),
		RefID:   m.ID,
	})
}

// alertBranchFor is which branch this admin's alerts belong to.
//
// ⚠️ An owner of the whole company is not pinned to a branch, and their edits
// have to reach somebody: the empty id is the company-wide settings document,
// which is the same shape every unscoped admin screen already uses.
func (h *Handler) alertBranchFor(r *http.Request) primitive.ObjectID {
	if admin, err := h.adminUser(r); err == nil {
		return admin.BranchID
	}
	return primitive.NilObjectID
}
