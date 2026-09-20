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
	// ---- Makers: they turn what they buy into what they sell ----
	//
	// ⚠️ **A bakery is not a shop, though it looks like one from the street.**
	// What is on its counter was flour an hour ago, which means a technical card,
	// a batch document and a cost per loaf — the whole half of the product a
	// grocery never opens. Reading it as a shop would take those screens away and
	// leave it counting bread it cannot cost.
	BizBakery BusinessType = "bakery"
	BizCoffee BusinessType = "coffee"
	BizPastry BusinessType = "pastry"
	// ---- Shops: what was delivered is what is sold ----
	BizGrocery   BusinessType = "grocery"
	BizButcher   BusinessType = "butcher"
	BizClothing  BusinessType = "clothing"
	BizCosmetics BusinessType = "cosmetics"
	BizFlowers   BusinessType = "flowers"
	BizPharmacy  BusinessType = "pharmacy"
	BizHardware  BusinessType = "hardware"
	// ---- Online only: the shelf is a web page ----
	//
	// ⚠️ **A shop with no room anybody walks into**, and that is the whole
	// difference. A grocery's site is a second way to reach a counter that
	// exists; an online store's site *is* the store, so the switches a shop
	// starts with are exactly backwards for it: delivery off, pickup on, and a
	// catalogue laid out for somebody standing in front of the shelf.
	//
	// ⚠️ **Not a variety of `clothing`**, though its first customers will sell
	// clothes. What it sells is unknown to us — a boutique, a phone case shop,
	// a book seller — and the one thing every one of them shares is that the
	// parcel leaves by post rather than on our own courier's motorbike.
	BizEcommerce BusinessType = "ecommerce"
)

// BusinessTypes is what the console offers, in the order it offers them.
//
// ⚠️ Ordered by how many of them there are rather than alphabetically: the list
// is read by somebody creating a customer, and the common answer belongs at the
// top.
//
// ⚠️ **Kitchens first, then shops**, because that is the question the person
// creating a customer has already answered before they open the list: they know
// whether the place cooks. Interleaving the two by popularity would make them
// read the whole list every time.
var BusinessTypes = []BusinessType{
	BizRestaurant, BizFastFood, BizCoffee, BizBakery, BizPastry,
	BizGrocery, BizButcher, BizClothing, BizCosmetics, BizFlowers,
	BizPharmacy, BizHardware,
	// ⚠️ **Last, and not among the shops.** Somebody creating a customer knows
	// first whether the place cooks and second whether it has a room — an
	// online store answers no to both, so it belongs after every business that
	// has an address a guest can walk to rather than interleaved with them.
	BizEcommerce,
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
	case BizGrocery, BizButcher, BizClothing, BizCosmetics,
		BizFlowers, BizPharmacy, BizHardware, BizEcommerce:
		return true
	}
	return false
}

// SellsOnlineOnly reports whether there is no room a customer can walk into.
//
// ⚠️ **Separate from SellsGoods, because a grocery answers the two
// differently.** A grocery sells what it bought *and* has a counter, so its
// till, its shelf labels and its scale are all real. An online store has the
// first half and none of the second: its whole shop is the site, which is why
// it is the one business whose navigation bar is worth letting somebody edit.
func (b BusinessType) SellsOnlineOnly() bool { return b.known() == BizEcommerce }

// ShipsByPost reports whether what is sold leaves in a parcel rather than in a
// bag handed across a counter or on our own courier's motorbike.
//
// ⚠️ **Three types, and the list is a fact about the goods rather than about
// the shop.** A dress and a lipstick are small, light, unbreakable and bought
// by somebody in another city; a kilo of mince and a bunch of tulips are none
// of those things and never travel by post whatever the shop would like. A
// carrier integration offered to a butcher is a setting that can only ever
// produce a parcel nobody collects.
//
// ⚠️ A default, not a rule — the same trade every predicate in this file
// makes. A hardware shop that starts posting screwdrivers adds the carrier by
// hand in the panel, and nothing here stops it.
func (b BusinessType) ShipsByPost() bool {
	switch b.known() {
	case BizEcommerce, BizClothing, BizCosmetics:
		return true
	}
	return false
}

// HasVariants reports whether one product is sold in sizes and colours.
//
// ⚠️ **A clothes shop and an online store, and only those two by default.**
// Sizes are the whole shape of a boutique's catalogue and are meaningless in a
// pharmacy — a box of paracetamol has no colour. A grocery does have pack
// sizes, but they are separate products with separate barcodes and separate
// prices, which is what it already enters them as; a matrix generator there
// would double its catalogue by accident.
//
// ⚠️ **An online store is in because its catalogue is the only thing it has.**
// Whatever it sells — a case, a dress, a kettle — the guest chooses from a
// page rather than from a shelf, and a page that cannot say "this one in blue"
// sends them to somebody else's page to find out.
//
// ⚠️ A default, not a rule: a product that already has variants keeps its
// editor whatever kind of business this is. Mirrored by `hasVariants` in
// frontend/src/lib/types.ts.
func (b BusinessType) HasVariants() bool {
	switch b.known() {
	case BizClothing, BizEcommerce:
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
// ⚠️ **Cooked *to order* is the test, not "has an oven".** A bakery and a
// pastry shop bake in batches before the doors open — their document is a
// production batch, not a ticket a cook watches — so a kitchen screen there
// would sit empty all day beside the screen they actually need. A coffee house
// makes every cup after somebody asks for it, which is the same shape as a
// kitchen ticket even though nothing is cooked.
func (b BusinessType) HasKitchen() bool {
	switch b.known() {
	case BizRestaurant, BizFastFood, BizCoffee:
		return true
	}
	return false
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
//
// ⚠️ **A bakery, a coffee house and a pastry shop compose without a floor and
// mostly without a kitchen screen**, which is the florist's lesson arriving
// three more times: bread is flour, water and salt, a cappuccino is a shot and
// milk, a cake is six lines and a box. Take the cards away and each of them
// loses the one screen that says what its own counter costs it.
func (b BusinessType) Composes() bool {
	switch b.known() {
	case BizRestaurant, BizFastFood, BizFlowers,
		BizBakery, BizCoffee, BizPastry:
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
	case b.known() == BizEcommerce:
		// ⚠️ **The shop rule below would be exactly backwards here**, and
		// silently: an online store created with delivery off is a catalogue
		// that cannot be bought from, and the owner's first order is the one
		// that does not arrive. There is no counter to hand anything over at —
		// pickup stays on only because a pickup point is a real thing an online
		// store may offer, and it is one tap to switch off.
		return BrandFeatures{Delivery: true, Pickup: true, DineIn: false, Booking: false}
	case b.SellsGoods():
		// A shop hands the goods over at the counter. Delivery is off rather
		// than impossible — a pharmacy that starts delivering turns it on.
		return BrandFeatures{Delivery: false, Pickup: true, DineIn: false, Booking: false}
	case b.known() == BizFastFood, b.known() == BizBakery,
		b.known() == BizCoffee, b.known() == BizPastry:
		// ⚠️ **A counter that cooks: hands it over and carries it, but seats
		// nobody.** A bakery's morning bread goes to shops and offices and a
		// pastry shop's cakes are ordered and delivered — switching delivery off
		// for them would be the florist's mistake with a different date on it.
		return BrandFeatures{Delivery: true, Pickup: true, DineIn: false, Booking: false}
	default:
		return BrandFeatures{Delivery: true, Pickup: true, DineIn: true, Booking: true}
	}
}
