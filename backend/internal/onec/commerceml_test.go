package onec

import (
	"strings"
	"testing"
	"time"
)

// ⚠️ **The element names are 1C's, in Russian, and they are matched
// literally.** A transliterated or "tidied" name is not an error on either
// side: 1C reads the file, finds nothing it recognises, imports nothing and
// reports success. The only symptom is an accountant saying the sales did not
// come through, with no log anywhere to look at — which is why the names are
// pinned here rather than trusted to review.
func TestTheElementNamesAreTheOnes1CLooksFor(t *testing.T) {
	out := string(Documents([]Doc{{
		ID: "abc", Operation: OpSale, Number: "A-1",
		At: time.Date(2026, 9, 7, 13, 5, 0, 0, time.UTC), Total: 120000,
		Lines: []Line{{ID: "x", Name: "Lag'mon", Unit: "pcs", Qty: 2, Price: 60000, Sum: 120000}},
	}}))
	for _, want := range []string{
		"<КоммерческаяИнформация", "<Документ>", "<Ид>", "<Номер>", "<Дата>",
		"<ХозОперация>", "<Роль>", "<Валюта>", "<Сумма>", "<Товары>", "<Товар>",
		"<БазоваяЕдиница", "<ЦенаЗаЕдиницу>", "<Количество>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("%s is missing — 1C would import nothing and report success", want)
		}
	}
	if !strings.Contains(out, `encoding="UTF-8"`) {
		t.Error("the encoding declaration is missing or not UTF-8")
	}
}

// ⚠️ **"Роль" is written from the exchange's point of view.** In a sale we are
// the seller; in a delivery the supplier is, and we are the buyer. Getting it
// backwards posts a month of takings into the accountant's purchases — every
// number present, every number on the wrong side.
func TestTheRoleSaysWhichSideOfTheBooksThisIs(t *testing.T) {
	sale := string(Documents([]Doc{{ID: "1", Operation: OpSale, At: time.Now()}}))
	if !strings.Contains(sale, "<Роль>Продавец</Роль>") {
		t.Error("a sale is not written as ours to sell")
	}
	buy := string(Documents([]Doc{{ID: "2", Operation: OpPurchase, At: time.Now()}}))
	if !strings.Contains(buy, "<Роль>Покупатель</Роль>") {
		t.Error("a delivery is not written as ours to buy")
	}
}

// ⚠️ **A dish called `Salat "Sezar" & co` is a dish, not markup** — the rule
// the receipt printer and the delivery note already follow. An unescaped
// ampersand makes 1C reject the entire file, and what the accountant is shown
// is nothing at all.
func TestNamesAreWrittenAsTextRatherThanAsMarkup(t *testing.T) {
	out := string(Documents([]Doc{{
		ID: "1", Operation: OpSale, At: time.Now(),
		Lines: []Line{{ID: "x", Name: `Salat "Sezar" & <b>co</b>`, Qty: 1}},
	}}))
	if strings.Contains(out, "<b>") {
		t.Fatalf("markup reached the file: %s", out)
	}
	if !strings.Contains(out, "&amp;") {
		t.Fatalf("an ampersand was not escaped: %s", out)
	}
}

// ⚠️ **Two files, one product.** 1C splits the nomenclature from the prices and
// joins them by `Ид`, which may carry a characteristic after a `#`. A parser
// that kept the whole string would create a second product for every size of
// the same thing — and each of them would get its own shelf balance.
func TestNomenclatureAndPricesJoinOnOneProduct(t *testing.T) {
	items, err := Parse([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<КоммерческаяИнформация ВерсияСхемы="2.05">
  <Каталог>
    <Товары>
      <Товар><Ид>111</Ид><Наименование>Un</Наименование><БазоваяЕдиница>кг</БазоваяЕдиница></Товар>
      <Товар><Ид>222#L</Ид><Наименование>Yog'</Наименование><БазоваяЕдиница>л</БазоваяЕдиница></Товар>
    </Товары>
  </Каталог>
  <ПакетПредложений>
    <Предложения>
      <Предложение><Ид>111</Ид><Цены><Цена><ЦенаЗаЕдиницу>5200</ЦенаЗаЕдиницу></Цена>
        <Цена><ЦенаЗаЕдиницу>9900</ЦенаЗаЕдиницу></Цена></Цены></Предложение>
      <Предложение><Ид>222#L</Ид><Цены><Цена><ЦенаЗаЕдиницу>24000</ЦенаЗаЕдиницу></Цена></Цены></Предложение>
    </Предложения>
  </ПакетПредложений>
</КоммерческаяИнформация>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d products, want 2: %+v", len(items), items)
	}
	if items[0].ID != "111" || items[0].Name != "Un" || items[0].Unit != "кг" {
		t.Errorf("the nomenclature did not survive: %+v", items[0])
	}
	// ⚠️ The **first** price, not the largest: an offer carries retail,
	// wholesale and purchase prices in the order the accountant configured, and
	// picking any other one prices the shelf from somebody else's column.
	if items[0].Price != 5200 {
		t.Errorf("price = %v, want the first one (5200)", items[0].Price)
	}
	if items[1].ID != "222" {
		t.Errorf("a characteristic split one product in two: %+v", items[1])
	}
}

// A file we cannot read is an error, not an empty catalogue: "the accountant
// sent nothing" and "we could not read what the accountant sent" send two
// different people to look.
func TestAnUnreadableFileIsNotAnEmptyCatalogue(t *testing.T) {
	if _, err := Parse([]byte("<not xml")); err == nil {
		t.Error("a broken file was read as an empty catalogue")
	}
}
