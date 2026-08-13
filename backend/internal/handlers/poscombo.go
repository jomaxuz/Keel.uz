package handlers

import (
	"context"
	"sort"
	"strings"

	"restaurant-backend/internal/models"
	"restaurant-backend/internal/pos"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Combos, on their way to the till.
//
// ⚠️ **A combo is ours, not the till's.** "Oilaviy combo" is a bundle the
// restaurant built here out of dishes that already exist; iiko, Poster and
// Clopos have never heard of it. Sent as one line it needs a product on the
// other side that nobody created, and `pos.CheckMapped` then refuses the
// **whole order** — the guest pays, the panel says confirmed, and the kitchen
// prints nothing. That is the worst failure this integration has, because it
// is silent on the only side that could fix it.
//
// So a set is expanded into the dishes it is made of. Each of those is already
// on the menu and already mapped, the kitchen reads what it has to cook rather
// than a name whose contents it has to remember, and nobody maintains their
// combos twice.
//
// The price is the part that needs care: a set sells for less than its parts,
// and a ticket whose lines add up to more than the guest paid is the first
// thing a shift manager queries. So the combo's own price is split across the
// member lines, exactly — see splitAmount.

// posLine is one till line together with the menu dish it came from, which is
// what the branch's mapping is keyed by.
type posLine struct {
	Item   pos.Item
	Source primitive.ObjectID
}

// splitAmount divides total across weights so the parts sum to exactly total,
// giving the spare so'm to the largest dropped fractions.
//
// ⚠️ Exactness is the whole point, and rounding each share on its own does not
// give it: three equal shares of 40 000 rounded to 13 333 lose a so'm, and that
// so'm has to come off somebody's receipt. Here the remainder is handed out one
// at a time, largest fraction first, so the total is preserved and the line it
// lands on is the one that came closest to earning it.
//
// Weights that sum to zero (a free set, or members priced at zero) split
// evenly — still exactly. Both cases should be impossible; neither may produce
// a ticket that fails to add up.
func splitAmount(total int, weights []int) []int {
	out := make([]int, len(weights))
	if len(weights) == 0 || total <= 0 {
		return out
	}
	sum := 0
	for _, w := range weights {
		if w > 0 {
			sum += w
		}
	}
	if sum == 0 {
		base, rem := total/len(weights), total%len(weights)
		for i := range out {
			out[i] = base
			if i < rem {
				out[i]++
			}
		}
		return out
	}

	type share struct {
		idx  int
		frac int // the dropped fraction's numerator, over sum
	}
	shares := make([]share, 0, len(weights))
	given := 0
	for i, w := range weights {
		if w < 0 {
			w = 0
		}
		whole := total * w / sum
		out[i] = whole
		given += whole
		shares = append(shares, share{idx: i, frac: total*w - whole*sum})
	}
	// Largest fraction first; ties by original position, so the same input
	// always produces the same ticket.
	sort.SliceStable(shares, func(a, b int) bool {
		if shares[a].frac != shares[b].frac {
			return shares[a].frac > shares[b].frac
		}
		return shares[a].idx < shares[b].idx
	})
	for i := 0; given < total; i++ {
		out[shares[i%len(shares)].idx]++
		given++
	}
	return out
}

// comboMember is one dish inside a set, as the till needs it.
type comboMember struct {
	MenuItemID primitive.ObjectID
	Name       string
	Qty        int
	Price      int // the dish's own price — only ever the split's weight
}

// comboMembersFor returns what a combo order line is made of, or nil when that
// cannot be established.
//
// Prefers the copy frozen onto the order: that is what was sold, and a set
// rebuilt since must not change a ticket already paid for. Falls back to the
// live definition for orders taken before those fields existed — without the
// fallback every combo ordered before this change would refuse to print.
func (h *Handler) comboMembersFor(
	ctx context.Context, line models.OrderItem,
) []comboMember {
	frozen := make([]comboMember, 0, len(line.ComboItems))
	complete := len(line.ComboItems) > 0
	for _, c := range line.ComboItems {
		if c.MenuItemID.IsZero() {
			complete = false
			break
		}
		frozen = append(frozen, comboMember{
			MenuItemID: c.MenuItemID, Name: c.Name, Qty: atLeastOne(c.Qty), Price: c.Price,
		})
	}
	if complete {
		return frozen
	}

	var combo models.MenuItem
	if err := h.Store.Menu.FindOne(ctx,
		bson.M{"_id": line.MenuItemID}).Decode(&combo); err != nil {
		return nil
	}
	if len(combo.ComboItems) == 0 {
		return nil
	}
	ids := make([]primitive.ObjectID, 0, len(combo.ComboItems))
	for _, c := range combo.ComboItems {
		ids = append(ids, c.MenuItemID)
	}
	cur, err := h.Store.Menu.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil
	}
	var dishes []models.MenuItem
	if err := cur.All(ctx, &dishes); err != nil {
		return nil
	}
	byID := make(map[primitive.ObjectID]models.MenuItem, len(dishes))
	for _, d := range dishes {
		byID[d.ID] = d
	}
	out := make([]comboMember, 0, len(combo.ComboItems))
	for _, c := range combo.ComboItems {
		d, ok := byID[c.MenuItemID]
		if !ok {
			// A member has been deleted from the menu. Sending the rest would
			// print a set missing a course, so the line cannot be expanded and
			// the caller stops the order with a name the panel can show.
			return nil
		}
		out = append(out, comboMember{
			MenuItemID: d.ID, Name: d.Name, Qty: atLeastOne(c.Qty), Price: d.Price,
		})
	}
	return out
}

func atLeastOne(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

// expandCombo turns one combo order line into the till lines it is made of.
//
// The set's unit price is split across one slot per portion — a member ordered
// twice gets two slots — so every share is a whole so'm per portion and the
// line still totals exactly what the guest paid. Portions of the same dish that
// land on the same price merge back into one line; an uneven split shows up as
// a second line for that dish, which is honest, rather than as a rounding
// error nobody can trace.
func expandCombo(line models.OrderItem, members []comboMember) []posLine {
	weights := make([]int, 0, 8)
	owner := make([]int, 0, 8)
	for i, m := range members {
		for q := 0; q < m.Qty; q++ {
			weights = append(weights, m.Price)
			owner = append(owner, i)
		}
	}
	if len(weights) == 0 {
		return nil
	}
	shares := splitAmount(line.Price, weights)

	type key struct{ member, price int }
	seen := map[key]int{}
	order := make([]key, 0, len(weights))
	for i, s := range shares {
		k := key{member: owner[i], price: s}
		if seen[k] == 0 {
			order = append(order, k)
		}
		seen[k]++
	}

	qty := atLeastOne(line.Qty)
	out := make([]posLine, 0, len(order))
	for _, k := range order {
		m := members[k.member]
		out = append(out, posLine{
			Source: m.MenuItemID,
			Item: pos.Item{
				Name:  m.Name,
				Qty:   seen[k] * qty,
				Price: k.price,
				// Which set these lines came out of, so the kitchen sees one
				// order rather than four unrelated dishes. The guest's own note
				// rides along: "piyozsiz" was written about a dish in the set.
				Comment: joinNonEmpty(" · ", line.Name, line.Comment),
			},
		})
	}
	return out
}

// joinNonEmpty joins the parts that are actually there.
func joinNonEmpty(sep string, parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
}
