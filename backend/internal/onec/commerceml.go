// Package onec speaks the exchange protocol every 1C configuration already has.
//
// ⚠️ **1C always knocks; we never call it.** The accountant's 1C sits on an
// office machine behind somebody's router, usually with no address of its own —
// which is exactly why the published protocol is built the way it is: a
// sequence of plain HTTP requests that 1C initiates, with plain-text answers
// (`success`, `progress`, `failure`). A design that tried to push into 1C would
// demonstrate beautifully and work in no real office.
//
// ⚠️ **The XML element names are Russian and are matched literally.** 1C reads
// them by name; one letter wrong and the document is skipped in silence — no
// error on their side, no error on ours, just an accountant who says "the sales
// did not come through" with nothing to look at. So the names here are copied
// from the standard rather than transliterated, and the test pins them.
//
// Read copy of the protocol: docs/vendor/1c-exchange.md.
package onec

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Doc is one document as 1C will read it: a day's takings, or a delivery.
type Doc struct {
	// Our own id for the document. ⚠️ Stable across exchanges: 1C matches on it
	// to decide "have I already got this one", and an id that changed between
	// two runs would file every sale twice.
	ID string
	// "Заказ товара" for a sale, "Поступление товаров" for a delivery — 1C
	// dispatches on this string.
	Operation string
	Number    string
	At        time.Time
	Total     float64
	// Whom it was with: a guest has no name and none is invented.
	Counterparty string
	CounterTIN   string
	Comment      string
	Lines        []Line
}

// Line is one row of a document.
type Line struct {
	ID    string
	Name  string
	Unit  string
	Qty   float64
	Price float64
	Sum   float64
}

// Product is one row of the catalogue 1C sends us, or that we send it.
type Product struct {
	ID      string
	Name    string
	Unit    string
	Barcode string
	Price   float64
	Group   string
}

const (
	// OpSale is a sale, as 1C's own list of operations names it.
	OpSale = "Заказ товара"
	// OpPurchase is a delivery from a supplier.
	OpPurchase = "Поступление товаров"
)

// header is the one line that has to be right before anything else can be.
//
// ⚠️ **windows-1251 is not offered, and that is deliberate.** Older exchanges
// used it and some examples still show it; every current configuration reads
// UTF-8, and supporting both would mean guessing which one a file we did not
// write was in — a guess that produces a catalogue of question marks rather
// than an error.
const header = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"

// Documents renders a set of documents as CommerceML 2.
func Documents(docs []Doc) []byte {
	var b strings.Builder
	b.WriteString(header)
	b.WriteString(`<КоммерческаяИнформация ВерсияСхемы="2.05" ДатаФормирования="`)
	b.WriteString(time.Now().Format("2006-01-02"))
	b.WriteString("\">\n")
	for _, d := range docs {
		writeDoc(&b, d)
	}
	b.WriteString("</КоммерческаяИнформация>\n")
	return []byte(b.String())
}

