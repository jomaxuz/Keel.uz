// Package pos talks to the till the restaurant already runs.
//
// A restaurant that has iiko, r_keeper or Clopos is not looking for a second
// place to keep its menu — it is looking for its website orders to appear on
// the kitchen printer and in the day's takings, like every other order. That is
// the whole job here: **push the order across**. The menu stays ours (the
// panel is where photos, translations and combos live), and each dish carries
// the id it has in the till.
//
// Three systems, one interface. They differ in wire format and in almost
// nothing else:
//
//	iiko     cloud JSON, api-ru.iiko.services, token from an apiLogin
//	Clopos   cloud JSON, integrations.clopos.com, JWT in an x-token header
//	r_keeper XML, and — the part that matters — **on the restaurant's own LAN**
//
// That last one is not a detail. r_keeper's XML interface answers on
// `https://<ip>:<port>/rk7api/v0/xmlinterface.xml` inside the restaurant, so a
// server in a data centre cannot reach it at all without the restaurant opening
// a port or running a VPN. The adapter is written and works; whether it can be
// *plugged in* is a networking question at each site, and the panel says so
// rather than letting an owner tick a box and wonder why nothing arrives.
package pos

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Provider ids, as stored.
const (
	IIKO = "iiko"
	// Syrve is iiko sold under its international name — the same cloud API on
	// api-eu.syrve.live. It is a separate provider here anyway, because the
	// restaurant knows its till as "Syrve": an owner who scans the list for
	// their system and finds only "iiko" concludes we do not support it.
	Syrve   = "syrve"
	Clopos  = "clopos"
	Poster  = "poster"
	RKeeper = "rkeeper"
)

// Product is one sellable thing in the till, reduced to what a human needs to
// pick it off a list. The panel shows these next to our own dishes so an
// operator maps "Lag'mon" to a name rather than to a 36-character GUID.
type Product struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Empty when the till does not group things, or the group is unnamed.
	Category string `json:"category"`
	// In so'm. Only ever shown — the price charged is the panel's, because the
	// panel is where the menu is priced.
	Price int `json:"price"`
	// Not currently sellable (stopped, archived, out of stock). Kept visible
	// but marked: mapping a dish to something the kitchen has stopped is
	// exactly the mistake worth catching at mapping time.
	Unavailable bool `json:"unavailable"`
}

// Item is one line of an order on its way to the till.
type Item struct {
	// The id in the till. An item without one cannot be sent — see ErrUnmapped.
	POSID string
	Name  string
	Qty   int
	// Unit price in so'm, as the customer agreed it.
	Price int
	// What the guest asked for ("piyozsiz"). Printed for the kitchen.
	Comment string
	// Modifier ids in the till, when the dish's options were mapped too.
	Modifiers []Modifier
}

type Modifier struct {
	POSID string
	Name  string
	Qty   int
	Price int
}

// Order is what gets pushed across, in our own vocabulary. Each adapter
// translates it; nothing here is provider-shaped.
type Order struct {
	Number string
	// "delivery" | "pickup" | "dinein"
	Type         string
	CustomerName string
	Phone        string
	Address      string
	Lat, Lng     float64
	// The guest's note for the whole order, plus anything the operator added.
	Comment     string
	Items       []Item
	DeliveryFee int
	Total       int
	// "cash" | "payme" | "click" | "uzum"
	PaymentMethod string
	// Whether the money is already in. A till that knows an order is paid does
	// not ask the courier to collect, which is the difference between a correct
	// shift report and an argument at the end of the day.
	Paid bool
	// Dine-in only.
	TableNumber string
	CreatedAt   time.Time
}

// Result is what the till answered with.
type Result struct {
	// The till's own id for the order, kept so status can be asked for later.
	POSOrderID string `json:"posOrderId"`
	// Free text worth showing the operator: the till's order number, a warning.
	Note string `json:"note"`
}

// Status is where the till thinks the order is. Deliberately coarse: the three
// systems disagree about the middle of the pipeline and agree about the ends,
// and only the ends are worth acting on.
type Status struct {
	// "unknown" | "accepted" | "cooking" | "ready" | "closed" | "cancelled"
	State string `json:"state"`
	// The till's own wording, for the operator who wants the detail.
	Raw string `json:"raw"`
}

