package posimport

import "testing"

// ⚠️ **The header is not row one.** Every one of these exports puts a report
// title, a date range and the restaurant's name above the table. Taking row one
// as the header names every column after a title, nothing matches, and a file
// that is perfectly readable looks unreadable.
func TestTheHeaderIsFoundBelowTheReportTitle(t *testing.T) {
	csv := "Отчёт по остаткам\n" +
		"Ресторан \"Osh\" за 01.08.2026 - 30.08.2026\n" +
		"\n" +
		"Наименование;Ед. изм.;Цена;Остаток\n" +
		"Говядина;кг;92 000;14,5\n" +
		"Лаваш;шт;3 000;40\n"

	sheet, err := Read("ostatki.csv", []byte(csv))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(sheet.Header) != 4 || sheet.Header[0] != "Наименование" {
		t.Fatalf("header = %q", sheet.Header)
	}
	if len(sheet.Rows) != 2 {
		t.Fatalf("rows = %d, want 2 (the title lines are not data)", len(sheet.Rows))
	}
}

// ⚠️ **The separator is a semicolon on a Russian Windows**, because the comma
// is the decimal mark. With the wrong one every row is a single cell and the
// file appears to have one column.
func TestASemicolonFileIsNotOneColumn(t *testing.T) {
	sheet, err := Read("x.csv", []byte("Название;Ед;Цена\nМасло;л;28 000\n"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(sheet.Header) != 3 {
		t.Fatalf("header = %q — the separator was not detected", sheet.Header)
	}
}

// ⚠️ **Windows-1251, and it is the ordinary case.** These systems run on
// Windows and their CSV export is not UTF-8; read as UTF-8 the header is
// replacement characters, no column is recognised, and the import looks like it
// does not understand the file at all.
func TestALegacyEncodedFileIsStillReadable(t *testing.T) {
	// "Наименование;Ед;Цена\nМасло;л;28000\n" in Windows-1251.
	raw := []byte{
		0xCD, 0xE0, 0xE8, 0xEC, 0xE5, 0xED, 0xEE, 0xE2, 0xE0, 0xED, 0xE8, 0xE5,
		';', 0xC5, 0xE4, ';', 0xD6, 0xE5, 0xED, 0xE0, '\n',
		0xCC, 0xE0, 0xF1, 0xEB, 0xEE, ';', 0xEB, ';', '2', '8', '0', '0', '0', '\n',
	}
	sheet, err := Read("x.csv", raw)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if sheet.Header[0] != "Наименование" {
		t.Fatalf("header[0] = %q — the legacy encoding was not decoded", sheet.Header[0])
	}
	cols := Detect(sheet.Header)
	if _, ok := cols[FieldName]; !ok {
		t.Fatal("the name column was not recognised after decoding")
	}
}

// ⚠️ **Exact heading match, not a substring search.** "Цена" is inside "Цена
// продажи" and inside "Средняя цена закупки", and a substring rule maps all
// three to the same field — after which whichever column was scanned last wins,
// silently, and the food cost is computed from the menu price.
func TestColumnsAreMatchedExactly(t *testing.T) {
	cols := Detect([]string{"Наименование", "Ед. изм.", "Цена", "Цена продажи", "Остаток"})
	if cols[FieldPrice] != 2 {
		t.Fatalf("price column = %d, want 2 — the selling price was picked up", cols[FieldPrice])
	}
	if cols[FieldQty] != 4 {
		t.Fatalf("qty column = %d, want 4", cols[FieldQty])
	}
}

// A tech-card export has both a dish column and an ingredient column, and
// taking the first "name"-like heading would call the dish "name" and leave the
// ingredient nowhere to go.
func TestADishColumnIsNotTheIngredientColumn(t *testing.T) {
	cols := Detect([]string{"Блюдо", "Ингредиент", "Ед", "Брутто"})
	if cols[FieldDish] != 0 {
		t.Fatalf("dish column = %d, want 0", cols[FieldDish])
	}
	if cols[FieldName] != 1 {
		t.Fatalf("ingredient column = %d, want 1", cols[FieldName])
	}
}
