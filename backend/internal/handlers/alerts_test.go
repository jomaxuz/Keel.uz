package handlers

import (
	"strings"
	"testing"
	"time"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ⚠️ **A zero threshold is "never configured", not "alert on everything".**
// Reading it the other way would have every restaurant that has not opened the
// settings page alerting on every thousand-so'm tea — on the day this shipped,
// which is also the day everybody learns to mute the bot.
func TestUnsetThresholdsAreNotZero(t *testing.T) {
	got := models.AlertSettings{}.WithDefaults()
	if got.DiscountFrom <= 0 || got.CashShortFrom <= 0 ||
		got.StockShortFrom <= 0 || got.VoidFrom <= 0 || got.DailyMax <= 0 {
		t.Fatalf("an unconfigured branch would alert on everything: %+v", got)
	}
}

// ⚠️ **A restaurant's own numbers win over ours.** The defaults exist so a
// branch that never opens the page is quiet, not so we can override somebody
// who decided their own figure.
func TestConfiguredThresholdsAreKept(t *testing.T) {
	got := models.AlertSettings{DiscountFrom: 1, VoidFrom: 2, DailyMax: 3}.WithDefaults()
	if got.DiscountFrom != 1 || got.VoidFrom != 2 || got.DailyMax != 3 {
		t.Fatalf("the restaurant's own settings were overwritten: %+v", got)
	}
}

// ⚠️ **The order of the two events is the entire signal.** A line removed while
// the table is still eating is ordinary work; removed after the guest has been
// shown the total, it is a question. Getting this backwards would alert on
// every kitchen mistake in the building and on none of the ones that matter.
func TestOnlyVoidsAfterThePrecheckCount(t *testing.T) {
	src := readLossSource(t, "alerts.go")
	if !strings.Contains(src, "v.At.Before(*o.Check.PrecheckAt)") {
		t.Fatal("voids are no longer ordered against the moment the guest saw the bill")
	}
	if !strings.Contains(src, "o.Check.PrecheckAt == nil") {
		t.Fatal("a check that was never prechecked is being alerted on")
	}
}

// ⚠️ **Only shortfalls.** A drawer with more in it than expected is usually a
// sale rung on the wrong tender — worth looking at, not worth a phone buzzing
// in the evening. Mixing the two turns the useful message into one of the two
// the owner scrolls past.
func TestSurplusesRaiseNothing(t *testing.T) {
	cash := readLossSource(t, "cash.go")
	if !strings.Contains(cash, "if short := figures.Expected - req.Counted; short > 0 {") {
		t.Fatal("a cash surplus can now raise an alert")
	}
	stock := readLossSource(t, "stocktake.go")
	if !strings.Contains(stock, "if in.Value < 0 {") {
		t.Fatal("a stock surplus can now raise an alert")
	}
}

// ⚠️ **The daily ceiling counts what was *sent*, never what was recorded.** A
// bad night must keep producing records — that is what the panel reads — and
// only the buzzing stops. Counting records would make the list stop growing on
// exactly the night it matters.
func TestTheCeilingLimitsSendingNotRecording(t *testing.T) {
	src := readLossSource(t, "alerts.go")
	i := strings.Index(src, "sent, err := h.Store.LossAlerts.CountDocuments")
	if i < 0 {
		t.Fatal("the ceiling is gone")
	}
	window := src[i : i+400]
	if !strings.Contains(window, `"sentAt"`) {
		t.Fatal("the ceiling counts records rather than deliveries")
	}
	// And the insert happens before the ceiling is consulted.
	if strings.Index(src, "h.Store.LossAlerts.InsertOne") > i {
		t.Fatal("the record is written after the ceiling check — a busy night loses events")
	}
}

// ⚠️ **Owners only.** A manager is one of the people these messages are about,
// and sending one to them is not a leak of anything they do not know — it is a
// warning that they have been noticed.
func TestOnlyOwnersAreMessaged(t *testing.T) {
	src := readLossSource(t, "alerts.go")
	if !strings.Contains(src, `"role":        "owner"`) &&
		!strings.Contains(src, `"role": "owner"`) {
		t.Fatal("alerts are being sent to somebody other than an owner")
	}
}

// ⚠️ **What happened, who, how much — and no verdict.** Every kind has an
// ordinary explanation that happens weekly, and a message that concluded
// anything would be wrong often enough to be resented. Resented notifications
// get muted rather than argued with.
func TestTheMessageDrawsNoConclusion(t *testing.T) {
	a := models.LossAlert{
		Kind: models.AlertVoidAfterPrecheck, At: time.Now(),
		By: "Aziz", Amount: 120000, Subject: "Lag'mon · 6-stol",
		Reason: "mehmon qaytardi", ByID: primitive.NewObjectID(),
	}
	text := alertText(a, "Osh Markazi", "uz")

	for _, forbidden := range []string{
		"o'g'ir", "firibgar", "shubhali", "aybdor", "tekshiring", "jarima",
	} {
		if strings.Contains(strings.ToLower(text), forbidden) {
			t.Fatalf("the message accuses somebody: %q in %q", forbidden, text)
		}
	}
	for _, want := range []string{"Aziz", "120 000", "Lag'mon", "mehmon qaytardi"} {
		if !strings.Contains(text, want) {
			t.Fatalf("%q is missing from the message:\n%s", want, text)
		}
	}

	// ⚠️ **The language is the group's, and the facts are not translated.** A
	// Russian group gets Russian headings around the same name, the same
	// figure and the same reason somebody typed — inventing a translation for
	// what an employee wrote would be putting words in their mouth in a
	// message about their conduct.
	ru := alertText(a, "Osh Markazi", "ru")
	if !strings.Contains(ru, "После счёта") {
		t.Fatalf("a Russian group got an Uzbek heading:\n%s", ru)
	}
	if !strings.Contains(ru, "сум") {
		t.Fatalf("the currency stayed Uzbek:\n%s", ru)
	}
	for _, keep := range []string{"Aziz", "120 000", "mehmon qaytardi"} {
		if !strings.Contains(ru, keep) {
			t.Fatalf("%q was lost in translation:\n%s", keep, ru)
		}
	}
}

// ⚠️ **A struct rather than a map, so a language cannot silently miss a line** —
// and the line it would miss is the one added last, which is how a Russian
// group ends up with one Uzbek word in the middle of every message.
func TestEveryLanguageIsComplete(t *testing.T) {
	for _, lang := range NotifyLangs {
		w := notifyWordsFor(lang)
		for name, v := range map[string]string{
			"VoidAfterPrecheck": w.VoidAfterPrecheck, "BigDiscount": w.BigDiscount,
			"CashShort": w.CashShort, "StockShort": w.StockShort,
			"RecipeUp": w.RecipeUp, "Unknown": w.Unknown,
			"Who": w.Who, "Approved": w.Approved, "Reason": w.Reason,
			"Currency": w.Currency, "NotAnAccusation": w.NotAnAccusation,
			"FeedbackFrom": w.FeedbackFrom, "Order": w.Order,
		} {
			if strings.TrimSpace(v) == "" {
				t.Fatalf("%s is empty in %q", name, lang)
			}
		}
	}
	// An unknown language is Uzbek, not blank: a stored value from a future
	// version must not produce empty messages.
	if notifyWordsFor("de").Who != notifyWordsFor("uz").Who {
		t.Fatal("an unknown language does not fall back to Uzbek")
	}
}

// The thousands separator is the one every other figure in this product uses —
// a total that reads differently on the phone than on the screen is a total
// somebody re-checks.
func TestAmountsAreGrouped(t *testing.T) {
	for in, want := range map[int]string{
		0: "0", 999: "999", 1000: "1 000", 120000: "120 000", 1234567: "1 234 567",
	} {
		if got := formatSom(in); got != want {
			t.Fatalf("formatSom(%d) = %q, want %q", in, got, want)
		}
	}
}

// ⚠️ **The link token is derived, never stored**, so nothing has to be
// generated, expired or cleaned up — and it must actually differ per owner, or
// one link would bind every one of them to the same chat.
func TestEachOwnerGetsTheirOwnLink(t *testing.T) {
	a := alertLinkToken("secret", "aaaaaaaaaaaaaaaaaaaaaaaa")
	b := alertLinkToken("secret", "bbbbbbbbbbbbbbbbbbbbbbbb")
	if a == b {
		t.Fatal("two owners share a link token")
	}
	if a != alertLinkToken("secret", "aaaaaaaaaaaaaaaaaaaaaaaa") {
		t.Fatal("the token is not stable, so a link stops working after a restart")
	}
	if alertLinkToken("other", "aaaaaaaaaaaaaaaaaaaaaaaa") == a {
		t.Fatal("the token does not depend on the secret")
	}
}

// ⚠️ **A zero branch means "the company", and reading it as "no settings"
// switched the whole feature off for the events that matter most.**
//
// Settings are saved against a real branch. Two kinds of event have no branch:
// a recipe belongs to the brand, and a panel action belongs to whoever is
// logged in — and an owner of the whole company is pinned to no branch, so
// their id is zero. Reading zero found nothing, the defaults said `Enabled:
// false`, and every recipe edit and every data export was dropped before it was
// even recorded. The restaurant had turned the feature on and nothing arrived.
func TestAZeroBranchFallsBackToTheCompany(t *testing.T) {
	src := readLossSource(t, "alerts.go")
	i := strings.Index(src, "func (h *Handler) alertSettingsOf")
	if i < 0 {
		t.Fatal("the settings reader is gone")
	}
	body := src[i:]
	if j := strings.Index(body, "\n// raiseAlert "); j > 0 {
		body = body[:j]
	}
	if !strings.Contains(body, "!branchID.IsZero()") {
		t.Fatal("a zero branch is being looked up as though it were a branch")
	}
	// ⚠️ Falling back to *enabled* settings rather than to any settings: a
	// company with three branches where one has switched this on has switched
	// it on, and a branch that never opened the page holds defaults nobody
	// chose.
	if !strings.Contains(body, `bson.M{"enabled": true}`) {
		t.Fatal("the company fallback would pick a branch that never enabled it")
	}
}

// ⚠️ **Two settings govern one feature**, on two pages — a channel on the
// Telegram page and the switch here. A restaurant that configures the channel,
// tests it successfully and then hears nothing has done everything that looked
// like the job, so this screen has to say which half is still undone rather
// than leaving somebody to guess.
func TestTheSettingsPageSaysWhetherThereIsAChannel(t *testing.T) {
	src := readLossSource(t, "alerts.go")
	if !strings.Contains(src, `"hasChannel": hasChannel`) {
		t.Fatal("the page cannot tell an owner the other half is missing")
	}
	// A group counts, and so does an owner's own linked chat: either is
	// somewhere for a message to go.
	if !strings.Contains(src, "tg.AlertChatID != 0 || admin.AlertChatID != 0") {
		t.Fatal("one of the two ways to receive these is not counted")
	}
}