func writeDoc(b *strings.Builder, d Doc) {
	b.WriteString("  <Документ>\n")
	tag(b, 4, "Ид", d.ID)
	tag(b, 4, "Номер", d.Number)
	tag(b, 4, "Дата", d.At.Format("2006-01-02"))
	tag(b, 4, "ХозОперация", d.Operation)
	// ⚠️ **"Продавец" is written from the exchange's point of view, not ours.**
	// In a sale we are the seller; in a delivery the supplier is. 1C uses this
	// to decide which side of its own books the document lands on, and getting
	// it backwards posts a month of takings as purchases.
	role := "Продавец"
	if d.Operation == OpPurchase {
		role = "Покупатель"
	}
	tag(b, 4, "Роль", role)
	tag(b, 4, "Валюта", "UZS")
	tag(b, 4, "Курс", "1")
	tag(b, 4, "Сумма", money(d.Total))
	tag(b, 4, "Время", d.At.Format("15:04:05"))
	if d.Counterparty != "" {
		b.WriteString("    <Контрагенты>\n      <Контрагент>\n")
		tag(b, 8, "Ид", firstNonEmpty(d.CounterTIN, d.Counterparty))
		tag(b, 8, "Наименование", d.Counterparty)
		if d.CounterTIN != "" {
			tag(b, 8, "ИНН", d.CounterTIN)
		}
		tag(b, 8, "Роль", map[string]string{OpPurchase: "Продавец"}[d.Operation])
		b.WriteString("      </Контрагент>\n    </Контрагенты>\n")
	}
	b.WriteString("    <Товары>\n")
	for _, l := range d.Lines {
		b.WriteString("      <Товар>\n")
		tag(b, 8, "Ид", l.ID)
		tag(b, 8, "Наименование", l.Name)
		b.WriteString(`        <БазоваяЕдиница Код="796" НаименованиеПолное="`)
		b.WriteString(escape(unitName(l.Unit)))
		b.WriteString(`">`)
		b.WriteString(escape(l.Unit))
		b.WriteString("</БазоваяЕдиница>\n")
		tag(b, 8, "ЦенаЗаЕдиницу", money(l.Price))
		tag(b, 8, "Количество", qty(l.Qty))
		tag(b, 8, "Сумма", money(l.Sum))
		b.WriteString("      </Товар>\n")
	}
	b.WriteString("    </Товары>\n")
	if d.Comment != "" {
		tag(b, 4, "Комментарий", d.Comment)
	}
	b.WriteString("  </Документ>\n")
}

// Catalogue renders products as an `import.xml` 1C can read.
//
// ⚠️ **Sent as well as received, because an accountant's first question is the
// other direction.** A restaurant that keeps its recipes and its shelf here
// wants that list *in* 1C once, not typed twice — and the same file is what
// arrives from them when the nomenclature already lives in 1C.
func Catalogue(name string, items []Product) []byte {
	var b strings.Builder
	b.WriteString(header)
	b.WriteString(`<КоммерческаяИнформация ВерсияСхемы="2.05" ДатаФормирования="`)
	b.WriteString(time.Now().Format("2006-01-02"))
	b.WriteString("\">\n  <Каталог СодержитТолькоИзменения=\"false\">\n")
	tag(&b, 4, "Ид", "keel-catalog")
	tag(&b, 4, "Наименование", name)
	b.WriteString("    <Товары>\n")
	for _, p := range items {
		b.WriteString("      <Товар>\n")
		tag(&b, 8, "Ид", p.ID)
		tag(&b, 8, "Наименование", p.Name)
		if p.Barcode != "" {
			tag(&b, 8, "Штрихкод", p.Barcode)
		}
		b.WriteString(`        <БазоваяЕдиница Код="796" НаименованиеПолное="`)
		b.WriteString(escape(unitName(p.Unit)))
		b.WriteString(`">`)
		b.WriteString(escape(p.Unit))
		b.WriteString("</БазоваяЕдиница>\n")
		if p.Group != "" {
			b.WriteString("        <Группы>\n")
			tag(&b, 10, "Ид", p.Group)
			b.WriteString("        </Группы>\n")
		}
		b.WriteString("      </Товар>\n")
	}
	b.WriteString("    </Товары>\n  </Каталог>\n</КоммерческаяИнформация>\n")
	return []byte(b.String())
}

// ---- Reading what 1C sends ----

