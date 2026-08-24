package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Restaurant (singleton) ----

type WorkingHour struct {
	Day      int    `bson:"day" json:"day"` // 0=Sunday .. 6=Saturday
	Open     string `bson:"open" json:"open"`
	Close    string `bson:"close" json:"close"`
	IsClosed bool   `bson:"isClosed" json:"isClosed"`
}

type GeoPoint struct {
	Text string  `bson:"text" json:"text"`
	Lat  float64 `bson:"lat" json:"lat"`
	Lng  float64 `bson:"lng" json:"lng"`
}

type Socials struct {
	Instagram string `bson:"instagram" json:"instagram"`
	Telegram  string `bson:"telegram" json:"telegram"`
	Facebook  string `bson:"facebook" json:"facebook"`
}

// DeliveryZone is an optional polygon-based zone drawn in the admin panel.
// Coordinates are [lat, lng] pairs. If no zone with 3+ points is configured,
// the radius model is used instead.
//
// Each zone prices independently, chosen by the restaurant:
//   - "fixed"  (default): flat Fee for anywhere inside the zone
//   - "perKm": BaseFee + ceil(km) * PerKm, km measured from the restaurant
//
// Pricing is empty on documents written before this field existed, which reads
// as "fixed" — the old behaviour.
type DeliveryZone struct {
	Name    string      `bson:"name" json:"name"`
	Polygon [][]float64 `bson:"polygon" json:"polygon"`
	Pricing string      `bson:"pricing" json:"pricing"` // "fixed" | "perKm"
	Fee     int         `bson:"fee" json:"fee"`         // fixed model
	BaseFee int         `bson:"baseFee" json:"baseFee"` // perKm model
	PerKm   int         `bson:"perKm" json:"perKm"`     // perKm model
}

type DeliverySettings struct {
	Enabled          bool `bson:"enabled" json:"enabled"`
	MinOrder         int  `bson:"minOrder" json:"minOrder"`
	FreeDeliveryFrom *int `bson:"freeDeliveryFrom" json:"freeDeliveryFrom"`

	// How the fee is computed: "radius" (distance from the restaurant) or
	// "zones" (polygons drawn on the map). Empty on documents written before
	// this field existed, which is inferred: zones if any are drawable.
	Mode  string         `bson:"mode" json:"mode"`
	Zones []DeliveryZone `bson:"zones" json:"zones"`

	// How close (metres) a courier must be to the customer address before the
	// app lets them mark the order delivered. 0 disables the check.
	ArrivalRadiusM int `bson:"arrivalRadiusM" json:"arrivalRadiusM"`

	// Radius model.
	BaseFee int     `bson:"baseFee" json:"baseFee"`
	PerKm   int     `bson:"perKm" json:"perKm"`
	MaxKm   float64 `bson:"maxKm" json:"maxKm"`
}

// PreorderSettings is "order now, for later" — a guest picking a time instead
// of being cooked for the moment they tap.
//
// It belongs to the **branch** for the same reason the delivery zones do: the
// kitchen that will cook it is the only one that knows how far ahead it needs
// warning, and one branch closing at 21:00 cannot take the other's late slots.
// ServiceCharge is the percentage a dining room adds to a table's bill.
//
// ⚠️ **A branch setting, not a company one.** A chain's restaurant with waiters
// charges for service and its counter outlet in a shopping centre does not, and
// one number for both would put a service charge on a takeaway coffee — which
// is the version of this feature guests complain about.
//
// ⚠️ **Zero is off**, like every other setting here: the field does not exist on
// any branch created before it, and reading a missing value as "charge nothing"
// is the only reading that leaves those restaurants alone.
type ServiceCharge struct {
	Enabled bool `bson:"enabled" json:"enabled"`
	// Whole percent. ⚠️ Not a fraction and not a fixed sum: every restaurant in
	// the country states this as "10%", and a field that means something else
	// than the sign on the door is a field somebody fills in wrong.
	Percent int `bson:"percent" json:"percent"`
}

type PreorderSettings struct {
	// Off by default, which is every install that predates this field: nobody
	// has ever placed a scheduled order, so "off" is exactly today's behaviour.
	Enabled bool `bson:"enabled" json:"enabled"`
	// ⚠️ **The whole feature, in one number.** How long before the requested
	// time the order becomes the kitchen's problem: the panel chimes, the pass
	// shows the ticket, and until then the order is stored and silent.
	//
	// It is the owner's to set because only they know what their kitchen needs
	// warning about — an hour for a plov, ten minutes for a coffee — and
	// guessing it for them means either a cold order or a cook who learns to
	// ignore the bell.
	LeadMinutes int `bson:"leadMinutes" json:"leadMinutes"`
	// The earliest a guest may ask for, measured from now. Separate from the
	// lead time on purpose: the lead is what the kitchen needs, this is what
	// the restaurant is willing to promise, and a restaurant that wants an
	// hour's notice from guests may still want its own bell 20 minutes ahead.
	MinMinutes int `bson:"minMinutes" json:"minMinutes"`
	// How far ahead a slot may be picked, in days. 0 = today only.
	MaxDays int `bson:"maxDays" json:"maxDays"`
	// The granularity the guest chooses on: 30 means half-hour slots.
	SlotMinutes int `bson:"slotMinutes" json:"slotMinutes"`
}

// ReviewSettings decides whether the guests' own words appear on the site.
//
// One switch, deliberately. There is no "only show 4 stars and above" knob and
// there should not be: a rule like that turns the section into a wall of praise
// the restaurant assembled about itself, which readers discount the moment they
// notice — and they notice. Which reviews appear is chosen one at a time on the
// feedback screen (see Feedback.IsPublic), where the owner is looking at the
// actual words rather than at a threshold.
type ReviewSettings struct {
	// Off by default, which is every install that predates this: nothing a
	// guest wrote privately starts appearing because a field was added.
	Enabled bool `bson:"enabled" json:"enabled"`
	// Show the average and how many ratings it is from, above the comments.
	// Separate from the comments because it is a different claim: a statistic
	// nobody is quoted in, and a restaurant may reasonably want one without
	// the other.
	ShowAverage bool `bson:"showAverage" json:"showAverage"`
}

// LocalizedText is a piece of site copy in the three supported languages.
// Uzbek is the base: an empty ru/en falls back to it, same rule as the menu.
type LocalizedText struct {
	Uz string `bson:"uz" json:"uz"`
	Ru string `bson:"ru" json:"ru"`
	En string `bson:"en" json:"en"`
}

// SiteContent is the editable copy of the public pages — everything that used
// to be either hardcoded or borrowed from the short description.
type SiteContent struct {
	AboutTitle LocalizedText `bson:"aboutTitle" json:"aboutTitle"`
	AboutText  LocalizedText `bson:"aboutText" json:"aboutText"`
	FooterNote LocalizedText `bson:"footerNote" json:"footerNote"`
	// Shown under the hero heading on the home page.
	Tagline LocalizedText `bson:"tagline" json:"tagline"`
	// The strip of cards on the home page. Empty means the built-in three
	// ("fast delivery", "fresh produce", "easy payment") — what every site has
	// shown so far, and what a restaurant that never opens this section keeps.
	Perks []PerkCard `bson:"perks" json:"perks"`
	// ⚠️ **"hide", not "show".** The zero value has to be the page as it is
	// today: a `showPerks` field would empty the strip on every existing site
	// the day it shipped. Same rule as booking.hidePlan and an empty mapProvider.
	HidePerks bool `bson:"hidePerks" json:"hidePerks"`
}

// PerkCard is one card in the home page strip: an icon and two lines of copy.
//
// The icon is a name, not a path — the drawings live in the frontend, where the
// console's own perk band already keeps them, so a restaurant cannot type an
// <svg> into its own page.
type PerkCard struct {
	Icon  string        `bson:"icon" json:"icon"`
	Title LocalizedText `bson:"title" json:"title"`
	Text  LocalizedText `bson:"text" json:"text"`
}

// IsEmpty reports whether a brand has been given any copy of its own.
//
// ⚠️ A plain `c != SiteContent{}` compiled until Perks arrived: a struct with a
// slice is not comparable. Written out rather than reflect.DeepEqual so the
// compiler still points here when a field is added — silently failing to notice
// a brand's copy would show the company's text on the brand's site.
func (c SiteContent) IsEmpty() bool {
	return c.AboutTitle == LocalizedText{} &&
		c.AboutText == LocalizedText{} &&
		c.FooterNote == LocalizedText{} &&
		c.Tagline == LocalizedText{} &&
		len(c.Perks) == 0 &&
		!c.HidePerks
}

// SiteTheme lets the restaurant restyle the public site from the admin panel:
// accent colour, corner roundness and the font pairing. Empty fields fall back
// to the built-in design.
type SiteTheme struct {
	Brand string `bson:"brand" json:"brand"` // "#e2590d"
	// Accent used in dark mode; empty = derived from Brand.
	BrandDark string `bson:"brandDark" json:"brandDark"`
	// Card corner radius in px (0–32). Controls derive from it.
	Radius *int `bson:"radius" json:"radius"`
	// "pill" (default) or "match" — square-ish buttons that follow Radius.
	ButtonShape string `bson:"buttonShape" json:"buttonShape"`
	// Font pairing preset: "classic" | "modern" | "soft".
	Font string `bson:"font" json:"font"`
	// Page/card background tone: "warm" (default) | "white" | "cool" | "sand".
	Background string `bson:"background" json:"background"`
	// Card depth: "soft" (default) | "none" | "strong".
	Shadow string `bson:"shadow" json:"shadow"`
	// Primary button look: "solid" (default) | "outline" | "soft".
	ButtonStyle string `bson:"buttonStyle" json:"buttonStyle"`
	// Root font size in px (15–18); scales text and spacing together.
	Scale *int `bson:"scale" json:"scale"`
}

// SEOSettings holds the strings a search engine hands the owner to prove the
// site is theirs, plus whatever else search consoles need in the page head.
//
// ⚠️ Like the map key, these are **meant** to reach the browser: a verification
// token's whole job is to sit in the HTML head where a crawler can read it.
// That is why they live here and not in a settings collection kept out of the
// public document — hiding one would only stop the verification from working.
// They prove nothing on their own; Google and Yandex only accept a token on the
// domain they issued it for.
type SEOSettings struct {
	// Google Search Console: the `content` of its meta tag, not the whole tag.
	Google string `bson:"google" json:"google"`
	// Yandex Webmaster: same shape. Yandex matters more here than Google —
	// it is the search most of these restaurants' customers actually use.
	Yandex string `bson:"yandex" json:"yandex"`
}

