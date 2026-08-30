// Package fiscal registers a sale with the tax committee through a virtual
// cash register.
//
// # Why this is one package and not six integrations
//
// The receipt body is defined by the state, not by the provider. Every virtual
// cash register in the registry files the same document — a line per item with
// its classifier code, packaging, price, quantity and VAT, and a payment split
// between cash and card, all in tiyin. So the interesting code is written once
// (Receipt, and the builder that turns one of our orders into one), and each
// provider is a thin thing that knows how to authenticate and where to POST.
//
// That is the same shape as the POS package next door, and for a stronger
// reason: there, iiko and Poster genuinely disagree about what an order is.
// Here they cannot disagree, because the document is not theirs.
//
// # What is not written yet, and why
//
// Most of these providers do not publish an API. They hand over documentation
// after a contract is signed — the same wall onlinePBX put up. So the providers
// are listed, described and selectable, and `New` returns ErrNoAdapter for the
// ones whose wire format we have not seen. Guessing an endpoint would produce
// code that compiles, reviews cleanly, and silently fails to file a single
// receipt at the one restaurant that trusted it.
//
// The listing is deliberately not gated on that: an owner scanning for their
// provider and not finding it concludes we do not support their till, and the
// answer "we support it, we need your contract's docs" is one a salesperson
// can give and a blank list cannot.
//
// # Two transports, because two of these providers are not on the internet
//
// ⚠️ **Multikassa is a program running on a PC inside the restaurant**, reached
// at something like http://192.168.14.65:9090 with no authentication at all —
// it is unauthenticated precisely because it is not reachable from outside the
// building. Our container cannot dial it, exactly as it cannot dial r_keeper.
//
// So a provider is either:
//
//   - **server-dialled** — we hold a Client and call it. The usual shape.
//   - **local** (Info.Local) — we build the request and *somebody standing in
//     the restaurant* makes it. That somebody is the till screen: the cashier's
//     tablet is already on the same LAN as the cash register, which is the one
//     machine in this whole system guaranteed to be able to reach it.
//
// ⚠️ The split is **transport only**. The request body is still built here, on
// the server, from the order — never assembled in the browser. Money and tax
// arithmetic are exactly the kind of thing a client must not be trusted with,
// and the browser's job is reduced to carrying an opaque blob across a network
// hop and bringing back the answer.
package fiscal

import (
	"context"
	"errors"
)

// Provider ids, as stored. All six are in the tax committee's registry of
// virtual cash registers.
const (
	// Multikassa, now sold under Rahmat POS — the two merged and multikassa.uz
	// redirects to rhmt.uz. Kept under the name owners still say.
	Multikassa = "multikassa"
	FirstOFD   = "firstofd"
	EPOS       = "epos"
	Regos      = "regos"
	Hippo      = "hippo"
	Simurg     = "simurg"
	// ⚠️ **`rahmat` is not `multikassa`, even though the panel's Multikassa row
	// says "(Rahmat POS)".** That row is the program on the till computer,
	// which Rahmat resells and which the local adapter already drives. Rahmat
	// also sells a *cloud* register under the same brand, reached over the
	// internet with an account rather than over the LAN with none — a different
	// transport, different credentials and a different failure mode. Folding
	// them into one id would mean an owner who bought the cloud product
	// selecting a provider that tries to dial their office network.
	Rahmat = "rahmat"
	QPOS   = "qpos"
	Arca   = "arca"
)

// ErrNoAdapter means the provider is real and legal but we have not built its
// wire format yet. Distinct from a configuration error on purpose: the fix is
// ours, not the restaurant's, and the panel says so rather than sending an
// owner to re-check credentials that were never wrong.
var ErrNoAdapter = errors.New("fiscal: provider not connected yet")

// ErrNotConfigured means the drawer is empty or half-filled.
var ErrNotConfigured = errors.New("fiscal: provider not configured")