// Provider is one till system.
type Provider interface {
	// Name is the stored provider id.
	Name() string
	// Ping proves the credentials work and returns something readable to show
	// the owner — the organisation or venue name it connected to. A settings
	// page with a "check" button that answers "connected to Maracanda,
	// terminal Kassa 1" is the difference between configured and configured
	// correctly.
	Ping(ctx context.Context) (string, error)
	// Products lists what the till can sell, for the mapping screen.
	Products(ctx context.Context) ([]Product, error)
	// SendOrder pushes one order.
	SendOrder(ctx context.Context, o Order) (Result, error)
	// OrderStatus asks what became of it. Providers that cannot answer return
	// ErrUnsupported, which is not a failure.
	OrderStatus(ctx context.Context, posOrderID string) (Status, error)
	// Cancel withdraws an order, where the till allows it.
	Cancel(ctx context.Context, posOrderID string) error
}

var (
	// ErrUnsupported is how an adapter says "this till does not do that". The
	// caller treats it as a shrug, not an error — r_keeper cannot be asked for
	// a delivery status the way iiko can, and that must not paint the
	// integration red.
	ErrUnsupported = errors.New("bu POS tizimi bu amalni qo'llab-quvvatlamaydi")
	// ErrUnmapped means a dish on the order has no id in the till. Sending it
	// anyway would put a line the kitchen cannot cook onto a real ticket, so
	// the whole order is refused and the operator is told which dish.
	ErrUnmapped = errors.New("taom POS tizimiga bog'lanmagan")
	// ErrNotConfigured is a provider that is switched on but not filled in.
	ErrNotConfigured = errors.New("POS tizimi to'liq sozlanmagan")
)

// Config is everything any of the three needs. One flat struct rather than
// three: it is stored as one document, and the adapters take only their own
// fields out of it.
type Config struct {
	Provider string

	// ---- iiko and Syrve ----
	//
	// Deliberately shared: the two speak the same API, so duplicating six
	// fields would only create a second place for the same bug. Which host is
	// dialled comes from Provider (see newIiko); what is *stored* stays
	// separate, so switching providers in the panel never mixes credentials.
	IikoAPILogin       string
	IikoOrganizationID string
	IikoTerminalGroup  string
	// Optional: which order type and payment type the till should file it as.
	// Left empty, iiko uses the organisation's defaults.
	IikoOrderTypeID   string
	IikoPaymentTypeID string
	IikoBaseURL       string

	// ---- Clopos ----
	CloposClientID     string
	CloposClientSecret string
	CloposBrand        string
	CloposIntegratorID string
	CloposVenueID      int
	CloposSaleTypeID   int
	CloposBaseURL      string

	// ---- Poster (joinposter.com) ----
	// The token is the whole authentication: it goes in the query string, and
	// it carries the account with it.
	PosterToken string
	// Which sales point (filial) the order is filed against. Poster requires
	// it on every order, and prices are per spot.
	PosterSpotID  int
	PosterBaseURL string

	// ---- r_keeper ----
	// The full URL of the XML interface inside the restaurant.
	RKeeperURL      string
	RKeeperLogin    string
	RKeeperPassword string
	// Which station (till) the order is filed against.
	RKeeperStation string
	// Subscription licensing signs SaveOrder/PayOrder; lifetime licences do
	// not need these.
	RKeeperAnchor string
	RKeeperToken  string
}

// New builds the adapter for a configuration, or nil when nothing is set up.
func New(cfg Config) (Provider, error) {
	client := &http.Client{Timeout: 25 * time.Second}
	switch cfg.Provider {
	case IIKO, Syrve:
		if cfg.IikoAPILogin == "" || cfg.IikoOrganizationID == "" {
			return nil, ErrNotConfigured
		}
		return newIiko(cfg, client), nil
	case Poster:
		if cfg.PosterToken == "" || cfg.PosterSpotID == 0 {
			return nil, ErrNotConfigured
		}
		return newPoster(cfg, client), nil
	case Clopos:
		if cfg.CloposClientID == "" || cfg.CloposClientSecret == "" ||
			cfg.CloposBrand == "" || cfg.CloposVenueID == 0 {
			return nil, ErrNotConfigured
		}
		return newClopos(cfg, client), nil
	case RKeeper:
		if cfg.RKeeperURL == "" || cfg.RKeeperLogin == "" {
			return nil, ErrNotConfigured
		}
		return newRKeeper(cfg, client), nil
	case "":
		return nil, nil
	}
	return nil, fmt.Errorf("noma'lum POS tizimi: %s", cfg.Provider)
}

// CheckMapped reports the first dish that has no id in the till.
//
// Called before anything is sent, so a half-sendable order fails here — in the
// panel, with a name in the message — rather than as a ticket the kitchen
// cannot fill.
func CheckMapped(items []Item) error {
	for _, it := range items {
		if it.POSID == "" {
			return fmt.Errorf("%w: %s", ErrUnmapped, it.Name)
		}
	}
	return nil
}