type Restaurant struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name         string             `bson:"name" json:"name"`
	Description  string             `bson:"description" json:"description"`
	LogoURL      string             `bson:"logoUrl" json:"logoUrl"`
	CoverURL     string             `bson:"coverUrl" json:"coverUrl"`
	Phones       []string           `bson:"phones" json:"phones"`
	Address      GeoPoint           `bson:"address" json:"address"`
	Socials      Socials            `bson:"socials" json:"socials"`
	WorkingHours []WorkingHour      `bson:"workingHours" json:"workingHours"`
	Delivery     DeliverySettings   `bson:"delivery" json:"delivery"`
	// Table booking: the hand-drawn floor plan and the rules around it.
	Booking BookingSettings `bson:"booking" json:"booking"`
	// Ordering ahead of time. Branch-owned like the hours and the zones; laid
	// over this document by GetRestaurant so the site reads one picture.
	Preorder PreorderSettings `bson:"preorder" json:"preorder"`
	// Whether guests' ratings and comments appear on the public site.
	Reviews  ReviewSettings `bson:"reviews" json:"reviews"`
	Currency string         `bson:"currency" json:"currency"`
	// Cashback points. Company-wide, because the customer is: a regular of the
	// samsa shop is the same person in the restaurant.
	Loyalty LoyaltySettings `bson:"loyalty" json:"loyalty"`
	Content SiteContent     `bson:"content" json:"content"`
	Theme   SiteTheme       `bson:"theme" json:"theme"`
	// The restaurant's own 2GIS MapGL key, entered in the admin panel.
	//
	// ⚠️ This one is **meant** to reach the browser, and that is the opposite
	// of the payment credentials — those were deliberately moved out of this
	// document because it is returned in full to every visitor. MapGL is a
	// browser library: the key is handed to `load({key})` in the page, so it is
	// visible in DevTools no matter where we keep it. No map SDK works any
	// other way.
	//
	// What protects it is not secrecy but the **domain restriction** set in the
	// 2GIS account: a copied key does not work anywhere else. The settings page
	// says so, because a key left unrestricted really is unprotected — and
	// somebody who "fixes" this by hiding it will only break the map.
	MapAPIKey string `bson:"mapApiKey" json:"mapApiKey"`
	// Which map draws the site. Empty means 2GIS — every install that predates
	// this field is on 2GIS, and reading the zero value as anything else would
	// blank the map on all of them.
	//
	// ⚠️ **A key per provider, not one shared field.** An owner who tries Yandex
	// and goes back to 2GIS must not end up handing one provider the other's
	// key: the map would simply not draw, with a console error nobody in a
	// restaurant reads. Same drawer-per-provider rule as the POS credentials.
	// All three are public for the reason above — the restriction that protects
	// them lives in each provider's own console, as a list of allowed domains.
	MapProvider  string      `bson:"mapProvider" json:"mapProvider"` // "" | "2gis" | "yandex" | "google"
	MapYandexKey string      `bson:"mapYandexKey" json:"mapYandexKey"`
	MapGoogleKey string      `bson:"mapGoogleKey" json:"mapGoogleKey"`
	SEO          SEOSettings `bson:"seo" json:"seo"`
	UpdatedAt    time.Time   `bson:"updatedAt" json:"updatedAt"`
}

// ---- Table booking ----

// FloorShape is a wall, a window or any other piece of the room the owner drew
// to make the plan recognisable. Purely decorative: nothing is booked on it.
type FloorShape struct {
	Kind  string  `bson:"kind" json:"kind"` // "wall" | "area"
	Label string  `bson:"label" json:"label"`
	X     float64 `bson:"x" json:"x"`
	Y     float64 `bson:"y" json:"y"`
	W     float64 `bson:"w" json:"w"`
	H     float64 `bson:"h" json:"h"`
	// What colour the owner painted it.
	//
	// ⚠️ **Empty is the default grey, not black.** Every shape drawn before this
	// existed has no colour, and reading that as a real value would repaint
	// every floor plan in the country on the deploy that shipped it — the same
	// zero-value rule an empty `mapProvider` follows.
	//
	// ⚠️ **A named choice, never free text.** The value comes from a fixed
	// palette (see the panel's colour picker): this string is rendered straight
	// into an SVG `fill`, so an arbitrary one is a place to put something that
	// is not a colour. The renderers refuse anything they do not recognise.
	Color string `bson:"color,omitempty" json:"color,omitempty"`
}

// FloorColors are the colours a plan may be painted in.
//
// ⚠️ **A closed list, and it is the whole of the validation.** These strings
// reach an SVG `fill` attribute on a page served to guests; accepting whatever
// the form sent would make the floor plan editor a way to put arbitrary content
// into everybody's booking page. Six is also about the number of areas a dining
// room actually has — a picker with thirty swatches is a decision nobody wants
// to make about a wall.
var FloorColors = map[string]bool{
	"slate": true, "amber": true, "green": true,
	"blue": true, "rose": true, "violet": true,
}

// FloorColor is the stored colour, or "" for the default.
func FloorColor(v string) string {
	if FloorColors[v] {
		return v
	}
	return ""
}

// FloorTable is one bookable table on the plan. Coordinates are in plan units
// (the plan has its own width/height), so the drawing scales to any screen.
type FloorTable struct {
	// Stable id generated when the table is drawn: reservations point at it, so
	// it must survive renaming and renumbering.
	ID     string  `bson:"id" json:"id"`
	Number string  `bson:"number" json:"number"`
	Seats  int     `bson:"seats" json:"seats"`
	Shape  string  `bson:"shape" json:"shape"` // "rect" | "circle"
	X      float64 `bson:"x" json:"x"`
	Y      float64 `bson:"y" json:"y"`
	W      float64 `bson:"w" json:"w"`
	H      float64 `bson:"h" json:"h"`
	// A table taken out of service stays on the plan but cannot be booked.
	IsActive bool   `bson:"isActive" json:"isActive"`
	Note     string `bson:"note" json:"note"`
	// What colour this table is drawn in — see FloorShape.Color for why it is a
	// name from a closed list rather than free text.
	//
	// ⚠️ **The till ignores it.** Over there colour means state — free, sitting,
	// billed, waited too long — and letting a decorative colour into that
	// language would make the one screen scanned across a room during a rush
	// unreadable. This is for the booking page and the panel's own plan, where
	// the question is "which is the terrace".
	Color string `bson:"color,omitempty" json:"color,omitempty"`

	// Which part of the business this table belongs to.
	//
	// ⚠️ **Empty means the default zone, and that zone is bookable** — every
	// table drawn before zones existed has no id here, and reading that as "no
	// zone" would take every restaurant's whole floor plan out of the booking
	// page on the deploy that added this. The usual zero-value rule, and the
	// direction that cannot break a live restaurant.
	ZoneID string `bson:"zoneId,omitempty" json:"zoneId,omitempty"`
}

// TableZone is a group of tables that behave the same way.
//
// ⚠️ **The reason this exists is that "table 112" is not a table.** A takeaway
// counter numbers its orders 100–130 and those numbers are not seats a guest
// can reserve — but the till needs them, because a takeaway order has to be
// opened against something. One list would either put them on the booking page
// or keep them off the till.
type TableZone struct {
	ID   string `bson:"id" json:"id"`
	Name string `bson:"name" json:"name"`
	// Whether a guest can reserve a table here.
	//
	// ⚠️ The whole point of the flag: the hall is bookable, the takeaway counter
	// is not, and both are tables as far as the till is concerned.
	Bookable bool `bson:"bookable" json:"bookable"`
	// How the till draws it: "map" follows the coordinates a floor plan gives
	// each table, "list" ignores them and shows numbers in a grid.
	//
	// ⚠️ Empty means **map**, because every zone that exists today is the drawn
	// floor plan. A takeaway zone is created as a list and never has
	// coordinates — drawing it would put thirty numbered squares in the top-left
	// corner on top of each other.
	Layout string `bson:"layout,omitempty" json:"layout,omitempty"`
	Sort   int    `bson:"sort" json:"sort"`
}

// ZoneLayout values.
const (
	ZoneMap  = "map"
	ZoneList = "list"
)

// Bookable reports whether a table may be reserved by a guest.
//
// ⚠️ **One function, because three screens ask.** The booking page, the
// reservation validator and the panel's plan all need the same answer, and a
// table that is reservable on one of them and not the others is a double
// booking waiting to happen.
//
// A table with no zone is bookable: that is every table drawn before zones
// existed, and the deploy that added them must not empty anybody's booking page.
func (b *BookingSettings) Bookable(t FloorTable) bool {
	if !t.IsActive {
		return false
	}
	if t.ZoneID == "" {
		return true
	}
	for _, z := range b.Zones {
		if z.ID == t.ZoneID {
			return z.Bookable
		}
	}
	// ⚠️ A table pointing at a zone that no longer exists stays bookable rather
	// than vanishing: a deleted zone is an editing accident, and silently
	// removing tables from the booking page is the kind of failure nobody
	// notices until a guest cannot reserve anything.
	return true
}

type BookingSettings struct {
	Enabled bool `bson:"enabled" json:"enabled"`
	// The parts of the business tables belong to — the hall, the takeaway
	// counter, a terrace. Empty on every restaurant that has not split them,
	// which is the ordinary case and reads as one unnamed bookable zone.
	Zones []TableZone `bson:"zones" json:"zones"`
	// The plan's own coordinate space; the site scales it to fit.
	Width  float64 `bson:"width" json:"width"`
	Height float64 `bson:"height" json:"height"`
	// How long one booking holds a table, and how far ahead guests may book.
	SlotMinutes  int `bson:"slotMinutes" json:"slotMinutes"`
	MaxDaysAhead int `bson:"maxDaysAhead" json:"maxDaysAhead"`
	// Minimum notice: a table cannot be booked for five minutes from now.
	MinNoticeMinutes int `bson:"minNoticeMinutes" json:"minNoticeMinutes"`
	MaxGuests        int `bson:"maxGuests" json:"maxGuests"`
	// Whether guests choose their own table, or only ask for a time.
	//
	// ⚠️ **Hides the choice, not the bookkeeping.** With the plan hidden the
	// server still assigns a real table — the smallest free one that fits — so
	// double-booking stays impossible and every screen downstream (the panel's
	// map for a moment, "is table 7 free at eight", the receipt's table number)
	// keeps working unchanged. A booking with no table at all would have meant
	// teaching all of those about a second kind of reservation.
	//
	// ⚠️ The flag is "hide" rather than "show" so its zero value is the
	// behaviour every existing restaurant already has. A `showPlan` field would
	// have switched table picking off for all of them on the day it shipped —
	// the same reason an empty `mapProvider` has to mean 2GIS.
	HidePlan bool         `bson:"hidePlan,omitempty" json:"hidePlan,omitempty"`
	Shapes   []FloorShape `bson:"shapes" json:"shapes"`
	Tables   []FloorTable `bson:"tables" json:"tables"`
	Note     string       `bson:"note" json:"note"`
}

type ReservationStatus string

const (
	ReservationPending   ReservationStatus = "pending"
	ReservationConfirmed ReservationStatus = "confirmed"
	ReservationSeated    ReservationStatus = "seated"
	ReservationDone      ReservationStatus = "done"
	ReservationCancelled ReservationStatus = "cancelled"
)