// ErrLocalProvider means the provider is built, but not reachable from here.
// The caller wanted a Client and this provider is filed from the till screen —
// see the transport note at the top of the file.
var ErrLocalProvider = errors.New("fiscal: provider is filed from the till screen")

// Info is what the panel needs to draw the provider list.
type Info struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Whether an adapter exists. A provider with Ready false can be selected
	// and its credentials saved — the owner is usually setting this up before
	// the contract completes — but enabling it is refused with a reason.
	Ready bool `json:"ready"`
	// Whether it runs inside the restaurant rather than on the internet.
	//
	// The panel needs this for two visible reasons: the credentials drawer asks
	// for a LAN address instead of a login, and the connection check has to be
	// pressed on the till screen rather than here — the owner may well be
	// reading the settings page from home.
	Local bool `json:"local"`
	// Shown under the name in the panel. One line, in the owner's terms.
	Note string `json:"note"`
}

// Providers lists every virtual cash register we know of, in the order the
// panel shows them.
//
// ⚠️ **Ordered by how likely an owner is to have it**, not alphabetically: this
// list is read by somebody looking for a name they already know, and the two
// they are most likely to know should not be below four they are not.
func Providers() []Info {
	return []Info{
		{Multikassa, "Multikassa (Rahmat POS)", true, true,
			"Eng keng tarqalgani. Kassa kompyuteridagi dastur bilan bevosita ishlaydi — kassa ekrani o'sha tarmoqda bo'lishi shart."},
		{FirstOFD, "Birinchi ОФД", false, false,
			"Bulutli fiskalizatsiya, buxgalteriya tizimlari bilan integratsiyasi bor."},
		{EPOS, "E-POS", false, false, "Virtual kassa va chek chop etish."},
		{Regos, "REGOS VCR", true, true,
			"API hujjati ochiq. Kassa dasturi restoran kompyuterida ishlaydi; sinov uchun bulutli muhit ham bor."},
		{Rahmat, "Rahmat POS (bulutli)", false, false,
			"Rahmat'ning bulutli virtual kassasi. ⚠️ Kassa kompyuteridagi dastur — yuqoridagi «Multikassa» qatori."},
		{Hippo, "Hippo POS", false, false, "Virtual kassa (943-son qaror bo'yicha)."},
		{QPOS, "QPOS", false, false, "Virtual kassa va to'lov terminali."},
		{Arca, "Arca Group", false, false, "PAX terminallaridagi onlayn kassa."},
		{Simurg, "SIMURG", false, false, "Reestrdagi virtual kassa dasturi."},
	}
}

// Known reports whether the id is one of ours.
func Known(id string) bool {
	for _, p := range Providers() {
		if p.ID == id {
			return true
		}
	}
	return false
}

// Name gives the display name for a stored id, falling back to the id itself so
// an old receipt filed by a provider we later dropped still says who filed it.
func Name(id string) string {
	for _, p := range Providers() {
		if p.ID == id {
			return p.Name
		}
	}
	return id
}

// Ready reports whether an adapter exists for this provider.
func Ready(id string) bool {
	for _, p := range Providers() {
		if p.ID == id {
			return p.Ready
		}
	}
	return false
}

// IsLocal reports whether this provider runs inside the restaurant and must
// therefore be reached from the till screen rather than from here.
func IsLocal(id string) bool {
	for _, p := range Providers() {
		if p.ID == id {
			return p.Local
		}
	}
	return false
}

// Request is one call to a provider, built here and made by whoever can reach
// it — us, for a server-dialled provider, or the till screen for a local one.
//
// ⚠️ Path is a **path, not a URL**. The host belongs to whoever makes the call:
// for a local provider it is the address of the machine on the restaurant's
// own network, which this server has never seen and has no business storing as
// something it might one day dial itself.
type Request struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers,omitempty"`
	// The body, already serialised. A string rather than a structure because
	// the far side is the browser, and every field it can see is a field it
	// could be tempted to adjust.
	Body string `json:"body,omitempty"`
}

