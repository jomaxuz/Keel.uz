package posimport

import (
	"fmt"
	"strings"
)

// Turning rows into the three things a restaurant is actually moving.

// Kind is what the file holds.
const (
	// The ingredient list: what the kitchen buys, in what unit, at what price.
	KindIngredients = "ingredients"
	// Tech cards: which ingredients go into which dish, and how much.
	KindRecipes = "recipes"
	// A stocktake: what is on the shelf today.
	KindStock = "stock"
)

// Line is one proposed row, with whatever went wrong with it attached.
//
// ⚠️ **A bad line is carried, not dropped.** An import that quietly skips the
// forty lines it could not read produces a list that looks complete and is not,
// and the missing forty are found weeks later as dishes with no cost. Every
// line comes back; the ones with a problem are shown, sorted to the top, and
// cannot be ticked until they are fixed.
type Line struct {
	Row int `json:"row"`

	Name     string  `json:"name"`
	Unit     string  `json:"unit,omitempty"`
	Price    int     `json:"price,omitempty"`
	Qty      float64 `json:"qty,omitempty"`
	Category string  `json:"category,omitempty"`
	Dish     string  `json:"dish,omitempty"`
	Note     string  `json:"note,omitempty"`

	// ⚠️ Set when the export named the recipe unit in the purchase column
	// (`гр`, `мл`). The price beside it then has to be read as per kilo or
	// litre, and saying so is the difference between an ingredient that costs
	// 42 000 and one that costs 42.
	SmallUnit bool `json:"smallUnit,omitempty"`

	// What is wrong with this line, in the owner's language. Empty when it is
	// fine.
	Problem string `json:"problem,omitempty"`
	// Already in the catalogue under this name.
	Exists bool `json:"exists,omitempty"`
}

// OK reports whether the line can be imported as it stands.
func (l Line) OK() bool { return l.Problem == "" && strings.TrimSpace(l.Name) != "" }

// Build turns a sheet into proposed lines for one kind of import.
func Build(kind string, s *Sheet, cols map[string]int) []Line {
	out := make([]Line, 0, len(s.Rows))
	for i, row := range s.Rows {
		l := Line{
			Row:      i + 1,
			Name:     Cell(row, idx(cols, FieldName)),
			Category: Cell(row, idx(cols, FieldCategory)),
			Dish:     Cell(row, idx(cols, FieldDish)),
			Note:     Cell(row, idx(cols, FieldNote)),
		}
		if l.Name == "" {
			// ⚠️ A nameless row is a subtotal, a section heading or the blank
			// line before one. Skipped rather than reported: an import that
			// lists forty "problems" that are all the export's own formatting
			// teaches people to ignore the problem column.
			continue
		}

		rawUnit := Cell(row, idx(cols, FieldUnit))
		if rawUnit != "" {
			unit, small, err := Unit(rawUnit)
			if err != nil {
				l.Problem = fmt.Sprintf("o'lchov birligi tanilmadi: %q", rawUnit)
			} else {
				l.Unit, l.SmallUnit = unit, small
			}
		} else if kind != KindRecipes {
			// A recipe line can borrow the ingredient's unit; an ingredient
			// cannot borrow from anywhere.
			l.Problem = "o'lchov birligi yo'q"
		}

		if raw := Cell(row, idx(cols, FieldPrice)); raw != "" {
			if p, err := ParseMoney(raw); err != nil {
				l.Problem = fmt.Sprintf("narxni o'qib bo'lmadi: %q", raw)
			} else {
				l.Price = p
			}
		}

		if raw := Cell(row, idx(cols, FieldQty)); raw != "" {
			q, err := ParseQty(raw)
			if err != nil {
				l.Problem = fmt.Sprintf("miqdorni o'qib bo'lmadi: %q", raw)
			} else {
				l.Qty = q
			}
		} else if kind == KindRecipes || kind == KindStock {
			// ⚠️ **Zero is not an acceptable recovery here.** A tech card line
			// at nothing makes the dish free, and a stock line at nothing is a
			// shortfall of the entire shelf on the first count.
			l.Problem = "miqdor ustuni bo'sh"
		}

		if kind == KindRecipes && l.Dish == "" {
			l.Problem = "qaysi taomga tegishli ekani ko'rsatilmagan"
		}
		out = append(out, l)
	}
	return sortProblemsFirst(out)
}

func idx(cols map[string]int, field string) int {
	if i, ok := cols[field]; ok {
		return i
	}
	return -1
}

// sortProblemsFirst puts the lines that need attention where they are seen.
//
// ⚠️ Stable within each group, so the rest of the list stays in the file's own
// order — somebody checking an import against the sheet on their other screen
// is comparing two lists, and reordering the good half makes that impossible.
func sortProblemsFirst(in []Line) []Line {
	bad := make([]Line, 0, 8)
	good := make([]Line, 0, len(in))
	for _, l := range in {
		if l.Problem != "" {
			bad = append(bad, l)
		} else {
			good = append(good, l)
		}
	}
	return append(bad, good...)
}