// Reservation is one table held for one guest for one slot.
type Reservation struct {
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Number   string             `bson:"number" json:"number"`
	TableID  string             `bson:"tableId" json:"tableId"`
	// Snapshot: the plan may be redrawn, the receipt must still read correctly.
	TableNumber string             `bson:"tableNumber" json:"tableNumber"`
	UserID      primitive.ObjectID `bson:"userId,omitempty" json:"userId,omitempty"`
	Customer    OrderCustomer      `bson:"customer" json:"customer"`
	Guests      int                `bson:"guests" json:"guests"`
	// Half-open slot [at, endsAt): a table is free again exactly at endsAt.
	At      time.Time         `bson:"at" json:"at"`
	EndsAt  time.Time         `bson:"endsAt" json:"endsAt"`
	Status  ReservationStatus `bson:"status" json:"status"`
	Comment string            `bson:"comment" json:"comment"`
	// Why the restaurant cancelled it — shown to the guest, like an order.
	CancelReason  string             `bson:"cancelReason,omitempty" json:"cancelReason,omitempty"`
	StatusHistory []ReservationEvent `bson:"statusHistory" json:"statusHistory"`
	CreatedAt     time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// ReservationEvent records when a booking reached a status (same idea as an
// order's timeline, with the booking's own status set).
type ReservationEvent struct {
	Status ReservationStatus `bson:"status" json:"status"`
	At     time.Time         `bson:"at" json:"at"`
}

// ---- Brand and branch ----
//
// A company may run more than one brand (a restaurant and a samsa chain, say),
// and a brand may have several branches. Both dimensions are optional: a single
// restaurant has exactly one of each and never sees them.
//
// What belongs where:
//
//	company (the `restaurant` singleton) — currency, socials, the customers
//	brand   — menu, look, site copy, which services it offers
//	branch  — address, hours, delivery zones, couriers, orders, floor plan
type BrandFeatures struct {
	Delivery bool `bson:"delivery" json:"delivery"`
	Pickup   bool `bson:"pickup" json:"pickup"`
	// Ordering from a table by QR, and holding a table in advance.
	DineIn  bool `bson:"dineIn" json:"dineIn"`
	Booking bool `bson:"booking" json:"booking"`
}

type Brand struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name" validate:"required"`
	Slug        string             `bson:"slug" json:"slug"`
	Description string             `bson:"description" json:"description"`
	LogoURL     string             `bson:"logoUrl" json:"logoUrl"`
	CoverURL    string             `bson:"coverUrl" json:"coverUrl"`
	Content     SiteContent        `bson:"content" json:"content"`
	Theme       SiteTheme          `bson:"theme" json:"theme"`
	Features    BrandFeatures      `bson:"features" json:"features"`
	SortOrder   int                `bson:"sortOrder" json:"sortOrder"`
	IsActive    bool               `bson:"isActive" json:"isActive"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type Branch struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BrandID primitive.ObjectID `bson:"brandId" json:"brandId"`
	Name    string             `bson:"name" json:"name" validate:"required"`
	Phones  []string           `bson:"phones" json:"phones"`
	Address GeoPoint           `bson:"address" json:"address"`
	// Each branch cooks for itself, so hours, zones and the minimum order are
	// its own — one may close at 21:00 and deliver 3km, another at 23:00.
	WorkingHours []WorkingHour    `bson:"workingHours" json:"workingHours"`
	Delivery     DeliverySettings `bson:"delivery" json:"delivery"`
	// The room this branch serves, for dine-in QR codes and bookings.
	Booking BookingSettings `bson:"booking" json:"booking"`
	// Whether this kitchen takes orders for later, and how much warning it
	// wants before one is due.
	Preorder PreorderSettings `bson:"preorder" json:"preorder"`
	// What this room adds for service, and whether it adds anything.
	Service ServiceCharge `bson:"service" json:"service"`
	// Rough kitchen time, shown to the guest and used to compare branches.
	PrepMinutes int `bson:"prepMinutes" json:"prepMinutes"`
	// How close (metres) an employee must be to this address before the app
	// lets them clock in or out. 0 disables the check entirely.
	//
	// Note the floor this sits on: a phone's civilian GPS is accurate to
	// roughly 10–30 m in the open and worse indoors, which is exactly where a
	// kitchen is. A radius below that does not catch cheats, it locks honest
	// staff out — so the default is deliberately wider than the door.
	StaffRadiusM int `bson:"staffRadiusM" json:"staffRadiusM"`
	// Rotating clock-in code shown on a screen at this branch (see
	// handlers/kiosk.go). The secret never leaves the server — the kiosk asks
	// for the current code, it does not compute one.
	//
	// A printed, unchanging QR would be pointless: photograph it once and you
	// can clock in from home forever. The code is derived from this secret plus
	// the current 30-second step, so a photo is worthless a minute later.
	KioskSecret string `bson:"kioskSecret,omitempty" json:"-"`
	// Bumped to revoke every kiosk token issued so far (a lost tablet).
	KioskVersion int `bson:"kioskVersion" json:"kioskVersion"`

	// Revocation counter for the till and floor screens on this branch's
	// monoblocks.
	//
	// ⚠️ **The screens are bound to the branch, not to a person.** Nobody types
	// a username and a password on a monoblock between two guests — the machine
	// is set up once, and after that everyone identifies themselves with four
	// digits (see handlers/tillpin.go). The device token is what makes that
	// safe, and this number is what kills it: bumping it invalidates every till
	// token this branch ever issued, which is the answer to a monoblock leaving
	// the building. Same shape as KioskVersion, and for the same reason.
	TillVersion int `bson:"tillVersion" json:"tillVersion"`
	// When true, clocking in also requires a valid code — the geofence alone
	// is not enough. Off by default so existing branches keep working.
	RequireKioskCode bool `bson:"requireKioskCode" json:"requireKioskCode"`
	// Short code printed in front of this branch's order numbers ("CHL-A71-4509").
	// Empty on a single-branch install, where the prefix would say nothing.
	Code string `bson:"code" json:"code"`
	// Dishes that have run out **here today**. The menu belongs to the brand and
	// is the same everywhere; what is left in the pot is the branch's own
	// business, so this lives on the branch rather than on the dish.
	SoldOut []primitive.ObjectID `bson:"soldOut" json:"soldOut"`
	// Dishes the till itself has stopped, mirrored from the POS (see
	// handlers/posstop.go).
	//
	// ⚠️ **A second list, deliberately.** One is tapped by a person at the
	// counter, the other is rewritten wholesale every few minutes by a poller.
	// Merged into one field they would quietly undo each other: the sync would
	// put back a dish the counter had just taken off, and a counter tap would
	// clear a stop the kitchen system is still holding. Kept apart, each writer
	// owns its own list and the site simply asks whether either one names the
	// dish.
	POSSoldOut []primitive.ObjectID `bson:"posSoldOut" json:"posSoldOut"`
	// When the till was last asked, and what it said if it refused.
	//
	// A timestamp rather than a "synced" flag: a stored flag goes stale the
	// moment the clock passes it, and a stop list that stopped updating at
	// lunchtime looks exactly like one with nothing stopped.
	POSSoldOutAt    *time.Time `bson:"posSoldOutAt,omitempty" json:"posSoldOutAt,omitempty"`
	POSSoldOutError string     `bson:"posSoldOutError" json:"posSoldOutError"`

	// ---- Stopped because the store is empty ----
	//
	// ⚠️ **A third list, and the same reason as the second.** This one is
	// rewritten by the stock sync (handlers/stockstop.go); merged into either
	// of the others it would undo a counter tap, or be undone by one, and each
	// undo reads as the feature being broken rather than busy.
	StockSoldOut []primitive.ObjectID `bson:"stockSoldOut" json:"stockSoldOut"`
	// When the shelves were last worked out.
	StockSoldOutAt *time.Time `bson:"stockSoldOutAt,omitempty" json:"stockSoldOutAt,omitempty"`
	// Whether this branch stops dishes when their ingredients run out.
	//
	// ⚠️ **Off unless the owner turns it on, and that is not timidity.** The
	// balance behind it is an estimate — the last count plus deliveries less
	// what the cards say was used — so it drifts exactly as far as the kitchen
	// drifts from its cards and as far back as the last count. A restaurant
	// that has not recorded Tuesday's delivery would have its till refuse food
	// that is physically on the shelf, in the middle of service, with the
	// guest already at the counter. Blocking a sale is the most expensive thing
	// this system can do, so it is done only where somebody has said the
	// numbers are good enough to do it on.
	StockStop bool `bson:"stockStop,omitempty" json:"stockStop,omitempty"`

	// ---- Stopped because today's batch is gone ----
	//
	// ⚠️ **A limit is a rule, not a fourth list — and it writes its own list
	// anyway.** "We cooked ten portions of osh" is a plan somebody makes in the
	// morning; the stop that follows at the tenth sale is a fact about this
	// evening. Storing the plan in `soldOut` would have the kitchen's morning
	// decision cleared by a cashier's tap, and storing the stop there would put
	// a dish back on sale that has physically run out. The same lesson as the
	// till's list and the stockroom's, arriving a fourth time.
	//
	// ⚠️ **The count is not kept here.** How many were sold today is already
	// written down — in the orders — and a counter beside them is a second copy
	// that drifts the first time a sale is cancelled, refunded or moved to
	// another branch. It is worked out from the orders when a sale lands, which
	// is the only moment the answer can change.
	DailyLimits []DailyLimit `bson:"dailyLimits,omitempty" json:"dailyLimits"`
	// What the limits have stopped today. Recomputed as sales land.
	LimitSoldOut []primitive.ObjectID `bson:"limitSoldOut,omitempty" json:"limitSoldOut"`
	// The day the list above belongs to, local, "YYYY-MM-DD".
	//
	// ⚠️ **Without it the list is a stop that never lifts.** Nothing runs at
	// midnight — there is no sweep and deliberately none, for the reason the
	// preorder queue gives — so yesterday's stops would still be holding at
	// eight this morning, before a single portion had been cooked. The date is
	// what makes the list expire on its own, read rather than swept.
	LimitDate string    `bson:"limitDate,omitempty" json:"limitDate"`
	SortOrder int       `bson:"sortOrder" json:"sortOrder"`
	IsActive  bool      `bson:"isActive" json:"isActive"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// IsSoldOut reports whether a dish has run out at this branch — because somebody
// said so at the counter, because the till has it stopped, because the store is
// empty, or because today's batch is sold.
func (b *Branch) IsSoldOut(id primitive.ObjectID) bool {
	return containsID(b.SoldOut, id) || containsID(b.POSSoldOut, id) ||
		containsID(b.StockSoldOut, id) || b.IsLimitSoldOut(id)
}

// DailyLimit is how many of one dish this branch sells in a day.
type DailyLimit struct {
	MenuItemID primitive.ObjectID `bson:"menuItemId" json:"menuItemId"`
	// ⚠️ **Zero is "no limit", not "sell none".** Every branch and every dish
	// that existed before this field has no value at all, and reading the zero
	// as a limit would empty every menu in the country on the day it shipped —
	// the same rule as an empty mapProvider meaning 2GIS. A kitchen that wants
	// to stop a dish has the stop list, one tap away, which says what it means.
	Limit int `bson:"limit" json:"limit"`
}

// LimitFor is how many of this dish the branch sells today, or 0 for no limit.
func (b *Branch) LimitFor(id primitive.ObjectID) int {
	for _, l := range b.DailyLimits {
		if l.MenuItemID == id {
			return l.Limit
		}
	}
	return 0
}

// IsLimitSoldOut reports whether today's batch of a dish is gone.
//
// ⚠️ **The date is checked here rather than swept at midnight.** A background
// job that cleared these lists would be a second writer needing a lock, would
// stop when a container restarts, and the restaurant would find out on the
// morning every limited dish stayed off the menu. Reading the date costs
// nothing and cannot fail to run.
func (b *Branch) IsLimitSoldOut(id primitive.ObjectID) bool {
	if b.LimitDate != time.Now().Format("2006-01-02") {
		return false
	}
	return containsID(b.LimitSoldOut, id)
}

// IsStockSoldOut is the store's half alone.
//
// ⚠️ Separate for the same reason the till's is: a dish stopped because the
// shelf is empty cannot be put back with the counter's toggle — the next sync
// would stop it again within minutes, and a button that springs back with no
// explanation teaches the room that the panel lies. Putting it back means
// recording the delivery, or counting the shelf.
func (b *Branch) IsStockSoldOut(id primitive.ObjectID) bool {
	return containsID(b.StockSoldOut, id)
}

// IsPOSSoldOut is the till's half alone. The panel needs it separately: a dish
// stopped over there cannot be put back from here, and a toggle that pretends
// otherwise would spring back a minute later with no explanation.
func (b *Branch) IsPOSSoldOut(id primitive.ObjectID) bool {
	return containsID(b.POSSoldOut, id)
}

func containsID(list []primitive.ObjectID, id primitive.ObjectID) bool {
	for _, x := range list {
		if x == id {
			return true
		}
	}
	return false
}

// ---- Category ----

type Category struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	// Which brand's menu this belongs to. Empty on documents written before
	// brands existed — the migration fills them in.
	BrandID primitive.ObjectID `bson:"brandId,omitempty" json:"brandId,omitempty"`
	Name    string             `bson:"name" json:"name" validate:"required"` // uz (base)
	// Optional translations; empty means "fall back to the base name".
	NameRu    string `bson:"nameRu" json:"nameRu"`
	NameEn    string `bson:"nameEn" json:"nameEn"`
	Slug      string `bson:"slug" json:"slug"`
	SortOrder int    `bson:"sortOrder" json:"sortOrder"`
	IsActive  bool   `bson:"isActive" json:"isActive"`
	ImageURL  string `bson:"imageUrl" json:"imageUrl"`
}