// Encoder turns our receipt into a provider's wire format and reads its answer
// back. Separate from Client because a local provider has no dialling half —
// and because these two functions are the part worth testing, while dialling is
// the part that cannot be.
type Encoder interface {
	// Sale builds the filing request for a receipt.
	Sale(r Receipt) (Request, error)
	// Hello builds the request that asks the register who it is, for the
	// connection check. Same purpose as Client.Ping.
	Hello() (Request, error)
	// Parse reads a response. status is the HTTP status the caller saw, which
	// several of these providers use for business errors as well as transport
	// ones — so it is evidence, not a verdict.
	Parse(status int, body []byte) (Result, error)
}

// DescribeFor turns a connection check's reply into the line the panel shows.
//
// ⚠️ Dispatched by provider rather than sniffed from the body: two registers
// answering different shapes that happen to share a field name would produce a
// confident description of the wrong machine, and this line exists precisely to
// tell one register from another.
func DescribeFor(provider string, body []byte) string {
	switch provider {
	case Multikassa:
		return Describe(body)
	case Regos:
		return DescribeRegos(body)
	}
	return ""
}

// EncoderFor builds the encoder for a provider.
func EncoderFor(provider string, creds Creds) (Encoder, error) {
	switch provider {
	case Multikassa:
		return newMultikassa(creds)
	case Regos:
		return newRegos(creds)
	case "":
		return nil, ErrNotConfigured
	}
	return nil, ErrNoAdapter
}

// Client is one provider's connection. Deliberately small: everything hard
// happens in Receipt, which every provider files unchanged.
type Client interface {
	// File registers the receipt and returns the fiscal sign and QR.
	File(ctx context.Context, r Receipt) (Result, error)
	// Refund files the reversal of an already-filed receipt.
	//
	// Separate from File-with-IsRefund because a refund needs the original's
	// id, and a provider that cannot do it at all should say ErrUnsupported
	// rather than filing a second sale with negative numbers.
	Refund(ctx context.Context, receiptID string, r Receipt) (Result, error)
	// Ping says what we are connected to, in words an owner can read — "kassa
	// №123, ishlayapti" rather than "ok". The POS package learned this: a bare
	// "connected" does not distinguish the right register from a stranger's.
	Ping(ctx context.Context) (string, error)
}

// ErrUnsupported is returned by an operation a provider genuinely lacks, so the
// panel can say "do it in the provider's cabinet" instead of showing a retry
// button that will never work.
var ErrUnsupported = errors.New("fiscal: not supported by this provider")

// Result is what came back from a filing.
type Result struct {
	// The provider's id for the receipt, needed to refund it later.
	ReceiptID string
	// The fiscal sign printed on the receipt.
	FiscalSign string
	// The text the QR encodes — usually a tax-committee URL the guest opens.
	// Stored rather than rebuilt: the format is the provider's to decide, and
	// a QR we composed ourselves would be a QR that leads nowhere.
	QRText string
}

// New builds the client for a provider we can dial ourselves.
//
// Every id in Providers() is handled, so adding an adapter is a visible,
// single-line change and forgetting one is a compile-time gap rather than a
// silent fallthrough.
//
// ⚠️ Multikassa answers ErrLocalProvider rather than ErrNoAdapter, and the
// difference is the whole point: the adapter exists and works, it is simply not
// dialled from here. Reporting "not built yet" would send an owner to wait for
// something that already shipped.
func New(provider string, creds Creds) (Client, error) {
	switch provider {
	case Multikassa, Regos:
		return nil, ErrLocalProvider
	case FirstOFD, EPOS, Hippo, Simurg:
		return nil, ErrNoAdapter
	case "":
		return nil, ErrNotConfigured
	}
	return nil, ErrNoAdapter
}

// Creds is what an adapter needs to dial its provider, in our vocabulary. The
// handler fills it from the branch's drawer so that no adapter reads the
// settings document — the same separation the POS adapters have.
type Creds struct {
	Login      string
	Password   string
	Token      string
	RegisterID string
	BaseURL    string
	TIN        string
}
