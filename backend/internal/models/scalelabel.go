package models

import (
	"strconv"
	"strings"
)

// ---- Reading a barcode a scale printed ----
//
// ⚠️ **This is the one piece of a shop's counter that gets money wrong in
// silence.** A scale prints a label whose barcode carries the weight or the
// price inside it, and every scale vendor lays those digits out differently. Read
// the layout wrong and the counter still beeps, still shows a product, still
// prints a receipt — with the wrong quantity on it. Nobody notices until a
// stocktake, and by then it is thousands of sales.
//
// So the layout is **configured per branch, not guessed**: the shop's scale is
// set up by whoever installed it, and the only person who knows how is standing
// in that shop. The defaults below are the most common arrangement here, and the
// settings screen shows what a real label decodes to before it is saved — a
// number somebody can check against the sticker in their hand.
//
// ⚠️ **The embedded value is weight *or* price, and the two are not
// interchangeable.** A scale set to print price sends money; read as grams it
// becomes a quantity in the thousands. That is the failure that empties a shelf
// on paper, so it is an explicit choice with no default that silently fits both.

// Scale label value kinds.
const (
	// ScaleWeight means the digits are grams.
	ScaleWeight = "weight"
	// ScalePrice means the digits are money, in whole som.
	ScalePrice = "price"
)

// ScaleLabel is how this branch's scales lay out a printed barcode.
type ScaleLabel struct {
	// Whether the counter should try to read scale labels at all.
	//
	// ⚠️ **Off by default, and that is not caution for its own sake.** A
	// restaurant never prints one; a shop with no scales never sees one; and a
	// prefix that matched an ordinary EAN would turn a normal product into a
	// weighed one — five digits of its barcode read as a price.
	Enabled bool `bson:"enabled" json:"enabled"`

	// What the first digits of a scale label are.
	//
	// ⚠️ **"2" is not a guess: GS1 reserves the 2x prefixes for in-store and
	// variable-measure items**, which is why an ordinary retail product never
	// begins with one. That reservation is most of what stops a normal barcode
	// being read as a weight — the rest is the exact length below.
	Prefix string `bson:"prefix,omitempty" json:"prefix,omitempty"`

	// How many digits after the prefix name the product.
	ItemLen int `bson:"itemLen,omitempty" json:"itemLen,omitempty"`

	// How many digits carry the value, and what the value is.
	ValueLen int    `bson:"valueLen,omitempty" json:"valueLen,omitempty"`
	Value    string `bson:"value,omitempty" json:"value,omitempty"`

	// The serial port a counter scale is wired to, when there is one.
	//
	// ⚠️ **Independent of everything above.** Those settings describe a label a
	// scale *prints*; this is a scale the till *reads*. A shop may have one, the
	// other, or both, and treating either as implying the other puts a control
	// in front of a cashier that cannot work.
	Port string `bson:"port,omitempty" json:"port,omitempty"`
}

// ScaleRead is what a label decoded to.
type ScaleRead struct {
	// The product's own code, as the scale was programmed with it.
	ItemCode string
	// Kilograms, when the label carried a weight.
	Kg float64
	// Whole som, when the label carried a price.
	Price int
}

// Defaults fills in the arrangement most scales here are shipped with.
//
// ⚠️ Applied on read rather than written into the document: a branch that has
// never opened the settings screen still decodes a label correctly, and a branch
// that has chosen something keeps its choice.
func (s ScaleLabel) Defaults() ScaleLabel {
	if s.Prefix == "" {
		s.Prefix = "2"
	}
	if s.ItemLen == 0 {
		// ⚠️ **Six, so prefix + item + value + check is thirteen.** A scale
		// prints an EAN-13 and nothing else; a default that added up to twelve
		// decoded no real label at all, which the test caught before any shop
		// did. 1 + 6 + 5 + 1 = 13.
		s.ItemLen = 6
	}
	if s.ValueLen == 0 {
		s.ValueLen = 5
	}
	if s.Value == "" {
		s.Value = ScaleWeight
	}
	return s
}

// Read decodes a scanned code, or reports that this is not a scale label.
//
// ⚠️ **Anything it is not certain about, it declines.** A wrong "no" costs one
// manual entry; a wrong "yes" charges for a quantity nobody weighed. So the
// prefix has to match, the length has to be exactly right, and every digit has
// to be a digit — a label one character short is a misread by the scanner, not
// a product.
func (s ScaleLabel) Read(code string) (ScaleRead, bool) {
	s = s.Defaults()
	if !s.Enabled {
		return ScaleRead{}, false
	}
	code = strings.TrimSpace(code)
	if !strings.HasPrefix(code, s.Prefix) {
		return ScaleRead{}, false
	}
	// Prefix + item + value + one check digit, which the scanner has already
	// verified and which carries nothing we need.
	want := len(s.Prefix) + s.ItemLen + s.ValueLen + 1
	if len(code) != want {
		return ScaleRead{}, false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return ScaleRead{}, false
		}
	}

	at := len(s.Prefix)
	item := code[at : at+s.ItemLen]
	raw := code[at+s.ItemLen : at+s.ItemLen+s.ValueLen]
	n, err := strconv.Atoi(raw)
	if err != nil {
		return ScaleRead{}, false
	}

	out := ScaleRead{ItemCode: item}
	if s.Value == ScalePrice {
		out.Price = n
		return out, true
	}
	// ⚠️ Grams to kilograms, and the division is the only arithmetic here. A
	// label carries whole grams because a barcode has no decimal point; the rest
	// of this product speaks kilograms, so it is converted once, here, rather
	// than in each screen that would eventually convert it differently.
	out.Kg = float64(n) / 1000
	return out, true
}