// ---- Menu item ----

// OptionChoice is one selectable value inside a MenuOption (e.g. "Katta").
// PriceDelta is added to the dish price when the choice is selected; it may be
// negative. Uzbek is the base name, RU/EN are optional translations.
type OptionChoice struct {
	Name       string `bson:"name" json:"name"`
	NameRu     string `bson:"nameRu" json:"nameRu"`
	NameEn     string `bson:"nameEn" json:"nameEn"`
	PriceDelta int    `bson:"priceDelta" json:"priceDelta"`
	// What this choice alone takes out of the store, per portion.
	//
	// ⚠️ **The bar sells the same bottle in three sizes.** A vodka poured at
	// 40 ml, 50 ml and 100 ml is one dish with a "Hajm" group, and until this
	// existed every one of them took the dish's own recipe out of the store —
	// so a hundred 100 ml pours and a hundred 40 ml pours emptied the shelf by
	// exactly the same amount. The price already varied; only the stock did
	// not, which is the half nobody sees until the count.
	//
	// ⚠️ **Beside the dish's recipe, not instead of it.** A gin and tonic is
	// the tonic, the ice and the lemon whichever measure of gin goes in — that
	// is the dish's own card — plus the gin, which is the choice's. Folding
	// them into one would mean repeating the garnish on every size.
	//
	// ⚠️ Empty on every choice that is only a price: "katta"/"kichik" on a
	// pizza changes what is charged and nothing the store can measure, and a
	// card is not invented for it. Same rule as an empty dish recipe — nobody
	// has written this one down, not "this needs nothing".
	Recipe []RecipeLine `bson:"recipe,omitempty" json:"recipe,omitempty"`
}

// MenuOption is a group of choices attached to a dish (e.g. "Hajm", "Qo'shimcha").
// Required groups must be answered before the dish can be ordered; Multiple
// groups allow any number of choices (checkboxes) instead of exactly one.
type MenuOption struct {
	Name     string         `bson:"name" json:"name"`
	NameRu   string         `bson:"nameRu" json:"nameRu"`
	NameEn   string         `bson:"nameEn" json:"nameEn"`
	Required bool           `bson:"required" json:"required"`
	Multiple bool           `bson:"multiple" json:"multiple"`
	Choices  []OptionChoice `bson:"choices" json:"choices"`
}

// ComboLine is one dish inside a combo, with how many of it the set contains.
type ComboLine struct {
	MenuItemID primitive.ObjectID `bson:"menuItemId" json:"menuItemId"`
	Qty        int                `bson:"qty" json:"qty"`
}

// OrderComboLine is what a combo contained **at the time it was ordered**.
// Frozen by name and quantity: the set may be rebuilt or repriced next week,
// and the kitchen still has to be able to read last night's receipt.
type OrderComboLine struct {
	Name string `bson:"name" json:"name"`
	Qty  int    `bson:"qty" json:"qty"`
	// Which dish this line is, and what it cost on its own when the set was
	// sold. Neither is for the guest — the receipt shows a name and a quantity,
	// and that has not changed.
	//
	// They are here because the till needs them: a combo reaches iiko as its
	// member dishes (see posItems), and a name cannot be looked up in somebody
	// else's product list. The price is the weight the combo's own price is
	// split by, kept alongside the id so the split a shift manager queries next
	// week is the one that was actually sent, not one recomputed from a menu
	// that has been repriced since.
	//
	// Both are absent on orders taken before this existed, and the sender falls
	// back to the live combo definition for those.
	MenuItemID primitive.ObjectID `bson:"menuItemId,omitempty" json:"menuItemId,omitempty"`
	Price      int                `bson:"price,omitempty" json:"price,omitempty"`
}

// MenuItem is a dish or — when ComboItems is non-empty — a combo.
//
// A combo is deliberately *the same document type* rather than its own
// collection: it needs a picture, translations, a category, a sort order, a
// brand, an availability flag and a place in the cart, and every one of those
// already works for a dish. Making it a second kind of thing would mean
// teaching the cart, the order, the QR menu and the admin list about it twice.
type MenuItem struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BrandID     primitive.ObjectID `bson:"brandId,omitempty" json:"brandId,omitempty"`
	CategoryID  primitive.ObjectID `bson:"categoryId" json:"categoryId" validate:"required"`
	Name        string             `bson:"name" json:"name" validate:"required"` // uz (base)
	Description string             `bson:"description" json:"description"`
	// Optional translations; empty means "fall back to the base text".
	NameRu        string `bson:"nameRu" json:"nameRu"`
	NameEn        string `bson:"nameEn" json:"nameEn"`
	DescriptionRu string `bson:"descriptionRu" json:"descriptionRu"`
	DescriptionEn string `bson:"descriptionEn" json:"descriptionEn"`
	Price         int    `bson:"price" json:"price" validate:"gte=0"`
	// What the ingredients cost the restaurant, per portion, in whole so'm.
	//
	// ⚠️ **Optional, and zero means "not known" rather than "free".** Nothing
	// in this system can work a cost out — there are no recipes and no stock —
	// so it is a number the owner types, and most of them will type it for the
	// ten dishes that matter and never for the rest. Every screen that uses it
	// has to say how much of the menu it covers; a margin computed over the
	// dishes that happen to have one is a figure that looks like arithmetic and
	// is a guess.
	//
	// ⚠️ It is **not** on the public menu API. What a plate costs the kitchen
	// is the one number in this document that a competitor across the street
	// would pay for, and the restaurant profile goes to every visitor.
	Cost int `bson:"cost,omitempty" json:"-"`
	// The tech card: what goes into one portion.
	//
	// ⚠️ **When it is not empty it wins over `Cost`.** Two sources for one
	// number drift, and the drift is silent — a dish would be costed at the
	// figure somebody typed in March while its card says something else. The
	// panel shows the computed number and stops asking for the typed one.
	//
	// ⚠️ Also `json:"-"`: a recipe is a competitor's shopping list with the
	// quantities filled in, and the dish document goes to every visitor.
	Recipe      []RecipeLine `bson:"recipe,omitempty" json:"-"`
	OldPrice    *int         `bson:"oldPrice" json:"oldPrice"`
	ImageURL    string       `bson:"imageUrl" json:"imageUrl"`
	Images      []string     `bson:"images" json:"images"`
	IsAvailable bool         `bson:"isAvailable" json:"isAvailable"`
	IsPopular   bool         `bson:"isPopular" json:"isPopular"`
	SortOrder   int          `bson:"sortOrder" json:"sortOrder"`
	Options     []MenuOption `bson:"options" json:"options"`
	Tags        []string     `bson:"tags" json:"tags"`
	// ИКПУ — the state product classifier code, for the fiscal receipt.
	//
	// ⚠️ **Optional, and empty must stay empty.** The code comes from the
	// restaurant's own accountant; we cannot derive it from a dish name, and a
	// plausible guess is worse than nothing — a wrong ИКПУ is a wrong fiscal
	// receipt, which is the restaurant's problem with the tax office rather than
	// a formatting mistake. So it is sent when it is known and the field is
	// omitted entirely when it is not (see handlers/payatmos.go).
	//
	// ⚠️ Read from the menu at invoice time rather than frozen onto the order,
	// unlike the name and the price beside it. Those are what the guest agreed
	// to and must never move; this is a fact about the *product* in a state
	// classifier, so an accountant correcting a typo has to take effect on the
	// orders that have not been billed yet — a frozen copy would keep sending
	// the wrong code until every old order was gone.
	Ikpu string `bson:"ikpu,omitempty" json:"ikpu,omitempty"`
	// The packaging code that goes on the receipt beside the ИКПУ.
	//
	// ⚠️ **It belongs to the ИКПУ, not to the dish.** The classifier entry a
	// code names is sold in packagings, and the receipt carries both — so a
	// package code without a classifier code describes nothing, and the two are
	// cleared together (see normalizeIkpu's caller). Keeping it after the ИКПУ
	// was cleared would leave the menu holding a number that no longer refers
	// to anything, and it would look filled-in on the form.
	//
	// Empty is ordinary and stays empty, for the same reason the ИКПУ does.
	PackageCode string `bson:"packageCode,omitempty" json:"packageCode,omitempty"`
	// VAT rate for this dish, as a percentage — an *override*, not the rate.
	//
	// ⚠️ **A pointer, because 0 is a real answer.** Zero-rated and "nobody
	// filled this in" are different facts that a plain int cannot tell apart,
	// and here they disagree about money: a restaurant that is not a VAT payer
	// needs 0 on every line, while an unfilled field must fall back to the rate
	// the branch is registered at rather than silently declaring an exemption.
	// This is the one place in the codebase where the usual "zero value is
	// today's behaviour" rule cannot be used, so it is written out instead.
	//
	// ⚠️ **The rate itself lives in the fiscal settings**, not here. Typing 12
	// onto two hundred dishes by hand is how a menu ends up with a handful of
	// them saying 1 or 120, and the mistake is invisible until an inspector
	// finds it. This field exists for the exceptions a real menu has (a
	// zero-rated item beside standard ones), so most restaurants leave every
	// dish empty and set the rate once.
	VatPercent *int `bson:"vatPercent,omitempty" json:"vatPercent,omitempty"`
	// Unit of measure code from the state classifier: 0 = piece, 10 = gram,
	// 11 = kilogram, 22 = metre, 41 = litre.
	//
	// The zero value is "piece", which is what a portion is — so the ordinary
	// restaurant never touches this and every existing dish is already right
	// (the same reasoning as an empty mapProvider meaning 2GIS). It is here for
	// the menus that sell by weight: a cake by the kilogram, a draught drink by
	// the litre.
	UnitCode int `bson:"unitCode,omitempty" json:"unitCode,omitempty"`
	// Dishes to suggest alongside this one, chosen by hand.
	//
	// ⚠️ **Beside the automatic suggestions, not instead of them.** What sells
	// together is a fact and the order history knows it better than anybody;
	// but a new dish has no history at all, and the one thing an owner most
	// wants to push is the thing nobody has ordered yet. An automatic-only
	// feature can never promote anything new, which is the opposite of what an
	// owner would use it for.
	//
	// Empty is the normal state and means "work it out from the orders".
	RecommendedIDs []primitive.ObjectID `bson:"recommendedIds,omitempty" json:"recommendedIds,omitempty"`
	UpdatedAt      time.Time            `bson:"updatedAt" json:"updatedAt"`

	// Non-empty makes this a combo: a fixed set sold for Price. The dishes are
	// referenced, not copied, so renaming one renames it everywhere.
	ComboItems []ComboLine `bson:"comboItems,omitempty" json:"comboItems,omitempty"`

	// Not stored: filled in per request from the serving branch's SoldOut list
	// (see Branch.SoldOut). "Not available anywhere" is IsAvailable; "the pot is
	// empty at this branch today" is this. A combo is sold out as soon as any
	// one of its dishes is — half a set is not a set.
	SoldOut bool `bson:"-" json:"soldOut,omitempty"`

	// Not stored: what the same dishes would cost bought separately, and the
	// names to show under the combo card. Computed per request so the saving
	// stays true after a member dish is repriced — a stored copy would quietly
	// start lying.
	ComboBasePrice int              `bson:"-" json:"comboBasePrice,omitempty"`
	ComboContents  []OrderComboLine `bson:"-" json:"comboContents,omitempty"`
}

// IsCombo reports whether this menu item is a set rather than a single dish.
func (m *MenuItem) IsCombo() bool { return len(m.ComboItems) > 0 }

