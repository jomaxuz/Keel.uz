package posimport

import (
	"errors"
	"strings"

	"restaurant-backend/internal/models"
)

// ---- Units ----
//
// ⚠️ **This is where an import silently becomes a thousand times wrong.**
//
// Our model has three purchase units — kg, l, pcs — and recipes are written in
// the thousandth of each (see models.PerUnit). The exports do not agree with
// that or with each other: iiko writes `кг` and `гр` as separate units in the
// same column, Poster writes `шт`, Clopos writes `dona`, and half the tech-card
// exports state their quantities in grams while the ingredient row states its
// price per kilo.
//
// Two rules follow, and both are about refusing rather than converting:
//
//  1. **A unit we do not recognise is an error on that line**, never a default.
//     Defaulting to `pcs` is the specific disaster: a kilo of beef becomes one
//     piece, so 180 g of it in a dish costs the price of 180 kilos — or of
//     nothing, depending which way the arithmetic lands. Every number on screen
//     stays plausible.
//  2. **A gram is not a unit, it is a kilo written small.** `гр` in an
//     ingredient's unit column means the list is priced per gram, which no
//     supplier invoice in the country is; it almost always means the exporter
//     wrote the recipe unit into the purchase column. So it maps to kg **and
//     the caller is told**, because the price beside it has to be read as per
//     kilo or the ingredient is a thousand times too cheap.

// ErrUnknownUnit is a unit this import cannot map. Its own error so the panel
// can list the lines and let somebody pick, rather than importing a guess.
var ErrUnknownUnit = errors.New("o'lchov birligi tanilmadi")

// Unit maps whatever the export wrote to one of ours.
//
// `small` reports that the cell named the recipe unit (gram, millilitre) rather
// than the purchase unit — see the note above.
func Unit(raw string) (unit string, small bool, err error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.Trim(s, ".")
	s = strings.ReplaceAll(s, " ", "")
	switch s {
	case "":
		return "", false, ErrUnknownUnit

	// Weight, bought by the kilo.
	case "кг", "kg", "килограмм", "килограммы", "kilogramm", "kilo", "кг.":
		return models.UnitKg, false, nil
	case "г", "гр", "грамм", "граммы", "g", "gr", "gramm", "gram":
		return models.UnitKg, true, nil

	// Volume, bought by the litre.
	case "л", "l", "литр", "литры", "litr", "liter", "litre":
		return models.UnitL, false, nil
	case "мл", "ml", "миллилитр", "millilitr":
		return models.UnitL, true, nil

	// Counted.
	case "шт", "штука", "штуки", "pcs", "pc", "dona", "ta", "порц", "порция",
		"порции", "porsiya", "portion", "уп", "упак", "упаковка", "paket",
		"бут", "бутылка", "banka", "банка":
		// ⚠️ A packet and a bottle are counted things, and counting them is
		// right — but only the buyer knows what is inside one. That is the same
		// decision the ingredient model already made by refusing free-text
		// units; here it means the import does not try to open the packet.
		return models.UnitPcs, false, nil
	}
	return "", false, ErrUnknownUnit
}

// RecipeQty converts a quantity written in the export's unit into ours.
//
// ⚠️ **Recipes are stored in the small unit** (grams, millilitres, pieces) and
// the exports are inconsistent about which they used. A tech card line saying
// `0,18` next to `кг` and one saying `180` next to `гр` are the same 180 grams,
// and storing either of them unconverted is out by a factor of a thousand — in
// opposite directions, so one dish looks free and the next looks ruinous.
func RecipeQty(qty float64, rawUnit string) (float64, error) {
	unit, small, err := Unit(rawUnit)
	if err != nil {
		return 0, err
	}
	if small {
		return qty, nil
	}
	return qty * float64(models.PerUnit(unit)), nil
}