// importFile is the part of `import.xml` we read: the goods and their names.
//
// ⚠️ **Only what we can act on.** A 1C catalogue carries property values,
// pictures, groups, characteristics and a dozen other branches; a parser that
// insisted on understanding them would refuse a file it merely does not need
// all of, and the accountant would be told the import failed because of a field
// nobody uses.
type importFile struct {
	XMLName xml.Name `xml:"КоммерческаяИнформация"`
	Catalog struct {
		Products []struct {
			ID      string `xml:"Ид"`
			Name    string `xml:"Наименование"`
			Barcode string `xml:"Штрихкод"`
			Unit    string `xml:"БазоваяЕдиница"`
		} `xml:"Товары>Товар"`
	} `xml:"Каталог"`
	Offers struct {
		Offers []struct {
			ID    string `xml:"Ид"`
			Name  string `xml:"Наименование"`
			Price []struct {
				Value float64 `xml:"ЦенаЗаЕдиницу"`
			} `xml:"Цены>Цена"`
			Qty float64 `xml:"Количество"`
		} `xml:"Предложения>Предложение"`
	} `xml:"ПакетПредложений"`
}

// Parse reads a catalogue or an offers file into products.
//
// ⚠️ **One id, two files.** 1C splits the nomenclature (`import.xml`) from the
// prices (`offers.xml`) and joins them by `Ид` — which may carry a
// characteristic after a `#`. The prefix is what identifies the product, and a
// parser that kept the whole string would create a second product for every
// size of the same thing.
func Parse(raw []byte) ([]Product, error) {
	var f importFile
	if err := xml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("1C: faylni o'qib bo'lmadi: %w", err)
	}
	byID := map[string]*Product{}
	order := []string{}
	for _, p := range f.Catalog.Products {
		id := baseID(p.ID)
		if id == "" {
			continue
		}
		if _, seen := byID[id]; !seen {
			order = append(order, id)
			byID[id] = &Product{ID: id}
		}
		byID[id].Name = strings.TrimSpace(p.Name)
		byID[id].Barcode = strings.TrimSpace(p.Barcode)
		byID[id].Unit = strings.TrimSpace(p.Unit)
	}
	for _, o := range f.Offers.Offers {
		id := baseID(o.ID)
		if id == "" {
			continue
		}
		if _, seen := byID[id]; !seen {
			order = append(order, id)
			byID[id] = &Product{ID: id, Name: strings.TrimSpace(o.Name)}
		}
		if byID[id].Name == "" {
			byID[id].Name = strings.TrimSpace(o.Name)
		}
		// ⚠️ **The first price, not the largest or the last.** A product can
		// carry a retail price, a wholesale price and a purchase price in one
		// offer; the exchange lists them in the order the accountant configured,
		// and picking any other one silently prices the shelf from somebody
		// else's column.
		if len(o.Price) > 0 && byID[id].Price == 0 {
			byID[id].Price = o.Price[0].Value
		}
	}
	out := make([]Product, 0, len(order))
	for _, id := range order {
		if byID[id].Name == "" {
			continue
		}
		out = append(out, *byID[id])
	}
	return out, nil
}

func baseID(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '#'); i > 0 {
		s = s[:i]
	}
	return s
}

// ---- small helpers ----

func tag(b *strings.Builder, indent int, name, value string) {
	if value == "" {
		return
	}
	b.WriteString(strings.Repeat(" ", indent))
	b.WriteString("<" + name + ">")
	b.WriteString(escape(value))
	b.WriteString("</" + name + ">\n")
}

// escape is the one place text from the restaurant meets markup.
//
// ⚠️ A dish called `Salat "Sezar" & co` is a dish, not markup — the same rule
// the receipt printer and the delivery note follow. An unescaped ampersand
// makes 1C reject the whole file, which is reported to the accountant as
// nothing at all.
func escape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// money writes an amount the way the standard does: a dot, two decimals.
func money(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }

// qty writes a quantity without trailing noise.
func qty(v float64) string {
	s := strconv.FormatFloat(v, 'f', 3, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// unitName spells the unit out, because 1C prints the full name and matches on
// the short one.
func unitName(u string) string {
	switch strings.ToLower(strings.TrimSpace(u)) {
	case "kg", "кг":
		return "Килограмм"
	case "l", "л":
		return "Литр"
	case "g", "г":
		return "Грамм"
	case "m", "м":
		return "Метр"
	case "", "pcs", "dona", "шт":
		return "Штука"
	}
	return u
}
