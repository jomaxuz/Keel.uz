package posimport

import "strings"

// Which column is which.
//
// ⚠️ **Detected and then shown, never detected and used.** Five systems, each
// with its own headings, in Russian, Uzbek and sometimes English, and every one
// of them free to rename a column in the next release. A parser hard-coded per
// system works until the day it does not, and the failure is an import that
// reads the wrong column — a price column read as a quantity is a tech card
// nobody can tell is wrong by looking at it.
//
// So the guess is a starting point the owner can correct on screen, and a
// column nothing matched stays unmapped rather than being assigned to whatever
// was left over.

// Field names this import understands.
const (
	FieldName     = "name"
	FieldUnit     = "unit"
	FieldPrice    = "price"
	FieldQty      = "qty"
	FieldCategory = "category"
	FieldDish     = "dish"
	FieldNote     = "note"
	FieldCode     = "code"
)

// headings maps a normalised heading to a field. Ordered by how specific the
// phrase is — see fieldOf.
var headings = map[string]string{
	// The dish a tech card belongs to. ⚠️ Listed before the plain "name"
	// entries: a tech-card export has both, and taking the first match would
	// call the dish column "name" and then have nowhere to put the ingredient.
	"блюдо": FieldDish, "товар/блюдо": FieldDish, "продукт/блюдо": FieldDish,
	"техкарта": FieldDish, "технологическаякарта": FieldDish,
	"наименованиеблюда": FieldDish, "dish": FieldDish, "taom": FieldDish,

	"ингредиент": FieldName, "ингредиенты": FieldName, "сырье": FieldName,
	"наименование": FieldName, "название": FieldName, "номенклатура": FieldName,
	"продукт": FieldName, "товар": FieldName, "name": FieldName,
	"nomi": FieldName, "nom": FieldName, "mahsulot": FieldName,
	"masalliq": FieldName, "ingredient": FieldName, "item": FieldName,

	"единица": FieldUnit, "единицаизмерения": FieldUnit, "ед": FieldUnit,
	"едизм": FieldUnit, "ед.изм": FieldUnit, "unit": FieldUnit,
	"birlik": FieldUnit, "o'lchov": FieldUnit, "olchov": FieldUnit,
	"o'lchovbirligi": FieldUnit, "измерение": FieldUnit,

	"цена": FieldPrice, "стоимость": FieldPrice, "себестоимость": FieldPrice,
	"ценазаединицу": FieldPrice, "средняяцена": FieldPrice, "price": FieldPrice,
	"narx": FieldPrice, "narxi": FieldPrice, "tannarx": FieldPrice,
	"cost": FieldPrice,

	"количество": FieldQty, "кол-во": FieldQty, "колво": FieldQty,
	"остаток": FieldQty, "остатки": FieldQty, "фактическийостаток": FieldQty,
	"факт": FieldQty, "брутто": FieldQty, "нормазакладки": FieldQty,
	"расход": FieldQty, "qty": FieldQty, "quantity": FieldQty,
	"miqdor": FieldQty, "miqdori": FieldQty, "qoldiq": FieldQty,
	"sarf": FieldQty, "balance": FieldQty, "stock": FieldQty,

	"категория": FieldCategory, "группа": FieldCategory, "раздел": FieldCategory,
	"category": FieldCategory, "kategoriya": FieldCategory,
	"bolim": FieldCategory, "guruh": FieldCategory,

	"код": FieldCode, "артикул": FieldCode, "code": FieldCode, "kod": FieldCode,

	"комментарий": FieldNote, "примечание": FieldNote, "note": FieldNote,
	"izoh": FieldNote,
}

// fieldOf recognises one heading.
//
// ⚠️ **Exact match on the normalised text, not a substring search.** "Цена" is
// inside "Цена продажи" and inside "Средняя цена закупки", and a substring rule
// maps all three to the same field — after which whichever column happened to
// be scanned last wins, silently, and the food cost is computed from the menu
// price. Anything not on the list is left for the owner to point at.
func fieldOf(heading string) string {
	return headings[normalHeading(heading)]
}

func normalHeading(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch r {
		case ' ', '\t', '\n', '\r', '_', '-', '(', ')', '№', '*', ':':
			continue
		case 'ʻ', 'ʼ', '‘', '’', '`':
			b.WriteRune('\'')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Detect guesses a column for each field.
//
// ⚠️ **First match wins per field, and a column is used once.** A stock export
// has "Остаток" and "Остаток на начало"; both are quantities, and letting the
// second overwrite the first imports the opening balance as the count.
func Detect(header []string) map[string]int {
	out := map[string]int{}
	used := map[int]bool{}
	for i, h := range header {
		f := fieldOf(h)
		if f == "" || used[i] {
			continue
		}
		if _, taken := out[f]; taken {
			continue
		}
		out[f] = i
		used[i] = true
	}
	return out
}

// Cell reads one column of a row, tolerating a short row.
//
// ⚠️ Short rows are the norm, not an error: a spreadsheet stops writing cells
// at the last non-empty one, so a row whose note is blank is simply shorter.
// Indexing it directly is a panic on somebody's real file.
func Cell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}