// ---- Order ----

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusConfirmed OrderStatus = "confirmed"
	StatusPreparing OrderStatus = "preparing"
	StatusOnTheWay  OrderStatus = "on_the_way"
	StatusDelivered OrderStatus = "delivered"
	StatusCancelled OrderStatus = "cancelled"
)

type OrderCustomer struct {
	Name  string `bson:"name" json:"name" validate:"required"`
	Phone string `bson:"phone" json:"phone" validate:"required"`
}

type OrderAddress struct {
	Text    string  `bson:"text" json:"text"`
	Lat     float64 `bson:"lat" json:"lat"`
	Lng     float64 `bson:"lng" json:"lng"`
	Comment string  `bson:"comment" json:"comment"`
}

// OrderItemOption is a selected choice, frozen at order time with the price
// delta that was applied — the menu may change later. Names are always the base
// (uz) ones so the admin panel reads in a single language.
type OrderItemOption struct {
	Name       string `bson:"name" json:"name"`     // option group
	Choice     string `bson:"choice" json:"choice"` // selected choice
	PriceDelta int    `bson:"priceDelta" json:"priceDelta"`
}

type OrderItem struct {
	MenuItemID primitive.ObjectID `bson:"menuItemId" json:"menuItemId"`
	Name       string             `bson:"name" json:"name"`
	// Unit price actually charged: dish price + the selected option deltas.
	Price   int               `bson:"price" json:"price"`
	Qty     int               `bson:"qty" json:"qty" validate:"gte=1"`
	Options []OrderItemOption `bson:"options" json:"options"`
	// Free text from the customer for this dish alone ("no onion", "extra
	// spicy"). Shown to the kitchen and on the receipt.
	Comment string `bson:"comment,omitempty" json:"comment,omitempty"`
	// What a combo contained, frozen at order time. The kitchen cooks from the
	// receipt, so "Oilaviy combo" alone would be an instruction it cannot follow.
	ComboItems []OrderComboLine `bson:"comboItems,omitempty" json:"comboItems,omitempty"`

	// ---- Till lines only (an open check). Absent on every other order. ----

	// Stable identity for one line of an open check.
	//
	// ⚠️ Needed because a check is **edited**, and nothing else here can name a
	// line: two guests ordering the same dish with the same options are one
	// line on a website (quantity two) and must stay two lines at a table, one
	// of which may be sent back. Position in the slice cannot do the job — the
	// waiter's tablet and the cashier's screen edit the same check seconds
	// apart, and an index shifts under the other one's feet.
	LineID string `bson:"lineId,omitempty" json:"lineId,omitempty"`

	// When this line was sent to the kitchen.
	//
	// ⚠️ **Per line, not per check** — this is the whole difference between a
	// till and the website's order form. Courses are fired as the meal goes:
	// starters now, mains when the table has finished them. A check-wide flag
	// would force the waiter to either send the whole dinner at once or open a
	// second check for the mains, and both are things real dining rooms refuse
	// to do.
	//
	// The kitchen screen shows fired lines and nothing else, so an unfired line
	// is invisible to the pass — the same rule `queuedAt` applies to whole
	// orders, one level down.
	FiredAt *time.Time `bson:"firedAt,omitempty" json:"firedAt,omitempty"`

	// Set when a line that had already been fired was taken off the check. The
	// line **stays on the document**: the food was cooked, somebody paid for it
	// in ingredients, and a void that leaves no trace is the oldest way to take
	// money out of a restaurant.
	Void *CheckLineVoid `bson:"void,omitempty" json:"void,omitempty"`

	// Which guest at the table this is for.
	//
	// ⚠️ **Zero means the table**, not "guest zero", and that is what keeps
	// every check written before this existed correct: an order nobody split is
	// one bill for the party, which is how most of them end.
	//
	// The number is the seat as the waiter counted them, not an identity — it
	// exists so a table of four can be handed four bills without the waiter
	// remembering who had the lamb. Splitting is decided at the **end** of a
	// meal, which is why it is the one thing that may still be changed after a
	// line has gone to the kitchen.
	Guest int `bson:"guest,omitempty" json:"guest,omitempty"`

	// Which course this dish belongs to: starters, mains, dessert.
	//
	// ⚠️ **Zero means "with everything else"** — the behaviour every check had
	// before courses existed, and still the right one for a counter selling
	// coffee. A restaurant that never numbers a course never sees the feature.
	//
	// ⚠️ A course is a **plan**, not a state: it says when the waiter intends to
	// send this, and `FiredAt` says whether they have. Storing "course 2 is
	// away" on the check instead would be a second place to be wrong about
	// something the lines already know.
	Course int `bson:"course,omitempty" json:"course,omitempty"`
}

// Live reports whether this line still counts — towards the bill, the kitchen
// and every total. A voided line is kept for the audit and counts for nothing.
func (i OrderItem) Live() bool { return i.Void == nil }

// ---- External delivery providers ----

// DeliveryProvider is an outside delivery service (Yandex Delivery, a taxi
// company, a door-to-door courier firm...) that a restaurant without its own
// couriers can hand an order to.
//
// Kind decides what the admin panel does with it:
//
//	"link"  — open URL with the order details substituted into it
//	"phone" — call the dispatcher and read out the details
//	"api"   — the server files the delivery request itself (see internal/delivery)
//
// Only providers that publish an API can use "api". Everything else still works
// through a link or a phone call, which is how most local services operate.
type DeliveryProvider struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name" validate:"required"`
	Kind      string             `bson:"kind" json:"kind"` // "link" | "phone" | "api"
	URL       string             `bson:"url" json:"url"`
	Phone     string             `bson:"phone" json:"phone"`
	Note      string             `bson:"note" json:"note"`
	IsActive  bool               `bson:"isActive" json:"isActive"`
	SortOrder int                `bson:"sortOrder" json:"sortOrder"`

	// ---- Kind "api" ----
	// Which integration to use. Currently only "yandex" (Yandex Delivery B2B).
	APIProvider string `bson:"apiProvider" json:"apiProvider"`
	// OAuth token from the provider's dashboard. Never sent to the browser.
	APIToken string `bson:"apiToken" json:"-"`
	// True when a token is stored, so the admin form can show "configured"
	// without ever exposing the secret.
	HasToken bool `bson:"-" json:"hasToken"`
	// Override for sandbox/testing. Empty = the provider's production host.
	APIBaseURL string `bson:"apiBaseUrl" json:"apiBaseUrl"`
	// Tariff/vehicle class, e.g. "express" or "courier" for Yandex.
	APITariff string `bson:"apiTariff" json:"apiTariff"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// ExternalDelivery records that an order was handed to an outside service, so
// the panel can show who is carrying it and when they were called.
type ExternalDelivery struct {
	ProviderID   primitive.ObjectID `bson:"providerId,omitempty" json:"providerId,omitempty"`
	ProviderName string             `bson:"providerName" json:"providerName"`
	TrackingID   string             `bson:"trackingId" json:"trackingId"`
	Note         string             `bson:"note" json:"note"`
	CalledAt     time.Time          `bson:"calledAt" json:"calledAt"`
	// What the service charged, entered by hand for link/phone providers (the
	// API ones report it themselves in Price). 0 = not recorded.
	Cost int `bson:"cost,omitempty" json:"cost,omitempty"`

	// ---- Filled in when the provider has a real API ----
	ClaimID      string    `bson:"claimId,omitempty" json:"claimId,omitempty"`
	Status       string    `bson:"status,omitempty" json:"status,omitempty"`
	Price        string    `bson:"price,omitempty" json:"price,omitempty"`
	CourierName  string    `bson:"apiCourierName,omitempty" json:"courierName,omitempty"`
	CourierPhone string    `bson:"apiCourierPhone,omitempty" json:"courierPhone,omitempty"`
	TrackURL     string    `bson:"trackUrl,omitempty" json:"trackUrl,omitempty"`
	SyncedAt     time.Time `bson:"syncedAt,omitempty" json:"syncedAt,omitempty"`
}

// ---- Courier ----

// CourierStatus is what the dispatcher sees at a glance.
//
//	off   — not on shift
//	free  — on shift, available
//	busy  — on shift, currently delivering
type CourierStatus string

const (
	CourierOff  CourierStatus = "off"
	CourierFree CourierStatus = "free"
	CourierBusy CourierStatus = "busy"
)

// CourierLocation is the last position reported by the courier app.
type CourierLocation struct {
	Lat      float64   `bson:"lat" json:"lat"`
	Lng      float64   `bson:"lng" json:"lng"`
	Accuracy float64   `bson:"accuracy" json:"accuracy"`
	At       time.Time `bson:"at" json:"at"`
}

// CourierPayout decides what a courier earns per delivered order:
//
//	"deliveryFee" (default) — the delivery fee the customer paid
//	"perOrder"              — a flat PayoutPerOrder amount
//	"percent"               — PayoutPercent % of the delivery fee
//
// The restaurant picks this per courier, because rates differ per person.
type CourierPayout string

const (
	PayoutDeliveryFee CourierPayout = "deliveryFee"
	PayoutPerOrder    CourierPayout = "perOrder"
	PayoutPercent     CourierPayout = "percent"
)

// Courier accounts are created by hand in the admin panel — there is no
// self-signup. The password is only ever stored hashed.
type Courier struct {
	BranchID     primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name         string             `bson:"name" json:"name" validate:"required"`
	Phone        string             `bson:"phone" json:"phone"`
	Username     string             `bson:"username" json:"username" validate:"required"`
	PasswordHash string             `bson:"passwordHash" json:"-"`
	Status       CourierStatus      `bson:"status" json:"status"`
	Vehicle      string             `bson:"vehicle" json:"vehicle"` // "moto" | "car" | "bike" | ""
	IsActive     bool               `bson:"isActive" json:"isActive"`
	Location     *CourierLocation   `bson:"location,omitempty" json:"location,omitempty"`

	// Payout rules — see CourierPayout. An empty mode reads as "deliveryFee".
	PayoutMode     CourierPayout `bson:"payoutMode" json:"payoutMode"`
	PayoutPerOrder int           `bson:"payoutPerOrder" json:"payoutPerOrder"`
	PayoutPercent  int           `bson:"payoutPercent" json:"payoutPercent"`
	CreatedAt      time.Time     `bson:"createdAt" json:"createdAt"`
	UpdatedAt      time.Time     `bson:"updatedAt" json:"updatedAt"`
}

// CourierSettlement is the courier handing collected cash back to the
// restaurant.
//
// Without this the "cash in hand" figure is really "cash ever collected": it
// only ever grows, and by the second week it tells nobody anything. A courier
// who has handed over everything should read zero.
type CourierSettlement struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CourierID primitive.ObjectID `bson:"courierId" json:"courierId"`
	Amount    int                `bson:"amount" json:"amount"`
	// Who took the money. A cash handover with no name on it is exactly the
	// record that gets disputed later.
	TakenBy string    `bson:"takenBy" json:"takenBy"`
	Note    string    `bson:"note,omitempty" json:"note,omitempty"`
	At      time.Time `bson:"at" json:"at"`
}

// ---- Cash: the till, and what it should contain ----

// CashShift is one till session: opened with a float, closed with a count.
//
// ⚠️ **The point of this record is the difference, not the total.** A system
// that shows what the till *should* hold and lets somebody type what it *does*
// hold, then quietly stores the second number, has recorded nothing: the
// shortfall it existed to surface has been overwritten by the person who might
// have caused it. So Expected is frozen at closing time, Counted is what was
// counted, and Variance is stored rather than derived — a later change to how
// expected cash is computed must not silently rewrite last month's shortfalls.
type CashShift struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`

	OpenedAt   time.Time          `bson:"openedAt" json:"openedAt"`
	OpenedBy   string             `bson:"openedBy" json:"openedBy"`
	OpenedByID primitive.ObjectID `bson:"openedById,omitempty" json:"-"`
	// The change left in the drawer to start with. Counted as cash on hand,
	// not as takings — it was already the restaurant's money.
	OpeningFloat int `bson:"openingFloat" json:"openingFloat"`

	ClosedAt *time.Time `bson:"closedAt,omitempty" json:"closedAt,omitempty"`
	ClosedBy string     `bson:"closedBy,omitempty" json:"closedBy,omitempty"`
	// What the till should have held, frozen at the moment of closing.
	Expected int `bson:"expected" json:"expected"`
	// Cash taken at the counter during this shift, frozen alongside Expected.
	//
	// ⚠️ **Stored so the register's own figure has something to be compared
	// against.** Expected is the whole drawer — float, counter takings, courier
	// handovers, manual movements — while the fiscal register only knows about
	// cash *sales*. Comparing the register against Expected looks like a
	// comparison and is arithmetic nonsense; this is the one component that
	// answers the same question the register does.
	CounterCash int `bson:"counterCash" json:"counterCash"`
	// What was actually in the drawer.
	Counted int `bson:"counted" json:"counted"`
	// Counted − Expected. Negative is a shortfall.
	Variance int `bson:"variance" json:"variance"`
	// Why, when it does not match. **Required for a non-zero variance**: an
	// unexplained shortfall recorded without a sentence is one nobody can act
	// on a week later, and the person who could explain it has gone home.
	VarianceNote string `bson:"varianceNote,omitempty" json:"varianceNote,omitempty"`

	Note string `bson:"note,omitempty" json:"note,omitempty"`
	// What the cash register totalled for the same day, once it has answered.
	// ⚠️ A second, independent count — never used to correct `Expected`, which
	// is frozen on purpose. See FiscalDay.
	Fiscal    *FiscalDay `bson:"fiscal,omitempty" json:"fiscal,omitempty"`
	CreatedAt time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time  `bson:"updatedAt" json:"updatedAt"`
}

