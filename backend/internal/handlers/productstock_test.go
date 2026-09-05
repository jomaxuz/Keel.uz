package handlers

import (
	"strings"
	"testing"

	"restaurant-backend/internal/models"
)

// ⚠️ **Everything sealed here fails silently.** A product that sells and never
// depletes, a second stock row that strands a counted balance, a shelf price
// written into the purchase price — none of them raise anything. They show up as
// a number somebody eventually disbelieves, weeks later.

func TestADishIsUntouched(t *testing.T) {
	// The whole promise to the twelve restaurants already running: this feature
	// does not exist for them.
	h := &Handler{}
	m := models.MenuItem{Name: "Lag'mon"}
	if err := h.syncProductStock(t.Context(), &m); err != nil {
		t.Fatal(err)
	}
	if !m.StockID.IsZero() || len(m.Recipe) != 0 {
		t.Fatal("a dish was given a stock row or a tech card")
	}
}

// ⚠️ The stock unit and the fiscal unit are two vocabularies for one fact, and
// only one of them divides a purchase. A code the classifier adds later must
// land on "piece" rather than on an empty string, which `PerUnit` would treat as
// one and a purchase screen would show as blank.
func TestEveryMeasureCodeLandsOnAStockUnit(t *testing.T) {
	cases := map[int]string{
		0:  models.UnitPcs, // piece — the default and almost everything
		10: models.UnitKg,  // gram
		11: models.UnitKg,  // kilogram
		41: models.UnitL,   // litre
		22: models.UnitPcs, // metre — cloth, sold by the piece in the store
		99: models.UnitPcs, // something the classifier adds after this build
	}
	for code, want := range cases {
		m := &models.MenuItem{UnitCode: code}
		if got := stockUnitOf(m); got != want {
			t.Fatalf("unitCode %d -> %q, want %q", code, got, want)
		}
	}
}

// ⚠️ **A shop product must not carry the shelf price into the store.**
// `Ingredient.Price` is what the goods cost us; seeding it from what we charge
// makes the first stock valuation and every margin read off it wrong — and
// wrong in the flattering direction, which is the wrong way to be wrong about
// money.
func TestTheStockRowIsNotSeededWithTheSalePrice(t *testing.T) {
	src := readSrc(t, "productstock.go")
	body := between(t, src, "func (h *Handler) syncProductStock", "\n}\n")
	if strings.Contains(body, "m.Price") {
		t.Fatal("the sale price reaches the stock row")
	}
	if !strings.Contains(body, "Price:     0") && !strings.Contains(body, "Price: 0") {
		t.Fatal("the purchase price is not explicitly started at zero")
	}
}

// ⚠️ **Turning the flag off must not delete the row.** It can carry purchases,
// write-offs and a counted balance; deleting it would take a real quantity of
// real goods out of the books to tidy up a checkbox.
func TestClearingTheFlagDoesNotDeleteStock(t *testing.T) {
	src := readSrc(t, "productstock.go")
	body := between(t, src, "func (h *Handler) syncProductStock", "\n}\n")
	if strings.Contains(body, "DeleteOne") || strings.Contains(body, "DeleteMany") {
		t.Fatal("the stock row is deleted when the product stops selling itself")
	}
}

// ⚠️ **The link has to survive a whole-document save.** `UpdateMenuItem` is a
// ReplaceOne and the panel never sends `stockId`; without this a price change
// mints a second stock row and strands the balance on the first.
func TestTheStockLinkSurvivesAPriceChange(t *testing.T) {
	src := readSrc(t, "admin.go")
	body := between(t, src, "func (h *Handler) UpdateMenuItem", "\n}\n")
	keep := strings.Index(body, "keepStockID")
	replace := strings.Index(body, "h.Store.Menu.ReplaceOne")
	if keep < 0 {
		t.Fatal("the stock link is not carried across the replace")
	}
	if replace < 0 || keep > replace {
		t.Fatal("the stock link is restored after the document was already written")
	}
}

// ⚠️ **The tech card is written before the insert, not after.** Run afterwards
// it needs a second update, and a crash between the two leaves a product that
// sells and never depletes — stock that drifts upward, quietly, from day one.
func TestTheCardIsWrittenBeforeTheProductIsStored(t *testing.T) {
	src := readSrc(t, "admin.go")
	body := between(t, src, "func (h *Handler) CreateMenuItem", "\n}\n")
	sync := strings.Index(body, "h.syncProductStock")
	insert := strings.Index(body, "h.Store.Menu.InsertOne")
	if sync < 0 || insert < 0 || sync > insert {
		t.Fatal("the stock row is created after the product is inserted")
	}
}
