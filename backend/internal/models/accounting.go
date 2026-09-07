package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- The accountant's two doors: Didox and 1C ----
//
// ⚠️ **Both exist because the same fact is being typed twice today.** A
// delivery arrives with an electronic invoice already filed at the tax
// committee (Didox), and somebody retypes its twenty lines into the panel; the
// month's sales are already in the panel, and somebody retypes their totals
// into 1C. Neither retyping produces anything new — it produces a second
// version of a number, which is the shape every wrong figure in this codebase
// has had.
//
// ⚠️ **They are two integrations and not one screen, because they answer to
// different people.** Didox is the *document*: it is legal evidence, it is
// signed with a key, and the counterparty sees the same paper. 1C is the
// *bookkeeping*: it is the accountant's own working copy, it has no legal
// weight of its own, and nobody outside the company ever looks at it. Folding
// them together would put an accountant's convenience and a taxable document
// behind one switch.
//
// ⚠️ **We never sign anything, and no screen may look as if we did.** Every
// Didox signature is made by an E-IMZO key on the person's own computer; the
// server has no key and must not have one. So Keel reads, drafts and imports —
// and for signing it says plainly whose job that is. A button that flipped a
// status without a signature would leave the document unsigned at Didox and
// missing from the tax return, while the panel showed it done: the exact class
// of silent failure the fiscal package is written to avoid.

// EDIProvider ids. Only one today, named rather than assumed: an EDI operator
// is chosen by the customer's accountant and the second one always arrives.
const (
	EDIDidox = "didox"
)

// EDISettings is this company's connection to its EDI operator.
//
// In its own collection, never on `restaurant`: the profile document goes to
// every visitor of the website in full, and these are credentials (the rule
// written at the top of CLAUDE.md §4).
type EDISettings struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"-"`

	Provider string `bson:"provider" json:"provider"`
	Enabled  bool   `bson:"enabled" json:"enabled"`

	// ⚠️ **The test rails are a first-class setting rather than a build flag.**
	// An accountant connecting a real company should be able to watch one
	// invoice travel end to end before anything is filed under their name, and
	// the alternative — trying it in production — is trying it on a document
	// the tax committee keeps.
	Sandbox bool `bson:"sandbox,omitempty" json:"sandbox"`

	// СТИР of the company whose documents these are.
	TIN string `bson:"tin" json:"tin"`

	// ---- Secrets: never returned, and an empty value keeps the stored one ----
	//
	// The partner token is issued to Keel by Didox and is the same for every
	// customer; the password is the customer's own. They are kept apart because
	// they are lost and rotated separately — and because a support engineer
	// pasting a partner token into a customer's field would otherwise look like
	// a working configuration.
	PartnerToken string `bson:"partnerToken,omitempty" json:"-"`
	Password     string `bson:"password,omitempty" json:"-"`

	// The user token Didox hands back, cached until it expires.
	//
	// ⚠️ **Cached with its expiry rather than re-fetched per request.** A token
	// lives 360 minutes and every login counts against the "too many attempts"
	// limit — a panel that logged in per page view would lock the customer's
	// own account out of didox.uz, which is not a screen we could explain.
	Token        string     `bson:"token,omitempty" json:"-"`
	TokenExpires *time.Time `bson:"tokenExpires,omitempty" json:"-"`

	// ---- What an outgoing invoice says about us ----
	//
	// ⚠️ **Stored rather than read from the Didox profile on every send.** They
	// are printed onto a document that is kept for years; a field that quietly
	// changed because somebody edited a profile elsewhere would rewrite what
	// last month's invoice claims we told the buyer.
	Seller EDIParty `bson:"seller" json:"seller"`

	// Where the inbox has been read up to, so the next pull is short.
	LastSyncAt *time.Time `bson:"lastSyncAt,omitempty" json:"lastSyncAt,omitempty"`
	// What the last pull said, in the operator's own words. ⚠️ Kept as text: a
	// boolean "ok" cannot say "the password was changed on didox.uz".
	LastError string    `bson:"lastError,omitempty" json:"lastError,omitempty"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// EDIParty is a seller or a buyer as an invoice prints them.
