package models

import "testing"

// ⚠️ **Every case here is a way to charge for a quantity nobody weighed.** A
// wrong decode does not fail: it beeps, shows the right product and prints a
// receipt with the wrong number on it. None of these would be found by using the
// till — only by a stocktake, weeks later.

func labelFor(kind string) ScaleLabel {
	return ScaleLabel{Enabled: true, Value: kind}.Defaults()
}

func TestAWeightLabelDecodesToKilograms(t *testing.T) {
	// 2 | 123456 | 01250 | check  ->  item 123456, 1.250 kg. Thirteen digits,
	// because a scale prints an EAN-13 and nothing else.
	if _, ok := labelFor(ScaleWeight).Read("212345601250 3"); ok {
		t.Fatal("a code with a space was accepted")
	}
	got, ok := labelFor(ScaleWeight).Read("2123456012503")
	if !ok {
		t.Fatal("a valid label was refused")
	}
	if got.ItemCode != "123456" {
		t.Fatalf("item = %q", got.ItemCode)
	}
	if got.Kg != 1.25 {
		t.Fatalf("kg = %v, want 1.25", got.Kg)
	}
	if got.Price != 0 {
		t.Fatal("a weight label produced a price")
	}
}

// ⚠️ **The failure that empties a shelf on paper.** A scale set to print price,
// read as grams, turns 12 500 som into 12.5 kg of goods leaving the store.
func TestPriceAndWeightAreNotInterchangeable(t *testing.T) {
	code := "2123456125003"
	byWeight, _ := labelFor(ScaleWeight).Read(code)
	byPrice, _ := labelFor(ScalePrice).Read(code)
	if byWeight.Kg == 0 || byPrice.Price == 0 {
		t.Fatal("one of the two kinds decoded nothing")
	}
	if byPrice.Kg != 0 {
		t.Fatal("a price label produced a weight")
	}
	if byWeight.Price != 0 {
		t.Fatal("a weight label produced a price")
	}
}

// ⚠️ **Off means off.** A prefix that matched an ordinary EAN would turn a
// normal product into a weighed one — five digits of its barcode read as a
// price — in every shop that has no scales at all.
func TestAnOrdinaryBarcodeIsNeverReadAsALabel(t *testing.T) {
	if _, ok := (ScaleLabel{}).Read("2123456012503"); ok {
		t.Fatal("a scale label was decoded while the feature is off")
	}
	on := labelFor(ScaleWeight)
	// ⚠️ An ordinary retail product never begins with 2 — GS1 reserves those
	// prefixes for in-store and variable-measure items — so a normal EAN is
	// turned away by its first digit rather than by luck.
	if _, ok := on.Read("4780123456789"); ok {
		t.Fatal("an ordinary EAN was decoded as a scale label")
	}
}

// ⚠️ **A wrong "no" costs one manual entry; a wrong "yes" costs money.** So
// anything not exactly right is declined.
func TestAnythingUncertainIsDeclined(t *testing.T) {
	on := labelFor(ScaleWeight)
	for _, bad := range []string{
		"",
		"212345601250",   // one short — a scanner misread, not a product
		"21234560125034", // one long
		"212345601250A",  // a letter where a digit belongs
		"2 123456 01250",
	} {
		if _, ok := on.Read(bad); ok {
			t.Fatalf("%q was decoded", bad)
		}
	}
}

// ⚠️ A branch that has never opened the settings screen must still decode the
// common arrangement, or the feature works only for shops that found a form.
func TestDefaultsCoverTheCommonScale(t *testing.T) {
	s := ScaleLabel{Enabled: true}.Defaults()
	if s.Prefix != "2" || s.ItemLen != 6 || s.ValueLen != 5 || s.Value != ScaleWeight {
		t.Fatalf("defaults changed: %+v", s)
	}
	// ⚠️ The default has to add up to a real EAN-13, or it decodes nothing at
	// all — which is what this caught the first time.
	if len(s.Prefix)+s.ItemLen+s.ValueLen+1 != 13 {
		t.Fatal("the default layout is not thirteen digits")
	}
	if _, ok := s.Read("2123456012503"); !ok {
		t.Fatal("the default arrangement does not decode a common label")
	}
}

// A shop whose scales are set up differently is configured, not refused.
func TestAnUnusualLayoutIsJustSettings(t *testing.T) {
	s := ScaleLabel{
		Enabled: true, Prefix: "22", ItemLen: 4, ValueLen: 6, Value: ScalePrice,
	}
	// 22 | 1234 | 004500 | check
	got, ok := s.Read("2212340045003")
	if !ok {
		t.Fatal("a configured layout was refused")
	}
	if got.ItemCode != "1234" || got.Price != 4500 {
		t.Fatalf("decoded %+v", got)
	}
}
