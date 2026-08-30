// Package instore charges a card from the counter, without a bank terminal.
//
// The screen this exists for is the one where a guest is standing at the till
// holding a phone. Three of the four ways a Keel till could take money already
// worked: cash goes in the drawer, a transfer is recorded, a slate is written.
// The fourth — "karta" — meant the cashier turned to a bank terminal, typed the
// total into it a second time, and waited. Every part of that is a place to be
// wrong: the amount is retyped, the receipt is a second piece of paper, and
// nothing on our screen knows whether the card was actually charged.
//
// ⚠️ **The guest's code is scanned, not shown.** This is the opposite direction
// from handlers/tillpay.go, and the difference is the whole point:
//
//	tillpay.go   we mint a link → the guest scans our QR → the bank calls us
//	             back → the till polls until the money lands. Asynchronous,
//	             and the cashier can only wait.
//	instore      the guest opens their app → the cashier scans *their* code →
//	             the card is charged in the same request. Synchronous, and the
//	             answer arrives before the guest has put the phone away.
//
// Both are kept. The QR on the screen is what a delivery courier and a guest
// with no app can use; this is what a queue at lunchtime needs.
//
// ⚠️ **Two providers, and the third is missing on purpose.** CLICK publishes
// CLICK Pass and Uzum publishes FastPay, both read and transcribed into
// docs/vendor/. Payme's equivalent — Payme GO — has no published merchant API:
// developer.help.paycom.uz documents the Merchant and Subscribe protocols,
// which are the e-commerce side, and describes the on-the-spot till as
// something that "starts accepting payments immediately after connection" —
// that is the Payme Business app's own scanner, with no integration point at
// all. So it is listed and cannot be switched on, exactly as an unbuilt fiscal
// provider is: an owner may recognise the name, and being told why is better
// than being told nothing. Same rule as internal/fiscal — guessing an endpoint
// produces code that compiles, reviews cleanly, and never charges a card.
package instore

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"time"
)

// Provider ids, as stored on the order's paymentMethod.
//
// ⚠️ **Deliberately not the same ids as the online rails.** `click` on an order
// means the guest paid on their own phone through a checkout page and a
// callback confirmed it; `click_pass` means a cashier scanned a code at the
// counter. The money is the same money, but the evidence, the refund route and
// the answer to "who was standing there" are all different, and a report that
// cannot tell them apart cannot answer any of the three.
const (
	ClickPass   = "click_pass"
	UzumFastPay = "uzum_fastpay"
	PaymeGo     = "payme_go"
)

// Kind separates the two shapes of counter payment.
//
// They share this package because they share the only thing the till cares
// about — the cashier presses one button and learns whether the card was
// charged — and they differ in what the cashier physically does, which is the
// one thing the screen has to get right.
const (
	// The cashier scans a code on the guest's phone.
	KindScan = "scan"
	// The amount is pushed to a bank terminal on the counter and the guest taps
	// their card on it. No driver exists yet; see terminal.go.
	KindTerminal = "terminal"
)

// Info is one provider as the settings page lists it.
//
// The same shape as fiscal.Info, and for the same reason: the panel's job is to
// show an owner every name they might recognise, say which of them can actually
// be switched on today, and ask for exactly the credentials that provider
// issues — no more, so nobody invents one, and no fewer, so a working provider
// is not impossible to configure.
type Info struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
	// Whether an adapter exists. A provider with Ready false may be selected
	// and its credentials saved — an owner often configures before the contract
	// closes — but never enabled.
	Ready bool   `json:"ready"`
	Note  string `json:"note"`
	// The credential boxes this provider issues, by the panel's names.
	Needs []string `json:"needs"`
}

// The credential boxes, as the panel names them.
const (
	NeedServiceID = "serviceId"
	NeedUserID    = "userId"
	NeedSecretKey = "secretKey"
	NeedBaseURL   = "baseUrl"
)

// Providers lists every counter rail we know of, in the order the panel shows
// them: what can be switched on today first.
func Providers() []Info {
	out := []Info{
		{ClickPass, "CLICK Pass", KindScan, true,
			"Mijoz Click ilovasida QR ochadi, kassir skanerlaydi. Kalitlar — Click kabinetidagi o'sha servisniki.",
			// merchant_user_id + secret_key sign every call; service_id names
			// the service the payment belongs to. No base URL: api.click.uz is
			// the only host and there is no sandbox to point at.
			[]string{NeedServiceID, NeedUserID, NeedSecretKey}},
		{UzumFastPay, "Uzum FastPay", KindScan, true,
			"Mijoz Uzum ilovasida QR ochadi, kassir skanerlaydi. ⚠️ QR har 20 soniyada yangilanadi — muddati o'tgan kod xato beradi, qayta skanerlanadi.",
			// merchant_service_user_id is the *till*, not the merchant: Uzum
			// identifies the exact checkout a request came from.
			[]string{NeedServiceID, NeedUserID, NeedSecretKey, NeedBaseURL}},
		{PaymeGo, "Payme GO", KindScan, false,
			"⚠️ Payme GO'ning ochiq API'si yo'q — to'lov Payme Business ilovasining o'z skaneri orqali qabul qilinadi. Hujjat kelsa ulanadi.",
			[]string{NeedServiceID, NeedUserID, NeedSecretKey, NeedBaseURL}},
	}
	return append(out, terminals()...)
}

// Known reports whether the id is one of ours.
func Known(id string) bool { return infoOf(id) != nil }

// Ready reports whether an adapter exists for this provider.
func Ready(id string) bool {
	p := infoOf(id)
	return p != nil && p.Ready
}

