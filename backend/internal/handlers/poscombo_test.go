package handlers

import (
	"testing"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// The rule this file exists for: whatever a set is broken into, the till lines
// add up to exactly what the guest paid. A ticket that is one so'm out is a
// shift manager's evening, and the error is invisible until somebody counts.

func TestSplitAmountIsExact(t *testing.T) {
	cases := []struct {
		name    string
		total   int
		weights []int
	}{
		{"even split of an odd total", 40000, []int{1, 1, 1}},
		{"real combo", 40000, []int{28000, 9000, 3000}},
		{"one member", 25000, []int{25000}},
		{"awkward remainder", 100001, []int{7, 11, 13}},
		{"weights that do not divide", 9999, []int{1, 2, 3, 4, 5, 6, 7}},
		{"free set", 0, []int{5000, 3000}},
		{"members priced at zero", 30000, []int{0, 0, 0}},
		{"one member priced at zero", 30000, []int{10000, 0}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := splitAmount(c.total, c.weights)
			if len(got) != len(c.weights) {
				t.Fatalf("got %d shares for %d weights", len(got), len(c.weights))
			}
			sum := 0
			for _, s := range got {
				if s < 0 {
					t.Fatalf("negative share in %v", got)
				}
				sum += s
			}
			if sum != c.total {
				t.Fatalf("shares %v sum to %d, want %d", got, sum, c.total)
			}
		})
	}
}

// The split follows the prices: the dish that costs most carries most of the
// set's price. Without this the discount would land arbitrarily and a cheap
// side could come out dearer than the main course on the kitchen's ticket.
func TestSplitAmountFollowsWeights(t *testing.T) {
	got := splitAmount(40000, []int{28000, 9000, 3000})
	if !(got[0] > got[1] && got[1] > got[2]) {
		t.Fatalf("shares %v do not follow the weights", got)
	}
}

func comboLine(price, qty int, members []comboMember) (models.OrderItem, []comboMember) {
	lines := make([]models.OrderComboLine, 0, len(members))
	for _, m := range members {
		lines = append(lines, models.OrderComboLine{
			Name: m.Name, Qty: m.Qty, MenuItemID: m.MenuItemID, Price: m.Price,
		})
	}
	return models.OrderItem{
		MenuItemID: primitive.NewObjectID(),
		Name:       "Oilaviy combo",
		Price:      price,
		Qty:        qty,
		ComboItems: lines,
	}, members
}

func member(name string, qty, price int) comboMember {
	return comboMember{MenuItemID: primitive.NewObjectID(), Name: name, Qty: qty, Price: price}
}

// A set reaches the till as its dishes, and the lines total what was charged.
func TestExpandComboTotalsTheLinePrice(t *testing.T) {
	cases := []struct {
		name    string
		price   int
		qty     int
		members []comboMember
	}{
		{"three dishes", 40000, 1, []comboMember{
			member("Lag'mon", 1, 28000), member("Achichuk", 1, 9000), member("Choy", 1, 3000),
		}},
		{"two of the same dish", 50000, 1, []comboMember{
			member("Somsa", 3, 12000), member("Choy", 1, 4000),
		}},
		{"ordered three times", 40000, 3, []comboMember{
			member("Lag'mon", 1, 28000), member("Salat", 1, 15000),
		}},
		{"price that will not divide", 33333, 1, []comboMember{
			member("A", 2, 10000), member("B", 1, 10000),
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			line, members := comboLine(c.price, c.qty, c.members)
			out := expandCombo(line, members)
			if len(out) == 0 {
				t.Fatal("no till lines produced")
			}
			total := 0
			portions := map[string]int{}
			for _, l := range out {
				if l.Source.IsZero() {
					t.Fatalf("line %q carries no menu id to map", l.Item.Name)
				}
				if l.Item.Qty < 1 || l.Item.Price < 0 {
					t.Fatalf("nonsense line %+v", l.Item)
				}
				total += l.Item.Price * l.Item.Qty
				portions[l.Item.Name] += l.Item.Qty
			}
			if want := c.price * c.qty; total != want {
				t.Fatalf("till lines total %d, guest paid %d", total, want)
			}
			// Every portion of every member has to be on the ticket: the
			// kitchen cooks what it reads.
			for _, m := range c.members {
				if got, want := portions[m.Name], m.Qty*c.qty; got != want {
					t.Fatalf("%s: %d portions on the ticket, set contains %d", m.Name, got, want)
				}
			}
		})
	}
}

// The combo itself never reaches the till, and every line names a real dish —
// this is the whole reason an order containing a set used to be refused.
func TestExpandComboSendsDishesNotTheSet(t *testing.T) {
	line, members := comboLine(40000, 1, []comboMember{
		member("Lag'mon", 1, 28000), member("Choy", 1, 3000),
	})
	out := expandCombo(line, members)
	ids := map[primitive.ObjectID]bool{}
	for _, m := range members {
		ids[m.MenuItemID] = true
	}
	for _, l := range out {
		if l.Item.Name == line.Name {
			t.Fatalf("the set itself was sent as a line: %+v", l.Item)
		}
		if !ids[l.Source] {
			t.Fatalf("line %q maps to something that is not a member", l.Item.Name)
		}
		// The kitchen has to be able to see the four lines are one order.
		if l.Item.Comment == "" {
			t.Fatalf("line %q does not say which set it came from", l.Item.Name)
		}
	}
}

// A guest's note on the set travels with every dish in it: "piyozsiz" was
// written about something inside.
func TestExpandComboKeepsTheGuestsNote(t *testing.T) {
	line, members := comboLine(40000, 1, []comboMember{member("Lag'mon", 1, 40000)})
	line.Comment = "piyozsiz"
	out := expandCombo(line, members)
	if len(out) != 1 {
		t.Fatalf("want one line, got %d", len(out))
	}
	if got := out[0].Item.Comment; got != "Oilaviy combo · piyozsiz" {
		t.Fatalf("comment is %q", got)
	}
}

// A set whose contents cannot be established produces nothing, so the caller
// refuses the order by name rather than printing half of it.
func TestExpandComboRefusesWhenEmpty(t *testing.T) {
	line, _ := comboLine(40000, 1, nil)
	if out := expandCombo(line, nil); len(out) != 0 {
		t.Fatalf("expanded an unresolvable set into %d lines", len(out))
	}
}
