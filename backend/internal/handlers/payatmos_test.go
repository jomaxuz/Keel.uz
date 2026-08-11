package handlers

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"restaurant-backend/internal/models"
)

// The signature is the only thing standing between a stranger's POST and a
// guest being charged, so every branch of it is pinned.
func TestAtmosSignature(t *testing.T) {
	const key = "s3cr3t-api-key"
	payload := atmosSignPayload("3", "998877", "MRC-YS19-1745", "10000000", key)

	// The documented order and concatenation, with nothing between the fields.
	if payload != "3998877MRC-YS19-174510000000"+key {
		t.Fatalf("payload = %q", payload)
	}

	// ⚠️ ATMOS documents the formula but never names the hash, so all three
	// plausible digests are accepted. That is not a weakening: the string being
	// hashed contains the api_key, so somebody who cannot produce one digest
	// cannot produce any of them.
	for name, sum := range map[string]string{
		"md5":    atmosHex(sum16(md5.Sum([]byte(payload)))),
		"sha1":   atmosHex(sum20(sha1.Sum([]byte(payload)))),
		"sha256": atmosHex(sum32(sha256.Sum256([]byte(payload)))),
	} {
		got, ok := atmosSignMatches(sum, payload)
		if !ok || got != name {
			t.Errorf("%s digest: got %q/%v, want accepted as %s", name, got, ok, name)
		}
		// Providers vary on case; ours must not.
		if _, ok := atmosSignMatches(strings.ToUpper(sum), payload); !ok {
			t.Errorf("%s uppercase rejected", name)
		}
	}

	// Anything else is refused — including an empty one, which is what an
	// unsigned probe sends.
	for _, bad := range []string{"", "  ", "deadbeef", atmosHex(sum16(md5.Sum([]byte(payload + "x"))))} {
		if _, ok := atmosSignMatches(bad, payload); ok {
			t.Errorf("accepted a bad signature: %q", bad)
		}
	}

	// The wrong key must not verify. This is the case that matters: an
	// installation that pasted the OAuth secret into the callback-key field
	// looks perfectly configured until the first real payment.
	other := atmosSignPayload("3", "998877", "MRC-YS19-1745", "10000000", "wrong-key")
	if _, ok := atmosSignMatches(atmosHex(sum16(md5.Sum([]byte(other)))), payload); ok {
		t.Error("a signature made with a different key verified")
	}
}

// Go will not slice [16]byte, [20]byte and [32]byte through one type
// parameter — they have different underlying types — so the callers slice.
func atmosHex(sum []byte) string { return hex.EncodeToString(sum) }

func sum16(a [16]byte) []byte { return a[:] }
func sum20(a [20]byte) []byte { return a[:] }
func sum32(a [32]byte) []byte { return a[:] }

// A gateway missing any one of its three secrets can never take money.
//
// The callback key is the one worth stating: without it ATMOS's confirmation
// cannot be verified, and an unverifiable confirmation has to be refused — so
// offering the guest that button would send them to a page that takes their
// card and then fails at the last step.
func TestAtmosConfiguredNeedsEverySecret(t *testing.T) {
	full := models.AtmosSettings{
		Enabled: true, StoreID: "3",
		ConsumerKey: "k", ConsumerSecret: "s", APIKey: "a",
	}
	s := &models.PaymentSettings{Atmos: full}
	if !s.Configured(models.ProviderAtmos) {
		t.Fatal("a fully configured gateway is not offered")
	}

	for name, mutate := range map[string]func(*models.AtmosSettings){
		"disabled":           func(a *models.AtmosSettings) { a.Enabled = false },
		"no store":           func(a *models.AtmosSettings) { a.StoreID = "" },
		"no consumer key":    func(a *models.AtmosSettings) { a.ConsumerKey = "" },
		"no consumer secret": func(a *models.AtmosSettings) { a.ConsumerSecret = "" },
		"no callback key":    func(a *models.AtmosSettings) { a.APIKey = "" },
	} {
		part := full
		mutate(&part)
		if (&models.PaymentSettings{Atmos: part}).Configured(models.ProviderAtmos) {
			t.Errorf("%s: offered to the guest anyway", name)
		}
	}
}

// The basket ATMOS bills for. Amounts are in tiyin, and the per-unit price is
// what goes on the line — sending the line total would double-charge every
// dish ordered twice on the fiscal receipt.
func TestAtmosInvoiceItems(t *testing.T) {
	order := &models.Order{
		Items: []models.OrderItem{
			{Name: "Lag'mon", Price: 32000, Qty: 2},
			{Name: "Choy", Price: 5000, Qty: 1},
		},
		DeliveryFee: 15000,
	}
	items := atmosInvoiceItems(order, nil)
	if len(items) != 3 {
		t.Fatalf("got %d lines, want 2 dishes + delivery", len(items))
	}
	if items[0].Amount != 3_200_000 || items[0].Quantity != 2 {
		t.Errorf("line 1 = %d × %d, want the unit price in tiyin", items[0].Amount, items[0].Quantity)
	}
	// Delivery is on the receipt so a guest comparing it with the total does
	// not find an unexplained difference.
	if items[2].Amount != 1_500_000 {
		t.Errorf("delivery line = %d", items[2].Amount)
	}
	// ⚠️ `details` must be an empty array, never null: Go marshals a nil slice
	// as `null`, and a gateway expecting a list is entitled to reject that.
	for i, it := range items {
		if it.Details == nil {
			t.Errorf("line %d has null details", i+1)
		}
	}

	// No delivery fee, no delivery line — a pickup receipt should not carry a
	// zero-som charge the guest has to ask about.
	pickup := &models.Order{Items: order.Items}
	if got := atmosInvoiceItems(pickup, nil); len(got) != 2 {
		t.Errorf("pickup produced %d lines, want 2", len(got))
	}
}