// KindOf says whether this provider is scanned or driven.
func KindOf(id string) string {
	if p := infoOf(id); p != nil {
		return p.Kind
	}
	return ""
}

// Name gives the display name for a stored id, falling back to the id itself so
// a sale taken through a provider we later dropped still says who took it.
func Name(id string) string {
	if p := infoOf(id); p != nil {
		return p.Name
	}
	return id
}

func infoOf(id string) *Info {
	for _, p := range Providers() {
		if p.ID == id {
			return &p
		}
	}
	return nil
}

// Config is one branch's connection to one provider.
type Config struct {
	Provider string
	// The service / branch the payment is filed against.
	ServiceID string
	// The cash register or cashier inside that service — merchant_user_id for
	// CLICK, merchant_service_user_id for Uzum.
	UserID    string
	SecretKey string
	// Overridable so a sandbox can be pointed at; empty is the live host.
	BaseURL string
	// What this till is called in the provider's cabinet. Free text set by us,
	// and it is what a settlement report is read by, so it names the branch.
	Cashbox string
}

// Charge is one attempt to take money from a scanned code.
type Charge struct {
	// Whole so'm, as everything in this codebase is. Each adapter converts to
	// whatever its provider counts in — Uzum takes tiyin, CLICK takes so'm —
	// and that conversion lives in exactly one place per provider.
	Amount int
	// The contents of the guest's QR code, verbatim from the scanner.
	OTPData string
	// Our order number. Both providers require it to be unique and both refuse
	// a repeat, which is the duplicate-charge guard we would otherwise have to
	// build ourselves.
	OrderID string
	// A UUID for this attempt. ⚠️ Fresh per attempt, not per order: a retry
	// after a timeout is a different attempt against the same order, and
	// reusing the id would have the provider answer about the first one.
	TxnID string
}

// Result is what the bank said.
type Result struct {
	PaymentID string
	// One of the states below.
	Status string
	// For the receipt and for the cashier's screen. Never a full PAN — both
	// providers return it masked and we store what they send.
	CardMask   string
	Processing string
	Phone      string
	// CLICK's confirm mode: the payment is reversed by the bank in 30 seconds
	// unless it is confirmed. See clickpass.go.
	NeedsConfirm bool
}

// Payment states, ours rather than either provider's.
const (
	StatusPaid    = "paid"
	StatusPending = "pending"
	StatusFailed  = "failed"
)

// Charger is what a counter rail can do.
//
// ⚠️ **Reverse and Fiscal are on the interface even though CLICK has no
// fiscal call.** The alternative is the caller asking "is this the one that
// takes a fiscal link?", which is the shape that rots: the third provider is
// added, the question is asked in four places, and one of them is missed. A
// provider that does not need a call answers it by doing nothing.
type Charger interface {
	// Charge takes the money. A returned error means we do not know whether it
	// worked; a Result with StatusFailed means the bank said no.
	Charge(ctx context.Context, c Charge) (Result, error)
	// Confirm finalises a payment held in confirm mode.
	Confirm(ctx context.Context, paymentID string) error
	// Status asks again — used when Charge times out and we have to find out
	// whether the guest was charged anyway.
	Status(ctx context.Context, paymentID, orderID string) (Result, error)
	// Reverse gives the money back in full. Partial refunds are not supported
	// by either provider and are refused before they get here.
	Reverse(ctx context.Context, paymentID, orderID string) error
	// Fiscal hands the bank the URL of the filed receipt, for providers that
	// show it to the guest in their own app.
	Fiscal(ctx context.Context, paymentID, url string) error
}

// ErrNoDriver is what an unbuilt provider returns. Deliberately an error and
// not a panic or a silent success: a restaurant that thinks it is charging
// cards and is not is the failure this whole package is written against.
var ErrNoDriver = errors.New("bu to'lov tizimi uchun adapter hali yozilmagan")

// New builds the driver for a configuration.
func New(cfg Config) (Charger, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	switch cfg.Provider {
	case ClickPass:
		return newClickPass(cfg, client)
	case UzumFastPay:
		return newUzumFastPay(cfg, client)
	}
	return nil, ErrNoDriver
}

// sign is the auth digest both providers happen to use: sha1 of the timestamp
// concatenated with the secret.
//
// ⚠️ **Shared, but the timestamp is not.** CLICK counts seconds and Uzum counts
// milliseconds in UTC+5, and a single "now" helper would be right for one of
// them and silently rejected by the other — with a 401 that reads like a wrong
// key. So the digest is shared and each adapter says what time it is.
func sign(stamp, secret string) string {
	sum := sha1.Sum([]byte(stamp + secret))
	return hex.EncodeToString(sum[:])
}

// msStamp is Uzum's timestamp: UNIX milliseconds.
//
// ⚠️ **The documentation says "in milliseconds for the UTC +5 time zone
// (regional time)", and a UNIX timestamp has no time zone.** Two readings are
// possible — a true epoch, or the Tashkent wall clock encoded as if it were
// UTC — and they differ by exactly five hours. The sample in the docs is a
// plain epoch, and the digest is signed over whatever we send, so a wrong
// reading is not a signature failure: it comes back as **403**, the error
// meaning "more than 50 seconds between the Authorization header and
// processing". That is the one symptom to look for here, and it is the reason
// this is a named function with a comment rather than an inline `UnixMilli()`.
func msStamp(now time.Time) string {
	return strconv.FormatInt(now.UnixMilli(), 10)
}

// secStamp is CLICK's: UNIX seconds, ten digits.
func secStamp(now time.Time) string {
	return strconv.FormatInt(now.Unix(), 10)
}