//
// ⚠️ **Every field here is required by the receiving side's validator**, which
// checks the structure strictly (see docs/vendor/didox.md): a missing bank code
// or VAT registration code is not a blank line on the paper, it is a document
// the counterparty's system refuses whole.
type EDIParty struct {
	Name         string `bson:"name,omitempty" json:"name"`
	VatRegCode   string `bson:"vatRegCode,omitempty" json:"vatRegCode"`
	VatRegStatus int    `bson:"vatRegStatus,omitempty" json:"vatRegStatus"`
	Account      string `bson:"account,omitempty" json:"account"`
	BankID       string `bson:"bankId,omitempty" json:"bankId"`
	Address      string `bson:"address,omitempty" json:"address"`
	Director     string `bson:"director,omitempty" json:"director"`
	Accountant   string `bson:"accountant,omitempty" json:"accountant"`
}

// ---- Didox document types and statuses, as the operator numbers them ----
//
// ⚠️ **Kept as the operator's own codes, not translated into ours.** A status
// on this screen is a claim about what Didox holds; renaming 3 to "signed" in
// the database would give us a second vocabulary to keep in step, and the day
// they diverge the panel would be describing a document that does not exist.
const (
	EDITypeFacturaNoAct = "002" // счёт-фактура без акта — the ordinary one
	EDITypeFactura      = "001"
	EDITypeFacturaFarm  = "008"
	EDITypeAct          = "005"
	EDITypeWaybill      = "041"
	EDITypeContract     = "007"
)

const (
	EDIStatusDraft         = 0
	EDIStatusWaitPartner   = 1
	EDIStatusWaitUs        = 2
	EDIStatusSigned        = 3
	EDIStatusRejected      = 4
	EDIStatusDeleted       = 5
	EDIStatusWaitAgent     = 6
	EDIStatusSignedByAgent = 8
	EDIStatusNotValid      = 40
)

// EDILine is one row of an electronic invoice, as it arrived.
//
// ⚠️ **Frozen exactly as the document said**, including the name and the
// classifier code, and never re-read from the linked ingredient. The document
// is evidence: what it called the goods on the day it was signed is part of
// what it says, and a row that renamed itself when somebody tidied the
// catalogue would make the invoice disagree with the counterparty's copy.
type EDILine struct {
	No int `bson:"no" json:"no"`
	// The seller's own words for the goods.
	Name string `bson:"name" json:"name"`
	// ИКПУ — the state classifier code. The strongest handle we have for
	// matching a line to something in our own catalogue, because it is the one
	// value both sides copied from the same list.
	CatalogCode string  `bson:"catalogCode,omitempty" json:"catalogCode,omitempty"`
	Barcode     string  `bson:"barcode,omitempty" json:"barcode,omitempty"`
	PackageName string  `bson:"packageName,omitempty" json:"packageName,omitempty"`
	PackageCode string  `bson:"packageCode,omitempty" json:"packageCode,omitempty"`
	Qty         float64 `bson:"qty" json:"qty"`
	// Per unit and per line, both as the document has them.
	//
	// ⚠️ **Never derived from one another.** An invoice rounds; a line total
	// recomputed from a rounded unit price is a different number from the one
	// the counterparty will be paid, and the difference lands in the cost of
	// everything the goods go into. Same rule as `Purchase.Total`.
	Price      float64 `bson:"price" json:"price"`
	Sum        float64 `bson:"sum" json:"sum"`
	VatRate    float64 `bson:"vatRate,omitempty" json:"vatRate,omitempty"`
	VatSum     float64 `bson:"vatSum,omitempty" json:"vatSum,omitempty"`
	SumWithVat float64 `bson:"sumWithVat" json:"sumWithVat"`

	// What this line was matched to when it was imported, if anything.
	IngredientID primitive.ObjectID `bson:"ingredientId,omitempty" json:"ingredientId,omitempty"`
}

