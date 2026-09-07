package handlers

import (
	"strings"
	"testing"
)

// ⚠️ **1C reads the first word of the body and nothing else.** `success`,
// `progress`, `failure` — that is the whole protocol's result channel. A JSON
// error, which is what every other endpoint in this codebase returns, is read
// by 1C as a failure whose reason is our JSON; the accountant then reads
// `{"error":"..."}` down the phone. So these handlers deliberately answer in
// plain text, and this pins it where somebody would "tidy" it.
func TestTheExchangeAnswersInPlainTextRatherThanJSON(t *testing.T) {
	src := readSource(t, "onec.go")
	fn := between(t, src, "func (h *Handler) OneCExchange", "\n}\n")
	if strings.Contains(fn, "httpx.JSON") || strings.Contains(fn, "httpx.Error") {
		t.Fatal("the exchange answers 1C with JSON")
	}
	if !strings.Contains(src, `oneCText(w, "success\n"+name+"\n"+value+"\n")`) {
		t.Error("checkauth no longer answers with the three positional lines 1C reads")
	}
	// ⚠️ **A refusal is HTTP 200 with `failure` in the body.** A 500 makes 1C
	// report a network problem, which sends the accountant to their IT person
	// instead of to the sentence that says what is actually wrong.
	fail := between(t, src, "func oneCFail", "\n}\n")
	if !strings.Contains(fail, `"failure\n"`) {
		t.Error("a refusal does not start with the word 1C looks for")
	}
	if strings.Contains(fail, "StatusInternalServerError") {
		t.Error("a refusal is sent as a server error")
	}
}

// ⚠️ **The credentials are checked before the mode is read.** This is the one
// endpoint in the product that is not behind the panel's token — 1C cannot hold
// one — so a wrong answer here is a data leak rather than a broken screen.
func TestTheDoorIsCheckedBeforeAnythingElse(t *testing.T) {
	fn := between(t, readSource(t, "onec.go"),
		"func (h *Handler) OneCExchange", "\n}\n")
	auth := strings.Index(fn, "oneCAuthorised")
	mode := strings.Index(fn, `Query().Get("mode")`)
	if auth < 0 || mode < 0 || auth > mode {
		t.Fatal("the exchange reads what was asked before checking who is asking")
	}
	// The password is compared as a bcrypt hash, never as text.
	if !strings.Contains(readSource(t, "onec.go"), "bcrypt.CompareHashAndPassword") {
		t.Error("the exchange password is not compared as a hash")
	}
}

// ⚠️ **A catalogue that arrives short of a line is not a product a restaurant
// has stopped selling** — far more often it is a filter somebody left on in 1C.
// A delete here would take a technical card, a shelf balance and a year of
// purchase history with it.
func TestAnImportNeverDeletes(t *testing.T) {
	fn := between(t, readSource(t, "onec.go"),
		"func (h *Handler) oneCWriteProducts", "\n}\n")
	for _, destructive := range []string{"DeleteMany", "DeleteOne", "isActive\": false"} {
		if strings.Contains(fn, destructive) {
			t.Errorf("the catalogue import removes products (%s)", destructive)
		}
	}
	// ⚠️ And a price of zero does not overwrite a price: an offers file without
	// prices is ordinary, and reading its silence as "free" would zero the cost
	// of every dish the ingredient goes into.
	if !strings.Contains(fn, "if p.Price > 0 {") {
		t.Error("a missing price would be written as zero")
	}
}

// ⚠️ **Both halves of the ledger, and as two kinds of document.** A month's
// takings without the month's purchases is the half that makes any business
// look enormously profitable — and an accountant who types the other half by
// hand is doing the work this integration exists to remove.
func TestTheExchangeSendsSalesAndDeliveriesBoth(t *testing.T) {
	fn := between(t, readSource(t, "onec.go"),
		"func (h *Handler) oneCDocuments", "\n}\n")
	if !strings.Contains(fn, "onec.OpSale") || !strings.Contains(fn, "onec.OpPurchase") {
		t.Fatal("only one side of the books is exported")
	}
	// ⚠️ Local time, because Mongo hands every date back in UTC: a sale at
	// 19:00 in Tashkent would otherwise be filed on the previous day — and at
	// the end of a month, in the previous month.
	if strings.Count(fn, ".In(time.Local)") < 2 {
		t.Error("a document date is used as it came out of Mongo")
	}
	// A till check is an order with a check on it (models/check.go), so one
	// query covers the counter and the website. If that ever stops being true,
	// this is the test that should fail.
	if !strings.Contains(fn, "models.StatusDelivered") {
		t.Error("the exchange no longer selects completed sales")
	}
}

// ⚠️ **`mode=success` marks nothing.** 1C is saying "I have them"; our sales
// and deliveries exist whether or not an accountant imported them. A flag here
// would create a second meaning for "this sale happened" and a support call the
// first time somebody re-imports a month.
func TestTellingUsItArrivedChangesNothing(t *testing.T) {
	fn := between(t, readSource(t, "onec.go"),
		"func (h *Handler) OneCExchange", "\n}\n")
	i := strings.Index(fn, `case "success":`)
	if i < 0 {
		t.Fatal("the success mode is gone")
	}
	rest := fn[i:]
	if j := strings.Index(rest, "case "); j > 0 {
		rest = rest[:j]
	}
	if strings.Contains(rest, "UpdateOne") || strings.Contains(rest, "UpdateMany") {
		t.Error("the acknowledgement writes to the restaurant's own documents")
	}
}
