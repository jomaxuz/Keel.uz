package handlers

import (
	"strings"
	"testing"

	"restaurant-backend/internal/models"
)

// ⚠️ **Nothing in this product signs an electronic document, and the day
// somebody adds a "Qabul qilish" button that merely flips a status is the day
// the panel starts lying about a tax return.** A signature is a PKCS#7 made by
// an E-IMZO key on the person's own computer; the server has no key. An
// accepted-looking document that Didox still holds unsigned is absent from the
// counterparty's books and from the tax committee's, while our screen says it
// is done — and nobody finds out until a reconciliation months later.
func TestNothingHerePretendsToSign(t *testing.T) {
	for _, file := range []string{"edi.go"} {
		src := readSource(t, file)
		for _, forbidden := range []string{
			`"/sign"`, `/sign"`, `"signature"`, "documents/{id}/sign",
		} {
			if strings.Contains(src, forbidden) {
				t.Errorf("%s reaches for the operator's signing endpoint (%s)", file, forbidden)
			}
		}
	}
	// And the client package has no Sign at all — the absence is the design,
	// so it is pinned where somebody would add one.
	client := readSourceIn(t, "../didox", "didox.go")
	if strings.Contains(client, "func (c *Client) Sign(") {
		t.Fatal("the client grew a Sign method: the server holds no key and must not")
	}
}

// ⚠️ **One document becomes one delivery, once.** The second import is a second
// delivery on the same shelf — counted twice in the balance, paid twice in the
// reports — and the two would be identical in every field that exists, so
// nothing downstream could tell them apart or undo the right one.
func TestAnInvoiceCanOnlyBecomeADeliveryOnce(t *testing.T) {
	fn := between(t, readSource(t, "edi.go"),
		"func (h *Handler) AdminEDIImport", "\n}\n")
	if !strings.Contains(fn, "doc.Imported()") {
		t.Fatal("an already-imported document is not refused")
	}
	// ⚠️ And the check is in the **filter of the write**, not only before it:
	// two storekeepers on two screens must not both pass the check and both
	// insert. The same rule the van's acceptance follows.
	if !strings.Contains(fn, `"purchaseId": bson.M{"$exists": false}`) {
		t.Fatal("the import can be run twice from two screens")
	}
	// The model's own reading of "imported" is the delivery's id, never the
	// timestamp: a stamp with no delivery behind it is a half-finished import,
	// and reading that as done loses the invoice.
	var d models.EDIDocument
	if d.Imported() {
		t.Fatal("an untouched document reads as imported")
	}
}

// ⚠️ **A line nobody matched is skipped, never guessed.** The names on an
// invoice were typed by another company for the same goods, which is the
// comparison that is right nine times and wrong once — and the wrong one puts
// beef onto the shelf of butter, silently, in the one part of the product where
// a wrong number stays invisible until a stocktake.
func TestTheMatchIsProposedAndNeverDecided(t *testing.T) {
	src := readSource(t, "edi.go")
	guess := between(t, src, "func (h *Handler) ediGuess", "\n}\n")
	// The guess is exact-name only: no prefixes, no "contains", no stemming.
	for _, clever := range []string{"HasPrefix", "Contains(", "Levenshtein"} {
		if strings.Contains(guess, clever) {
			t.Errorf("the guess got clever (%s) — a false match is silent stock arithmetic", clever)
		}
	}
	imp := between(t, src, "func (h *Handler) AdminEDIImport", "\n}\n")
	if !strings.Contains(imp, "in.IngredientID") {
		t.Fatal("the import does not read the mapping the storekeeper confirmed")
	}
}

// ⚠️ **The delivery is dated by the invoice, never by today.** A delivery is a
// measurement with its own date (models/purchase.go); importing Friday's beef
// on Monday under Monday's date moves the price history with it.
func TestTheDeliveryKeepsTheInvoicesOwnDate(t *testing.T) {
	fn := between(t, readSource(t, "edi.go"),
		"func (h *Handler) AdminEDIImport", "\n}\n")
	if !strings.Contains(fn, "ediDate(doc.Date)") {
		t.Fatal("the delivery is dated by the import rather than by the paper")
	}
	// And the date is read in the local zone: Mongo's UTC trap files every
	// morning's invoice against the previous evening in Tashkent.
	parse := between(t, readSource(t, "edi.go"), "func ediDate", "\n}\n")
	if !strings.Contains(parse, "ParseInLocation") {
		t.Fatal("the invoice date is read as UTC")
	}
}

// ⚠️ **The supplier is matched by tax number, never by name.** Our list holds
// whatever somebody typed; the document names its sender with nine digits.
// Matching on text is what filed one company's deliveries under three rows in
// the first place — the fault the supplier list exists to end.
func TestTheSupplierIsFoundByItsTaxNumber(t *testing.T) {
	fn := between(t, readSource(t, "edi.go"),
		"func (h *Handler) ediSupplier", "\n}\n")
	if !strings.Contains(fn, `bson.M{"tin": tin}`) {
		t.Fatal("the supplier is looked up by something other than its tax number")
	}
	if strings.Contains(fn, `"name": name`) {
		t.Error("a supplier is matched by name somewhere in here")
	}
	if got := tinDigits("302 936 161"); got != "302936161" {
		t.Errorf("a typed tax number did not normalise: %q", got)
	}
}

// ⚠️ **An empty secret keeps the stored one.** The rule every settings page in
// this codebase follows: an owner correcting an address must not silently
// disconnect the integration because the password field rendered blank.
func TestSavingTheSettingsWithoutTheSecretsKeepsThem(t *testing.T) {
	fn := between(t, readSource(t, "edi.go"),
		"func (h *Handler) AdminUpdateEDI", "\n}\n")
	if !strings.Contains(fn, `if v := strings.TrimSpace(req.PartnerToken); v != ""`) ||
		!strings.Contains(fn, `if v := strings.TrimSpace(req.Password); v != ""`) {
		t.Fatal("a blank field would overwrite a stored credential")
	}
	// ⚠️ And a **new** password drops the cached session: keeping it would
	// leave the integration working until the token expired hours later, with
	// nothing on screen connecting the failure to the change.
	if !strings.Contains(fn, `set["token"] = ""`) {
		t.Error("changing the password leaves an old session alive")
	}
}

// The settings response carries no secret, only whether one is stored.
func TestTheSettingsScreenNeverSeesACredential(t *testing.T) {
	fn := between(t, readSource(t, "edi.go"),
		"func (h *Handler) AdminGetEDI", "\n}\n")
	for _, leak := range []string{"s.PartnerToken,", "s.Password,", "s.Token,"} {
		if strings.Contains(fn, leak) {
			t.Errorf("a credential is returned to the browser: %s", leak)
		}
	}
	if !strings.Contains(fn, `"hasPartnerKey"`) || !strings.Contains(fn, `"hasPassword"`) {
		t.Error("the screen cannot tell whether the credentials are stored")
	}
}