// Open reports whether this shift is still running.
func (s *CashShift) Open() bool { return s.ClosedAt == nil }

// Cash movements that are not an order and not a courier handover.
const (
	CashIn  = "in"
	CashOut = "out"
)

// CashEntry is money put into or taken out of the till by hand.
//
// Every one carries a name and a reason, for the same reason a courier
// settlement does: cash that moved with neither is the entry that becomes an
// argument three weeks later, and by then nobody remembers.
type CashEntry struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	ShiftID  primitive.ObjectID `bson:"shiftId" json:"shiftId"`
	// "in" | "out".
	Kind string `bson:"kind" json:"kind"`
	// Free text chosen by the restaurant ("mahsulot", "avans", "inkassatsiya").
	// Not an enum: every kitchen spends money on something the next one does
	// not, and a fixed list would send all of it to "boshqa".
	Category string             `bson:"category" json:"category"`
	Amount   int                `bson:"amount" json:"amount"`
	Note     string             `bson:"note,omitempty" json:"note,omitempty"`
	ByID     primitive.ObjectID `bson:"byId,omitempty" json:"-"`
	By       string             `bson:"by" json:"by"`
	At       time.Time          `bson:"at" json:"at"`
}

// StatusEvent records when an order moved to a status — used by the admin
// panel to answer "when exactly was this order confirmed / delivered?".
type StatusEvent struct {
	Status OrderStatus `bson:"status" json:"status"`
	At     time.Time   `bson:"at" json:"at"`
}

type Order struct {
	// The id the till gave this sale before the server ever saw it.
	//
	// ⚠️ **The whole of offline safety is this field.** A till that took an
	// order with no network holds it on its own disk and sends it when the
	// connection returns — and the send is retried, by a program that cannot
	// know whether the first attempt arrived. Without an id minted by the
	// till, a retry is a second dinner: charged twice, counted twice in the
	// day's takings, and cooked twice if the kitchen screen is watching.
	//
	// Sparse and unique: every sale rung up online has none, and they are the
	// overwhelming majority. Same pattern as the delivery provider's
	// request_id, for the same reason.
	ClientID string `bson:"clientId,omitempty" json:"clientId,omitempty"`

	// Which brand's menu this was ordered from and which branch cooks it.
	BrandID  primitive.ObjectID `bson:"brandId,omitempty" json:"brandId,omitempty"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID   primitive.ObjectID `bson:"userId,omitempty" json:"userId,omitempty"`
	// Assigned courier (delivery orders). Name is denormalised so the receipt
	// still reads correctly if the courier account is later removed.
	CourierID   primitive.ObjectID `bson:"courierId,omitempty" json:"courierId,omitempty"`
	CourierName string             `bson:"courierName,omitempty" json:"courierName,omitempty"`
	// Set instead of a courier when an outside service carries the order.
	ExternalDelivery *ExternalDelivery `bson:"externalDelivery,omitempty" json:"externalDelivery,omitempty"`
	Number           string            `bson:"number" json:"number"`
	Status           OrderStatus       `bson:"status" json:"status"`
	Customer         OrderCustomer     `bson:"customer" json:"customer"`
	Type             string            `bson:"type" json:"type"` // "delivery" | "pickup" | "dinein"
	// Dine-in only: which table the guest scanned. The number is a snapshot —
	// the floor plan may be redrawn, the receipt must still read correctly.
	TableID     string       `bson:"tableId,omitempty" json:"tableId,omitempty"`
	TableNumber string       `bson:"tableNumber,omitempty" json:"tableNumber,omitempty"`
	Address     OrderAddress `bson:"address" json:"address"`
	Items       []OrderItem  `bson:"items" json:"items"`
	Subtotal    int          `bson:"subtotal" json:"subtotal"`
	// Every discount that was applied, in the order it was applied. The receipt
	// shows them as separate lines: one lump "−27 000" is unanswerable.
	Discounts     []OrderDiscount `bson:"discounts,omitempty" json:"discounts,omitempty"`
	DiscountTotal int             `bson:"discountTotal,omitempty" json:"discountTotal,omitempty"`
	// Points the guest put towards this order, and the cashback it earned once
	// it was delivered. Both frozen here so the receipt explains itself and the
	// refund on a cancellation knows exactly what to undo.
	PointsSpent   int     `bson:"pointsSpent,omitempty" json:"pointsSpent,omitempty"`
	PointsEarned  int     `bson:"pointsEarned,omitempty" json:"pointsEarned,omitempty"`
	DeliveryFee   int     `bson:"deliveryFee" json:"deliveryFee"`
	Total         int     `bson:"total" json:"total"`
	PaymentMethod string  `bson:"paymentMethod" json:"paymentMethod"`
	DeliveryZone  string  `bson:"deliveryZone" json:"deliveryZone"`
	DistanceKm    float64 `bson:"distanceKm" json:"distanceKm"`
	// What the room added for service, in so'm, frozen at the moment the check
	// was closed.
	//
	// ⚠️ **A copied amount, never a percentage recomputed later.** The rate is
	// a branch setting that changes; the bill the guest agreed to does not, and
	// a receipt reprinted next month has to say what they paid. Same reason
	// every discount is copied onto the order by name and amount.
	ServiceCharge int `bson:"serviceCharge,omitempty" json:"serviceCharge,omitempty"`
	// The rate that produced it, so the receipt can say "10%" rather than a
	// number the guest has to divide.
	ServicePercent int `bson:"servicePercent,omitempty" json:"servicePercent,omitempty"`

	// Set on a check that was joined onto another one. ⚠️ The document stays
	// (cancelled) rather than being deleted: it carries voided lines, a number
	// that may be on a printed bill, and who opened it.
	MergedIntoID primitive.ObjectID `bson:"mergedIntoId,omitempty" json:"mergedIntoId,omitempty"`

	// What was said at the counter when a check was left as a debt.
	//
	// ⚠️ On the order rather than in a separate ledger, because the debt **is**
	// this sale: one document to chase, one to mark paid, and no second place
	// for the two to disagree about the amount.
	DebtNote string `bson:"debtNote,omitempty" json:"debtNote,omitempty"`

	// Money handed back after the sale was closed. ⚠️ The sale stays; see
	// CheckRefund.
	Refund *CheckRefund `bson:"refund,omitempty" json:"refund,omitempty"`

	StatusHistory []StatusEvent `bson:"statusHistory" json:"statusHistory"`
	// Why the restaurant cancelled it. The customer sees this on the tracking
	// page, so "why was my order cancelled?" never needs a phone call.
	CancelReason string `bson:"cancelReason,omitempty" json:"cancelReason,omitempty"`
	// How the money stands. Cash orders are "unpaid" for their whole life and
	// that is not a problem; an online order starts "pending" and only the
	// provider's own callback moves it to "paid". Empty on orders written
	// before online payment existed, which reads as "unpaid" — the same thing
	// those orders always were.
	PaymentStatus string     `bson:"paymentStatus,omitempty" json:"paymentStatus,omitempty"`
	PaidAt        *time.Time `bson:"paidAt,omitempty" json:"paidAt,omitempty"`
	// When this order became the kitchen's problem. The same instant as
	// CreatedAt for cash, and the moment the bank confirmed for an online one.
	//
	// It exists because the two are genuinely different events, and the wrong
	// one was being used: the new-order chime keys off this, so without it the
	// kitchen is called to a bill that may never be paid — and is *not* called
	// when the money finally lands, because by then the order is minutes old.
	QueuedAt *time.Time `bson:"queuedAt,omitempty" json:"queuedAt,omitempty"`
	// When the guest asked for it. Nil on an ordinary order, which is "now" and
	// always has been — the field only exists for the ones that are not.
	//
	// ⚠️ **It does not schedule anything by itself.** What keeps a pre-order out
	// of the kitchen is `QueuedAt`, set to this time minus the branch's lead:
	// the pass, the chime and the "waiting" counts all read that one timestamp
	// and none of them had to learn a second kind of order. This field is the
	// promise made to the guest — what the receipt, the tracking page and the
	// courier's screen say out loud — and the input the lead is subtracted from.
	ScheduledAt *time.Time `bson:"scheduledAt,omitempty" json:"scheduledAt,omitempty"`
	// When the kitchen said "this one is done".
	//
	// ⚠️ **A timestamp, deliberately not a new status.** "Ready" sits between
	// preparing and on_the_way for a delivery, but for pickup and dine-in it
	// sits somewhere else entirely, and a status is read by the courier app,
	// the customer's tracking page, the statistics, the POS bridge and three
	// dictionaries. Adding one would have meant touching every one of them to
	// express a fact only the kitchen and the counter care about.
	//
	// As a timestamp it composes instead: the ticket leaves the kitchen screen,
	// the order list shows "tayyor", and nothing that reasons about status
	// changes at all. Cleared when an order is pushed back to an earlier stage,
	// because a ticket that returns to the kitchen is not ready any more.
	ReadyAt *time.Time `bson:"readyAt,omitempty" json:"readyAt,omitempty"`
	// The operator who took this order over the phone, by name. Absent on
	// orders the guest placed themselves, which is what makes it useful: it
	// answers "did somebody type this in, and who?" without a second lookup.
	TakenBy string `bson:"takenBy,omitempty" json:"takenBy,omitempty"`
	// Which door the order came in through: "web", "telegram" or "operator".
	//
	// ⚠️ **Attribution, not authorisation** — and the distinction is what makes a
	// browser-sent value acceptable here. The mini app *is* the site, rendered in
	// Telegram's WebView, so there is nothing on the server that can tell the two
	// apart: no header, no address, no session difference. The client says which
	// it is, the server narrows it to the enum, and a guest who lies about it
	// mislabels their own order and changes nothing else.
	//
	// "operator" is the exception and is set **server-side**, from the fact that
	// an admin session created the order — that one is never taken on trust,
	// because it is the answer to "who typed this wrong address in?".
	//
	// Worth storing at all because the two channels answer a question the owner
	// cannot otherwise ask: a restaurant paying for a bot wants to know whether
	// anybody orders through it, and "half our orders come from Telegram" and
	// "nobody has ever used it" lead to opposite decisions.
	Channel string `bson:"channel,omitempty" json:"channel,omitempty"`
	// What happened when this order was pushed to the restaurant's till.
	// Absent when no POS is connected, which is most installs.
	POS *OrderPOS `bson:"pos,omitempty" json:"pos,omitempty"`
	// Set only on a sale rung up on our own till. Absent on website, bot and
	// operator orders, which is what makes it a usable filter for "what is open
	// in the dining room right now".
	Check *OrderCheck `bson:"check,omitempty" json:"check,omitempty"`
	// Whether this sale was registered with the tax committee, and what came
	// back. Absent on every order that does not need one — a website order paid
	// by card is fiscalised by the payment provider (see payatmos.go), and an
	// unpaid order is not a sale at all.
	Fiscal    *FiscalReceipt `bson:"fiscal,omitempty" json:"fiscal,omitempty"`
	CreatedAt time.Time      `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time      `bson:"updatedAt" json:"updatedAt"`
}

