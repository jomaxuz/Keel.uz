package models

// ---- What kind of business this brand is ----
//
// ⚠️ **A template, never a mode, and the difference is the whole design.** A
// mode would say "this is a shop, so there is no kitchen screen" — and five
// types across a dozen screens is a set of combinations nobody tests, which the
// first real customer walks straight into: a fast food with a shop beside it, a
// flower shop that sells coffee. The system answers "no", and that is heard as
// "Keel did not fit us".
//
// A template only decides what is **switched on when the brand is created**.
// Every one of those switches stays visible and changeable in the panel
// afterwards, so an unusual business is a few taps rather than a refusal.
//
// ⚠️ **The stock module is deliberately not in this file.** A purchase is a
// purchase, a write-off is a write-off, and a balance is a balance — in a
// restaurant, in a pharmacy and in a flower shop alike. Arithmetic that changed
// with the business type would be arithmetic nobody could check, and this is the
// one part of the product where a wrong number is silent.
//
// ⚠️ **It lives on the brand, not on the branch.** The menu belongs to a brand
// and the store belongs to a branch (see CLAUDE.md); what separates a shop from
// a restaurant is the *catalogue* and how it is sold, so it separates where the
// catalogue lives. A branch already belongs to exactly one brand and inherits
// this — written in both places, the two would eventually disagree, and one
// catalogue sold two different ways is the kind of fault that is correct on one
// screen and quietly wrong on the next.

// BusinessType is what a brand sells and how it is rung up.
type BusinessType string

const (
	// BizRestaurant is the empty value on purpose.
	//
	// ⚠️ **Every install that predates this field is a restaurant**, and reading
	// the zero value as anything else would change all of them on the day it
	// shipped. Same rule as an empty `mapProvider` meaning 2GIS.
	BizRestaurant BusinessType = ""
	BizFastFood   BusinessType = "fastfood"
	BizGrocery    BusinessType = "grocery"
	BizClothing   BusinessType = "clothing"
	BizFlowers    BusinessType = "flowers"
	BizPharmacy   BusinessType = "pharmacy"
)

// BusinessTypes is what the console offers, in the order it offers them.
//
// ⚠️ Ordered by how many of them there are rather than alphabetically: the list
// is read by somebody creating a customer, and the common answer belongs at the
// top.
var BusinessTypes = []BusinessType{
	BizRestaurant, BizFastFood, BizGrocery, BizClothing, BizFlowers, BizPharmacy,
}

// Valid reports whether a stored value is one this build knows.
//
// ⚠️ An unknown value is treated as a restaurant rather than refused: a brand
// written by a newer build and read by an older one must not lose its catalogue
// — it loses the tailoring, which is a much cheaper wrong answer.
func (b BusinessType) Valid() bool {
	for _, t := range BusinessTypes {
		if t == b {
			return true
		}
	}
	return false
}

// known collapses anything this build does not recognise to a restaurant.
//
// ⚠️ **Every predicate goes through this, and the test is why.** They were
// written as `b == BizRestaurant` and `switch b`, which quietly answered "no
// tables, no kitchen" for a value written by a newer build — so a brand read by
// an older container would have lost its floor plan rather than its tailoring.
// Falling back has to be one decision in one place, or each predicate falls back
// differently and only some of them are noticed.
func (b BusinessType) known() BusinessType {
	if b.Valid() {
		return b
	}
	return BizRestaurant
}

// SellsGoods reports whether this brand sells the thing it bought.
//
// ⚠️ **The one question the whole difference reduces to.** A kitchen turns
// inputs into outputs, so what is sold and what is stocked are two objects with
// a tech card between them. A shop sells the object it purchased, so they are
// one — and every screen that differs, differs because of this.
func (b BusinessType) SellsGoods() bool {
	switch b.known() {
	case BizGrocery, BizClothing, BizFlowers, BizPharmacy:
		return true
	}
	return false
}

// ScansToSell reports whether the counter's first action is a scanner.
//
// ⚠️ Separate from SellsGoods, because they are not the same question and a fast
// food is the case that proves it: it cooks, so it does not sell goods — and it
// has no tables either, so its till opens on the menu rather than on a floor
// plan. Folding the two together would put a floor plan in front of a counter
// nobody sits at.
func (b BusinessType) ScansToSell() bool { return b.SellsGoods() }

// HasTables reports whether guests sit down.
func (b BusinessType) HasTables() bool { return b.known() == BizRestaurant }

// HasKitchen reports whether anything is cooked to order.
//
// ⚠️ A default, not a rule: a flower shop with a coffee machine switches the
// kitchen screen back on in the panel, and nothing here stops it.
func (b BusinessType) HasKitchen() bool {
	k := b.known()
	return k == BizRestaurant || k == BizFastFood
}

// Composes reports whether what is sold is assembled from other things it
// stocks — which is the question a technical card actually answers.
//
// ⚠️ **Not the same question as HasKitchen, and reading it as the same one was
// wrong for exactly one business.** A florist has no kitchen and cooks nothing,
// and a bouquet is fifteen stems, a wrap and a ribbon — the most literal
// technical card in the product. Hiding the cards from flower shops took away
// the screen that tells them what a bouquet costs and what a bad week wrote
// off, which for a shop with that much waste is the screen worth paying for.
//
// ⚠️ A grocery, a clothes shop and a pharmacy genuinely do not compose: the
// packet on the shelf is the packet that was delivered, and their products keep
// the one-line card the server writes for them.
func (b BusinessType) Composes() bool {
	switch b.known() {
	case BizRestaurant, BizFastFood, BizFlowers:
		return true
	}
	return false
}

// Defaults returns the switches a new brand of this type starts with.
//
// ⚠️ **Returned rather than enforced.** This is called once, when the brand is
// created; nothing reads it afterwards, and the panel's own toggles are the
// truth from that moment on.
func (b BusinessType) Defaults() BrandFeatures {
	switch {
	case b.known() == BizFlowers:
		// ⚠️ **A florist delivers, and switching that off was a real mistake
		// rather than a conservative default.** Half of what a flower shop
		// sells is carried to somebody else's address — on the eighth of March
		// it is nearly all of it — and a shop set up the week before that with
		// delivery off would find out on the busiest morning of its year.
		return BrandFeatures{Delivery: true, Pickup: true, DineIn: false, Booking: false}
	case b.SellsGoods():
		// A shop hands the goods over at the counter. Delivery is off rather
		// than impossible — a pharmacy that starts delivering turns it on.
		return BrandFeatures{Delivery: false, Pickup: true, DineIn: false, Booking: false}
	case b.known() == BizFastFood:
		return BrandFeatures{Delivery: true, Pickup: true, DineIn: false, Booking: false}
	default:
		return BrandFeatures{Delivery: true, Pickup: true, DineIn: true, Booking: true}
	}
}
