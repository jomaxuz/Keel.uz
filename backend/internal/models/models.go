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
	Booking  BookingSettings `bson:"booking" json:"booking"`
	Currency string          `bson:"currency" json:"currency"`
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
	MapAPIKey string      `bson:"mapApiKey" json:"mapApiKey"`
	SEO       SEOSettings `bson:"seo" json:"seo"`
	UpdatedAt time.Time   `bson:"updatedAt" json:"updatedAt"`
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
}

type BookingSettings struct {
	Enabled bool `bson:"enabled" json:"enabled"`
	// The plan's own coordinate space; the site scales it to fit.
	Width  float64 `bson:"width" json:"width"`
	Height float64 `bson:"height" json:"height"`
	// How long one booking holds a table, and how far ahead guests may book.
	SlotMinutes  int `bson:"slotMinutes" json:"slotMinutes"`
	MaxDaysAhead int `bson:"maxDaysAhead" json:"maxDaysAhead"`
	// Minimum notice: a table cannot be booked for five minutes from now.
	MinNoticeMinutes int          `bson:"minNoticeMinutes" json:"minNoticeMinutes"`
	MaxGuests        int          `bson:"maxGuests" json:"maxGuests"`
	Shapes           []FloorShape `bson:"shapes" json:"shapes"`
	Tables           []FloorTable `bson:"tables" json:"tables"`
	Note             string       `bson:"note" json:"note"`
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
	// When true, clocking in also requires a valid code — the geofence alone
	// is not enough. Off by default so existing branches keep working.
	RequireKioskCode bool `bson:"requireKioskCode" json:"requireKioskCode"`
	// Short code printed in front of this branch's order numbers ("CHL-A71-4509").
	// Empty on a single-branch install, where the prefix would say nothing.
	Code string `bson:"code" json:"code"`
	// Dishes that have run out **here today**. The menu belongs to the brand and
	// is the same everywhere; what is left in the pot is the branch's own
	// business, so this lives on the branch rather than on the dish.
	SoldOut   []primitive.ObjectID `bson:"soldOut" json:"soldOut"`
	SortOrder int                  `bson:"sortOrder" json:"sortOrder"`
	IsActive  bool                 `bson:"isActive" json:"isActive"`
	CreatedAt time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time            `bson:"updatedAt" json:"updatedAt"`
}

// IsSoldOut reports whether a dish has run out at this branch.
func (b *Branch) IsSoldOut(id primitive.ObjectID) bool {
	for _, x := range b.SoldOut {
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
	NameRu        string       `bson:"nameRu" json:"nameRu"`
	NameEn        string       `bson:"nameEn" json:"nameEn"`
	DescriptionRu string       `bson:"descriptionRu" json:"descriptionRu"`
	DescriptionEn string       `bson:"descriptionEn" json:"descriptionEn"`
	Price         int          `bson:"price" json:"price" validate:"gte=0"`
	OldPrice      *int         `bson:"oldPrice" json:"oldPrice"`
	ImageURL      string       `bson:"imageUrl" json:"imageUrl"`
	Images        []string     `bson:"images" json:"images"`
	IsAvailable   bool         `bson:"isAvailable" json:"isAvailable"`
	IsPopular     bool         `bson:"isPopular" json:"isPopular"`
	SortOrder     int          `bson:"sortOrder" json:"sortOrder"`
	Options       []MenuOption `bson:"options" json:"options"`
	Tags          []string     `bson:"tags" json:"tags"`
	UpdatedAt     time.Time    `bson:"updatedAt" json:"updatedAt"`

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
}

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
	// What was actually in the drawer.
	Counted int `bson:"counted" json:"counted"`
	// Counted − Expected. Negative is a shortfall.
	Variance int `bson:"variance" json:"variance"`
	// Why, when it does not match. **Required for a non-zero variance**: an
	// unexplained shortfall recorded without a sentence is one nobody can act
	// on a week later, and the person who could explain it has gone home.
	VarianceNote string `bson:"varianceNote,omitempty" json:"varianceNote,omitempty"`

	Note      string    `bson:"note,omitempty" json:"note,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
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
	PointsSpent   int           `bson:"pointsSpent,omitempty" json:"pointsSpent,omitempty"`
	PointsEarned  int           `bson:"pointsEarned,omitempty" json:"pointsEarned,omitempty"`
	DeliveryFee   int           `bson:"deliveryFee" json:"deliveryFee"`
	Total         int           `bson:"total" json:"total"`
	PaymentMethod string        `bson:"paymentMethod" json:"paymentMethod"`
	DeliveryZone  string        `bson:"deliveryZone" json:"deliveryZone"`
	DistanceKm    float64       `bson:"distanceKm" json:"distanceKm"`
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
	POS       *OrderPOS `bson:"pos,omitempty" json:"pos,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
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
	// Who created this account, kept by name so it still reads after that
	// admin is removed.
	CreatedBy string `bson:"createdBy" json:"createdBy"`
	// A pointer so "never signed in" stays absent from the JSON instead of
	// arriving as the year 1.
	LastLoginAt *time.Time `bson:"lastLoginAt,omitempty" json:"lastLoginAt,omitempty"`
	CreatedAt   time.Time  `bson:"createdAt" json:"createdAt"`
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