// LiveItems returns the lines that still count — everything except voided ones.
// Used everywhere a total is computed, so a void can never be forgotten in one
// place and honoured in another.
func (o *Order) LiveItems() []OrderItem {
	out := make([]OrderItem, 0, len(o.Items))
	for _, it := range o.Items {
		if it.Live() {
			out = append(out, it)
		}
	}
	return out
}

// ---- Discounts: promo codes and campaigns ----
//
// One document type, two triggers. A promo code and an "aksiya" are the same
// thing to the kitchen and to the till — a rule that takes money off an order —
// and differ only in what sets them off: a code the guest types, or simply the
// day and time. Splitting them into two features would mean writing the same
// validity window, the same caps and the same receipt line twice, and then
// discovering they disagree.
type PromotionTrigger string

const (
	// The guest types a code at checkout.
	TriggerCode PromotionTrigger = "code"
	// Applies by itself whenever its conditions are met.
	TriggerAuto PromotionTrigger = "auto"
)

type PromotionKind string

const (
	PromoPercent      PromotionKind = "percent"
	PromoFixed        PromotionKind = "fixed"
	PromoFreeDelivery PromotionKind = "freeDelivery"
)

// PromotionScope decides what the discount is calculated on.
type PromotionScope string

const (
	// The whole order.
	ScopeOrder PromotionScope = "order"
	// Only lines in the listed categories.
	ScopeCategory PromotionScope = "category"
	// Only the listed dishes.
	ScopeItems PromotionScope = "items"
)

type Promotion struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BrandID primitive.ObjectID `bson:"brandId,omitempty" json:"brandId,omitempty"`
	// Empty = every branch of the brand. A campaign may be one branch's own.
	BranchIDs []primitive.ObjectID `bson:"branchIds" json:"branchIds"`

	// Shown to the guest on the receipt, so it has to read as a reason:
	// "Osh kuni −10%", not "promo3".
	Name    string           `bson:"name" json:"name" validate:"required"`
	NameRu  string           `bson:"nameRu" json:"nameRu"`
	NameEn  string           `bson:"nameEn" json:"nameEn"`
	Trigger PromotionTrigger `bson:"trigger" json:"trigger"`
	// Uppercase, unique per brand. Only meaningful for TriggerCode.
	Code string `bson:"code" json:"code"`

	Kind  PromotionKind `bson:"kind" json:"kind"`
	Value int           `bson:"value" json:"value"` // percent (1..100) or UZS
	// Ceiling for a percentage discount. 0 = no ceiling. Without it "20% off"
	// on a banquet order is an unbounded promise.
	MaxDiscount int `bson:"maxDiscount" json:"maxDiscount"`
	// The order must be at least this much *before* discounts to qualify.
	MinOrder int `bson:"minOrder" json:"minOrder"`

	Scope       PromotionScope       `bson:"scope" json:"scope"`
	CategoryIDs []primitive.ObjectID `bson:"categoryIds" json:"categoryIds"`
	MenuItemIDs []primitive.ObjectID `bson:"menuItemIds" json:"menuItemIds"`

	// ---- When it is live ----
	StartsAt *time.Time `bson:"startsAt,omitempty" json:"startsAt,omitempty"`
	EndsAt   *time.Time `bson:"endsAt,omitempty" json:"endsAt,omitempty"`
	// Weekdays it runs on (0=Sunday). Empty = every day.
	Days []int `bson:"days" json:"days"`
	// Time of day, "HH:MM". Both empty = all day. TimeTo before TimeFrom means
	// it runs overnight, the same rule the working hours use.
	TimeFrom string `bson:"timeFrom" json:"timeFrom"`
	TimeTo   string `bson:"timeTo" json:"timeTo"`
	// delivery / pickup / dinein. Empty = all of them.
	OrderTypes []string `bson:"orderTypes" json:"orderTypes"`

	// ---- Limits (codes) ----
	// Total redemptions allowed, 0 = unlimited.
	UsageLimit int `bson:"usageLimit" json:"usageLimit"`
	// Redemptions allowed per customer, 0 = unlimited.
	PerUserLimit int `bson:"perUserLimit" json:"perUserLimit"`
	// Only for a customer who has never ordered before.
	FirstOrderOnly bool `bson:"firstOrderOnly" json:"firstOrderOnly"`
	UsedCount      int  `bson:"usedCount" json:"usedCount"`

	IsActive  bool      `bson:"isActive" json:"isActive"`
	SortOrder int       `bson:"sortOrder" json:"sortOrder"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`

	// Computed per request, never stored: what this rule is actually doing
	// right now. "Active" on a code that expired last month is a lie the panel
	// used to tell — the owner would keep advertising it.
	Status string `bson:"-" json:"status,omitempty"`
}

// PromotionStatus values, in the order the panel checks them.
const (
	// Switched off by hand.
	PromoStatusOff = "off"
	// Its start date has not arrived.
	PromoStatusScheduled = "scheduled"
	// Past its end date.
	PromoStatusExpired = "expired"
	// Every allowed redemption has been taken.
	PromoStatusUsedUp = "usedUp"
	// Live, but not right now — wrong day or wrong hour.
	PromoStatusIdle = "idle"
	// Running.
	PromoStatusRunning = "running"
)

// StatusAt reports what a rule is doing at a moment in time.
func (p *Promotion) StatusAt(now time.Time) string {
	switch {
	case !p.IsActive:
		return PromoStatusOff
	case p.StartsAt != nil && now.Before(*p.StartsAt):
		return PromoStatusScheduled
	case p.EndsAt != nil && now.After(*p.EndsAt):
		return PromoStatusExpired
	case p.UsageLimit > 0 && p.UsedCount >= p.UsageLimit:
		return PromoStatusUsedUp
	}
	// Inside its dates but possibly outside today's window.
	if PromotionInWindow(p, now) {
		return PromoStatusRunning
	}
	return PromoStatusIdle
}

