package models

import "testing"

// ⚠️ **The reading of "empty" is the whole migration**, and it is the opposite
// of the one directly above it in the same struct. A printer with no *kinds*
// chosen is half set up and must print nothing. A printer with no *categories*
// chosen is every restaurant that exists today — one kitchen printer taking all
// the food — and reading that as "nothing" would stop every kitchen ticket in
// the product on the day this shipped.
func TestAPrinterWithNoCategoriesTakesEverything(t *testing.T) {
	p := Printer{}
	if !p.Takes("dish-1", "cat-food") || !p.Takes("dish-2", "cat-drinks") {
		t.Fatal("an unconfigured printer stopped printing")
	}
	// And a dish whose category could not be looked up still prints: a database
	// blink must not silently take the kitchen offline.
	if !p.Takes("dish-3", "") {
		t.Fatal("a dish with no known category was dropped")
	}
}

func TestAPrinterTakesItsOwnSections(t *testing.T) {
	bar := Printer{Categories: []string{"drinks", "cocktails"}}
	if !bar.Takes("d1", "drinks") {
		t.Fatal("the bar refused a drink")
	}
	if bar.Takes("d2", "food") {
		t.Fatal("the bar printed the food")
	}
}

// ⚠️ **The exceptions are what make this usable at all.** Every menu has a few:
// the dessert that comes off the bar's ice cream machine, the soup the grill
// section makes. Without them a restaurant has to reorganise its menu to match
// its printers.
func TestTheExceptionsBeatTheSection(t *testing.T) {
	bar := Printer{
		Categories: []string{"drinks"},
		Only:       []string{"ice-cream"}, // made at the bar, filed under food
		Except:     []string{"bottled-water"}, // taken from the fridge, no ticket
	}
	if !bar.Takes("ice-cream", "desserts") {
		t.Fatal("an explicit dish did not reach its printer")
	}
	if bar.Takes("bottled-water", "drinks") {
		t.Fatal("an excluded dish printed because of its section")
	}
	// ⚠️ Never beats always: an exception that a category could override would
	// not be an exception.
	both := Printer{Only: []string{"x"}, Except: []string{"x"}}
	if both.Takes("x", "anything") {
		t.Fatal("'never' lost to 'always'")
	}
}