// An order with no lines at all must still produce a well-formed basket rather
// than `null`, which is what an empty menu or a promo-only order looks like.
func TestAtmosInvoiceItemsNeverNull(t *testing.T) {
	if got := atmosInvoiceItems(&models.Order{ID: primitive.NewObjectID()}, nil); got == nil {
		t.Fatal("nil basket")
	}
}

// The callback's numbers may arrive quoted or not, and both must verify.
//
// ⚠️ ATMOS documents the fields and the signature formula but never the JSON
// types. Declaring them as numbers refuses every callback that quotes them;
// declaring them as strings refuses every callback that does not. Either
// mistake is total and looks identical from outside — the guest reaches the
// payment page, pays, and is told it failed — so the reader takes both.
func TestAtmosCallbackAcceptsQuotedAndBareNumbers(t *testing.T) {
	type body struct {
		StoreID       atmosScalar `json:"store_id"`
		TransactionID atmosScalar `json:"transaction_id"`
		Amount        atmosScalar `json:"amount"`
	}
	for name, raw := range map[string]string{
		"bare":   `{"store_id":3,"transaction_id":998877,"amount":9200000}`,
		"quoted": `{"store_id":"3","transaction_id":"998877","amount":"9200000"}`,
		"mixed":  `{"store_id":3,"transaction_id":"998877","amount":9200000}`,
	} {
		var b body
		if err := json.Unmarshal([]byte(raw), &b); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if b.StoreID.String() != "3" || b.TransactionID.String() != "998877" ||
			b.Amount.String() != "9200000" {
			t.Errorf("%s: got %q/%q/%q", name,
				b.StoreID, b.TransactionID, b.Amount)
		}
	}

	// ⚠️ And the text is kept **exactly as written**: an amount sent as
	// "9200000.00" must sign as "9200000.00". Normalising it to an int and
	// printing it back would break a signature that was perfectly good.
	var b body
	if err := json.Unmarshal([]byte(`{"amount":9200000.00}`), &b); err != nil {
		t.Fatal(err)
	}
	if b.Amount.String() != "9200000.00" {
		t.Errorf("amount was reformatted to %q", b.Amount.String())
	}
}

// ⚠️ **A dish with no ИКПУ sends no `code` field at all**, and one that has a
// code sends exactly what the accountant entered.
//
// The gap is the whole design: we cannot derive a state classifier code from a
// dish name, and a plausible-looking placeholder is not a smaller mistake than
// an absent field — it is a fiscal receipt filed against the wrong product,
// which is the restaurant's problem with the tax office rather than ours.
// `omitempty` is what makes the absence real, so it is asserted on the encoded
// JSON rather than on the struct.
func TestAtmosInvoiceCarriesIkpuOnlyWhenKnown(t *testing.T) {
	withCode := primitive.NewObjectID()
	without := primitive.NewObjectID()
	order := &models.Order{Items: []models.OrderItem{
		{MenuItemID: withCode, Name: "Lag'mon", Price: 32000, Qty: 1},
		{MenuItemID: without, Name: "Choy", Price: 5000, Qty: 1},
	}}
	items := atmosInvoiceItems(order, map[primitive.ObjectID]string{
		withCode: "01234567890123456",
	})
	if items[0].Code != "01234567890123456" {
		t.Errorf("code = %q, want the entered one", items[0].Code)
	}
	if items[1].Code != "" {
		t.Errorf("code = %q, want empty for a dish with none", items[1].Code)
	}
	raw, err := json.Marshal(items[1])
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(`"code"`)) {
		t.Errorf("an unknown ИКПУ was sent as an empty field: %s", raw)
	}
}

// The form is copied off a spreadsheet, so it arrives with separators and the
// occasional wrong column. Anything that is not a 17-digit code clears the
// field — empty is a supported state, a nearly-right code is not.
func TestNormalizeIkpu(t *testing.T) {
	cases := map[string]string{
		"01234567890123456":     "01234567890123456",
		"0123 4567 8901 23456":  "01234567890123456",
		"01234-56789-0123456":   "01234567890123456",
		"0123456789012345":      "", // one short: half-typed
		"012345678901234567":    "", // one long: two codes run together
		"Lag'mon":               "", // the wrong column
		"01234567890123456 kod": "",
		"":                      "",
	}
	for in, want := range cases {
		if got := normalizeIkpu(in); got != want {
			t.Errorf("normalizeIkpu(%q) = %q, want %q", in, got, want)
		}
	}
}