// EDIDocument is our copy of one document held by the operator.
//
// ⚠️ **A copy, and it is never the truth.** Didox holds the document and its
// signatures; this row exists so the panel can list, search and link without
// dialling the operator for every screen, and so an imported delivery can point
// back at the paper it came from. Anything this row claims about a *status* is
// as fresh as the last pull, which is why `syncedAt` is stored beside it.
type EDIDocument struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	// The operator's own 32-character id. ⚠️ **Unique**: a pull that overlapped
	// with the previous one would otherwise write a second copy, and a
	// storekeeper would import the same delivery twice — counted twice on the
	// shelf and paid for twice in the reports.
	DocID string `bson:"docId" json:"docId"`
	// "in" — somebody sent it to us; "out" — we sent it.
	Direction string `bson:"direction" json:"direction"`
	Type      string `bson:"type" json:"type"`
	Number    string `bson:"number" json:"number"`
	Date      string `bson:"date" json:"date"`
	Status    int    `bson:"status" json:"status"`

	PartnerTIN  string `bson:"partnerTin,omitempty" json:"partnerTin,omitempty"`
	PartnerName string `bson:"partnerName,omitempty" json:"partnerName,omitempty"`

	ContractNo   string `bson:"contractNo,omitempty" json:"contractNo,omitempty"`
	ContractDate string `bson:"contractDate,omitempty" json:"contractDate,omitempty"`

	Total        float64 `bson:"total" json:"total"`
	VatTotal     float64 `bson:"vatTotal,omitempty" json:"vatTotal,omitempty"`
	TotalWithVat float64 `bson:"totalWithVat" json:"totalWithVat"`
	HasVat       bool    `bson:"hasVat,omitempty" json:"hasVat"`
	HasMarks     bool    `bson:"hasMarks,omitempty" json:"hasMarks"`

	// The lines, once somebody has opened the document. ⚠️ The list endpoint
	// does not carry them, so a document pulled by the inbox has none until it
	// is read — and an empty slice here means "not fetched yet", not "an
	// invoice with nothing on it".
	Lines []EDILine `bson:"lines,omitempty" json:"lines"`

	// ---- What happened on our side ----
	//
	// ⚠️ **The link is one-way and one-time.** A document that has become a
	// delivery carries the delivery's id; the import refuses to run twice
	// against the same document, because the second run is a second delivery on
	// the same shelf and nothing downstream could tell them apart.
	PurchaseID primitive.ObjectID `bson:"purchaseId,omitempty" json:"purchaseId,omitempty"`
	ImportedAt *time.Time         `bson:"importedAt,omitempty" json:"importedAt,omitempty"`
	ImportedBy string             `bson:"importedBy,omitempty" json:"importedBy,omitempty"`

	SyncedAt  time.Time `bson:"syncedAt" json:"syncedAt"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

// Incoming reports whether somebody sent this to us.
func (d EDIDocument) Incoming() bool { return d.Direction == EDIIn }

// Imported reports whether this document has already become a delivery.
//
// ⚠️ **The delivery's id is the test, not the timestamp.** A stamp with no
// document behind it is what a half-finished import leaves, and reading that as
// "done" would lose the invoice entirely.
func (d EDIDocument) Imported() bool { return !d.PurchaseID.IsZero() }

const (
	EDIIn  = "in"
	EDIOut = "out"
)

// ---- 1C ----

// OneCSettings is the door an accountant's 1C knocks on.
//
// ⚠️ **1C always starts the exchange and we never start it.** The customer's
// 1C sits on an office machine that usually cannot be reached from the
// internet; the published protocol is built around that fact (see
// docs/vendor/1c-exchange.md), and a design that tried to push into 1C would
// work in a demonstration and in no real office.
type OneCSettings struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	Enabled bool               `bson:"enabled" json:"enabled"`

	// The credentials typed into 1C's own exchange settings.
	//
	// ⚠️ **Their own, not a panel administrator's.** The password is stored on
	// an office machine, in a configuration form, in clear text — that is what
	// 1C does with it — so it must not be a login that can also open the panel.
	Login string `bson:"login" json:"login"`
	// bcrypt, like every other password in this codebase. ⚠️ Never returned.
	PasswordHash string `bson:"passwordHash,omitempty" json:"-"`

	// Which branch's counter the sales come from. Empty = every branch, which
	// is what a single-branch customer wants and never has to choose.
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`

	// ⚠️ **How far back a fresh exchange looks, in days.** 1C asks for
	// "everything since last time" and has no memory of ours; without a floor,
	// the first exchange of a two-year-old restaurant would build a document
	// with a hundred thousand lines in it and time out — repeatedly, because
	// 1C retries.
	Days int `bson:"days,omitempty" json:"days"`

	// What the last exchange did, so the panel can answer "is it working?"
	// without anybody opening 1C.
	LastSeenAt *time.Time `bson:"lastSeenAt,omitempty" json:"lastSeenAt,omitempty"`
	LastType   string     `bson:"lastType,omitempty" json:"lastType,omitempty"`
	LastMode   string     `bson:"lastMode,omitempty" json:"lastMode,omitempty"`
	LastError  string     `bson:"lastError,omitempty" json:"lastError,omitempty"`
	// How many products the last catalogue import wrote, and how many documents
	// the last sales query answered with.
	LastImported int `bson:"lastImported,omitempty" json:"lastImported,omitempty"`
	LastExported int `bson:"lastExported,omitempty" json:"lastExported,omitempty"`

	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// OneCDays is how far back a sales query reaches when nothing is configured.
const OneCDays = 31