// PromotionInWindow answers the "today and now" half of a rule's validity: the
// weekdays it runs on and the hours within them. Dates and limits are checked
// separately, because they mean different things to the owner — "expired" and
// "not on Mondays" are not the same answer.
func PromotionInWindow(p *Promotion, now time.Time) bool {
	if len(p.Days) > 0 {
		day := int(now.Weekday())
		found := false
		for _, d := range p.Days {
			if d == day {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if p.TimeFrom != "" && p.TimeTo != "" {
		cur := now.Format("15:04")
		if p.TimeTo < p.TimeFrom { // overnight, e.g. 22:00–02:00
			if cur < p.TimeFrom && cur > p.TimeTo {
				return false
			}
		} else if cur < p.TimeFrom || cur > p.TimeTo {
			return false
		}
	}
	return true
}

// OrderDiscount is one discount as it was applied, frozen onto the order.
//
// Kept by name and amount rather than by reference alone: a campaign gets
// renamed, repriced or deleted, and last week's receipt must still explain
// itself. "Why is this 12 000 less?" is a question the receipt has to answer on
// its own.
type OrderDiscount struct {
	PromotionID primitive.ObjectID `bson:"promotionId,omitempty" json:"promotionId,omitempty"`
	Name        string             `bson:"name" json:"name"`
	Kind        PromotionKind      `bson:"kind" json:"kind"`
	Trigger     PromotionTrigger   `bson:"trigger" json:"trigger"`
	// The code the guest actually typed, for a support call later.
	Code string `bson:"code,omitempty" json:"code,omitempty"`
	// What it took off. Delivery discounts record the fee they removed.
	Amount int `bson:"amount" json:"amount"`
}

// ---- Loyalty: cashback points ----
//
// One point is one so'm. Kept that way on purpose: "you have 6 000 points" and
// "that is 6 000 so'm off" are the same sentence, so nobody has to learn a
// conversion rate to know what their balance is worth.
type LoyaltySettings struct {
	Enabled bool `bson:"enabled" json:"enabled"`
	// Percentage of what the guest actually paid that comes back as points.
	EarnPercent int `bson:"earnPercent" json:"earnPercent"`
	// Orders below this earn nothing.
	MinOrderToEarn int `bson:"minOrderToEarn" json:"minOrderToEarn"`
	// The ceiling on how much of one order points may cover, as a percentage.
	// Without it a large balance turns an order into a free one, and the
	// kitchen still has to cook it.
	MaxRedeemPercent int `bson:"maxRedeemPercent" json:"maxRedeemPercent"`
	// Points handed to a customer the first time they sign in. 0 = none.
	WelcomePoints int `bson:"welcomePoints" json:"welcomePoints"`
}

// LoyaltyKind is why a balance moved.
type LoyaltyKind string

const (
	// Cashback for a completed order.
	LoyaltyEarn LoyaltyKind = "earn"
	// Paid part of an order with points.
	LoyaltySpend LoyaltyKind = "spend"
	// An order was cancelled: points spent come back, points earned go away.
	LoyaltyRevoke LoyaltyKind = "revoke"
	// Given by hand from the panel, or on sign-up.
	LoyaltyAdjust LoyaltyKind = "adjust"
)

// LoyaltyTxn is one movement of a customer's balance.
//
// The balance itself is kept on the user for speed, but this is the record: a
// number nobody can explain is a number the customer will not trust, and
// "where did my 6 000 go?" has to be answerable months later.
type LoyaltyTxn struct {
	ID     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID primitive.ObjectID `bson:"userId" json:"userId"`
	// The order that caused it, when there was one.
	OrderID     primitive.ObjectID `bson:"orderId,omitempty" json:"orderId,omitempty"`
	OrderNumber string             `bson:"orderNumber,omitempty" json:"orderNumber,omitempty"`
	Kind        LoyaltyKind        `bson:"kind" json:"kind"`
	// Signed: positive adds to the balance, negative takes away.
	Points int `bson:"points" json:"points"`
	// The balance right after this movement, so a statement reads like one.
	BalanceAfter int       `bson:"balanceAfter" json:"balanceAfter"`
	Note         string    `bson:"note,omitempty" json:"note,omitempty"`
	At           time.Time `bson:"at" json:"at"`
}

// ---- Feedback and complaints ----
//
// One rating per delivered order, left by the guest. Low ratings are the point
// of the whole thing: a complaint that nobody sees is a customer lost quietly,
// so an unhandled low rating stays visible until someone says what was done
// about it.
type Feedback struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrderID     primitive.ObjectID `bson:"orderId" json:"orderId"`
	OrderNumber string             `bson:"orderNumber" json:"orderNumber"`
	UserID      primitive.ObjectID `bson:"userId,omitempty" json:"userId,omitempty"`
	BranchID    primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	// Who left it, copied from the order: the account may be deleted, the
	// complaint still has to be answerable.
	Customer OrderCustomer `bson:"customer" json:"customer"`

	// 1..5. Anything at or below lowRating counts as a complaint.
	Rating  int    `bson:"rating" json:"rating"`
	Comment string `bson:"comment" json:"comment"`

	// Only meaningful for a complaint: has someone dealt with it?
	Handled   bool       `bson:"handled" json:"handled"`
	HandledBy string     `bson:"handledBy,omitempty" json:"handledBy,omitempty"`
	HandledAt *time.Time `bson:"handledAt,omitempty" json:"handledAt,omitempty"`
	// What was actually done — the part that makes the record worth keeping.
	Resolution string `bson:"resolution,omitempty" json:"resolution,omitempty"`

	// Whether this one may be shown on the public site.
	//
	// ⚠️ **Per review, and off by default — this is a consent boundary, not a
	// display option.** Every row in this collection was written by a guest
	// answering "how was your order?" on their own tracking page: a private
	// message to the restaurant, with their name on it. Publishing the lot the
	// moment an owner ticks "show reviews" would put words on the internet that
	// were never offered to it, including the angry ones, under real names.
	//
	// So the switch in settings only opens the section; what appears in it is
	// chosen one review at a time. That also happens to be the only version of
	// this feature that cannot be turned into a wall of five-star quotes by
	// accident — the owner has to look at each one.
	IsPublic bool `bson:"isPublic,omitempty" json:"isPublic,omitempty"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

// ---- Site customer (phone + one-time SMS code) ----

// UserAddress is a delivery address saved on the customer profile.
type UserAddress struct {
	Label   string  `bson:"label" json:"label"`
	Text    string  `bson:"text" json:"text"`
	Lat     float64 `bson:"lat" json:"lat"`
	Lng     float64 `bson:"lng" json:"lng"`
	Comment string  `bson:"comment" json:"comment"`
}

type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FirstName string             `bson:"firstName" json:"firstName"`
	LastName  string             `bson:"lastName" json:"lastName"`
	Phone     string             `bson:"phone" json:"phone"` // 998XXXXXXXXX, verified by SMS
	// Kept for forward compatibility if another sign-in method is added.
	AuthProvider string        `bson:"authProvider" json:"authProvider"`
	Addresses    []UserAddress `bson:"addresses" json:"addresses"`
	// Cashback balance in points (1 point = 1 so'm). Denormalised for speed;
	// loyalty_txn is the record that explains it.
	Points int `bson:"points" json:"points"`

	// ---- What the restaurant knows about them (CRM) ----
	//
	// Birthday is stored as **"MM-DD"**, without a year. The only thing the
	// restaurant does with it is greet the guest on the day; a birth year is
	// personal data with no use here, and asking for it costs answers.
	Birthday string `bson:"birthday,omitempty" json:"birthday,omitempty"`
	// Free labels an operator puts on a customer: "VIP", "korporativ",
	// "shikoyatchi". Deliberately free text — every restaurant sorts its
	// regulars differently.
	Tags []string `bson:"tags,omitempty" json:"tags,omitempty"`
	// What whoever picks up the phone needs to know before speaking:
	// "yong'oqqa allergiya", "doim qo'shimcha non".
	Note string `bson:"note,omitempty" json:"note,omitempty"`
	// Where this customer came from: qr | site | instagram | referral | phone.
	// Filled in automatically on the first order and editable by hand.
	Source string `bson:"source,omitempty" json:"source,omitempty"`
	// The Telegram account this guest signs in with, when they came through the
	// mini app.
	//
	// ⚠️ **A phone number is a separate question.** Telegram never hands one over
	// with a login — it gives an id and a display name — so a guest can be fully
	// signed in and still have no way to receive an order. The checkout asks for
	// it once (see handlers/telegramauth.go), and `authProvider` records which
	// door they came through rather than pretending both are the same.
	TelegramID int64 `bson:"telegramId,omitempty" json:"telegramId,omitempty"`
	// Telegram's own UI language ("ru", "en-GB", "uz"). Kept so an order
	// notification is written in the language the guest is actually reading:
	// the site's `lang` cookie is whatever the last visitor picked on this
	// device, which is not the same question.
	TelegramLang string `bson:"telegramLang,omitempty" json:"telegramLang,omitempty"`
	// Their @username, when they have one. Kept because it is what an operator
	// searches by when a guest writes to the bot rather than phoning.
	TelegramUsername string `bson:"telegramUsername,omitempty" json:"telegramUsername,omitempty"`

	// The language this guest **chose**, as opposed to the one we guessed.
	//
	// ⚠️ This is not a duplicate of the site's `lang` cookie, and it is not a
	// duplicate of `telegramLang` either — the three answer different questions.
	// The cookie is "what this device is showing right now" and dies with the
	// browser's storage; `telegramLang` is Telegram's own UI setting, which is a
	// guess about a person who may well be an Uzbek speaker running an English
	// phone. This field is the only one that records an answer the guest gave on
	// purpose, so it wins over both — see notifiableLang.
	//
	// It has to live on the account rather than in a cookie because the thing
	// that needs it most is a **bot message**, and a message sent hours later
	// from a background goroutine has no browser and no cookie to read.
	Lang string `bson:"lang,omitempty" json:"lang,omitempty"`

	// ⚠️ The bot asked for an opinion and is waiting for the next thing they type.
	//
	// A flag rather than a conversation state machine: there is exactly one question
	// the bot ever asks, and the honest scope of "state" here is one boolean. It is
	// cleared as soon as an answer arrives or the guest does anything else, so a
	// forgotten flag cannot turn a later "salom" into a review.
	AwaitingFeedback bool `bson:"awaitingFeedback,omitempty" json:"-"`
	// Which feedback row the next message belongs to — the one the star created.
	// Without it a comment typed a minute later opens a second, ratingless row
	// beside the rating, and the panel shows one visit as two guests.
	AwaitingFeedbackID primitive.ObjectID `bson:"awaitingFeedbackId,omitempty" json:"-"`

	// Dishes this guest marked to come back to. ⚠️ On the account rather than in the
	// browser: a heart in localStorage disappears on the next device, and the whole
	// point of it is the second visit.
	Favorites []primitive.ObjectID `bson:"favorites,omitempty" json:"favorites,omitempty"`

	// This guest does not want campaign messages.
	//
	// ⚠️ **Checked on every send and never overridable from the campaign
	// screen.** One-time login codes and order updates still go out — those are
	// the service the guest asked for. What this stops is marketing, and it has
	// to be a hard exclusion rather than a filter somebody can untick: the
	// person who asked to be left alone and then gets another advert does not
	// complain to us, they stop being a customer of the restaurant.
	NoMarketing bool `bson:"noMarketing,omitempty" json:"noMarketing,omitempty"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// PhoneCode is a pending one-time SMS code. Codes are stored hashed and expire a
// few minutes after they are issued.
//
// Purpose is what the code may be used for, and it is not decoration: the
// restaurant's owner is very often also a customer on the same phone number.
// Without it, the code texted for a customer login would also unlock an admin
// password reset — a six-digit SMS meant for "sign in and order lunch" must not
// hand over the panel. Codes are keyed by (phone, purpose), so the two flows
// cannot see, consume or overwrite each other's.
type PhoneCode struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	Phone     string             `bson:"phone" json:"-"`
	Purpose   string             `bson:"purpose" json:"-"`
	CodeHash  string             `bson:"codeHash" json:"-"`
	Attempts  int                `bson:"attempts" json:"-"`
	ExpiresAt time.Time          `bson:"expiresAt" json:"-"`
	CreatedAt time.Time          `bson:"createdAt" json:"-"`
}

// ---- Admin user ----

type AdminUser struct {
	// A manager may be tied to one branch; empty means the whole company, which
	// is what an owner gets.
	BranchID           primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	ID                 primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username           string             `bson:"username" json:"username"`
	PasswordHash       string             `bson:"passwordHash" json:"-"`
	Role               string             `bson:"role" json:"role"`
	MustChangePassword bool               `bson:"mustChangePassword" json:"mustChangePassword"`
	// The site customer this admin is. Panel accounts are handed out to people
	// who already signed in on the site with their phone, so an account always
	// has a verified human behind it. Empty on the seeded first owner.
	UserID primitive.ObjectID `bson:"userId,omitempty" json:"userId,omitempty"`
	Name   string             `bson:"name" json:"name"`
	Phone  string             `bson:"phone" json:"phone"`
	// This person's internal extension on the phone system. Needed for
	// click-to-call — the exchange rings the operator's own handset first —
	// and to tell which of them answered an incoming call.
	PBXExtension string `bson:"pbxExtension,omitempty" json:"pbxExtension,omitempty"`
	// Which dashboard tiles this person wants, and in what order.
	//
	// ⚠️ **Per admin, not per company.** An owner watches takings and an
	// average bill; a branch manager watches what is unconfirmed and who is on
	// shift. One shared layout would mean the last person to tidy the screen
	// decided what everybody else sees, and the manager's version would look
	// like the owner had taken their tools away.
	Dashboard DashboardPrefs `bson:"dashboard,omitempty" json:"dashboard"`
	// Who created this account, kept by name so it still reads after that
	// admin is removed.
	CreatedBy string `bson:"createdBy" json:"createdBy"`
	// A pointer so "never signed in" stays absent from the JSON instead of
	// arriving as the year 1.
	LastLoginAt *time.Time `bson:"lastLoginAt,omitempty" json:"lastLoginAt,omitempty"`
	CreatedAt   time.Time  `bson:"createdAt" json:"createdAt"`
}

// DashboardPrefs is one admin's arrangement of the front page.
//
// ⚠️ **Both fields are empty by default, and empty must mean exactly today's
// screen.** This is the `hidePlan` rule again (§ "Stol bron qilish"): every
// account that already exists has no preferences, so a zero value that meant
// anything else — "show nothing", "show only these" — would blank the
// dashboard for every admin of every install on the day it shipped.
//
// So the field says what to **hide**, not what to show. A tile added to the
// panel next year appears for everyone automatically, which is the behaviour
// an owner expects from an update; a stored allowlist would hide every new
// tile from every existing account forever, and nobody would ever find out why.
type DashboardPrefs struct {
	// Tile ids this admin has switched off. Unknown ids are ignored rather
	// than cleaned up: an id can disappear from the panel and come back a
	// version later, and silently un-hiding it in between would look like the
	// setting had failed.
	Hidden []string `bson:"hidden,omitempty" json:"hidden"`
	// Tile ids in the order this admin wants them. Partial: ids the list does
	// not name keep their default position after the ones it does, so a person
	// who dragged two tiles to the top is not also deciding the order of the
	// eighteen they never touched.
	Order []string `bson:"order,omitempty" json:"order"`
}

// ---- Admin activity log ----

// AdminLog is one action an admin took in the panel: who, what, when. Written
// by handlers through Handler.logAction and never edited or deleted, so the
// owner can always answer "who cancelled that order?".
type AdminLog struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	AdminID primitive.ObjectID `bson:"adminId,omitempty" json:"adminId,omitempty"`
	// Denormalised so the log still reads after the account is deleted.
	AdminName string `bson:"adminName" json:"adminName"`
	AdminRole string `bson:"adminRole" json:"adminRole"`
	// Stable id like "order.cancel"; the panel translates it.
	Action string `bson:"action" json:"action"`
	// What was acted on: "order" | "courier" | "admin" | "menu" | ...
	TargetType  string    `bson:"targetType" json:"targetType"`
	TargetID    string    `bson:"targetId" json:"targetId"`
	TargetLabel string    `bson:"targetLabel" json:"targetLabel"`
	Details     string    `bson:"details" json:"details"`
	At          time.Time `bson:"at" json:"at"`
}
