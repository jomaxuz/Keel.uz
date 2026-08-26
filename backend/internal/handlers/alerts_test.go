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

// ⚠️ **Cooked food is the test, not the printed bill — and this assertion was
// the other way round for a week.**
//
// The first version required a precheck, reasoning that removing a line before
// the guest sees the bill is ordinary work. Half right: removing a line
// *nobody has cooked* is ordinary work, and `FiredAt` is what tests that.
// Removing one the kitchen has already made is food the restaurant paid for —
// the sentence written on `CheckLineVoid` itself, which calls it the single
// event a till exists to record. A restaurant voided a main course and heard
// nothing, because no bill had been printed yet.
func TestVoidsOfCookedFoodCount(t *testing.T) {
	src := readLossSource(t, "alerts.go")
	if !strings.Contains(src, "if v == nil || it.FiredAt == nil {") {
		t.Fatal("a dish nobody cooked is being alerted on, or a cooked one is not")
	}
	// ⚠️ The precheck is still carried — in the words, where it belongs: the
	// guest had been shown a total and then the total went down. It makes the
	// event worse; it was never what made it worth knowing.
	if !strings.Contains(src, "o.Check.PrecheckAt != nil && v.At.After(*o.Check.PrecheckAt)") {
		t.Fatal("the strongest fact about a void is no longer said")
	}
}

// ⚠️ **The case this whole feature was asked for, and the one it shipped
// without.** "Take the cash and cancel the check as a mistake" is the first
// thing anybody describes when asked how a cashier steals — and every trigger
// hung on the *close* path, which a cancelled check never reaches. Six kinds of
// alert, and the headline one was missing.
func TestACancelledCheckRaisesOne(t *testing.T) {
	src := readLossSource(t, "tillclose.go")
	if !strings.Contains(src, "h.alertOnCancelledCheck(o, who, reason)") {
		t.Fatal("cancelling a check tells the owner nothing again")
	}
	// ⚠️ Only a check something was actually cooked for: a table opened by
	// mistake and closed again is the commonest cancellation in any restaurant,
	// and alerting on it would put a message on a phone several times a day.
	if !strings.Contains(src, "it.Live() && it.FiredAt != nil") {
		t.Fatal("an empty check cancelled would now raise an alert")
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

// ⚠️ **The update map is not the order, and forgetting that made the alert
// silent while everything else was right.**
//
// The till discount is written into `set`, which is what reaches Mongo — and
// `alertOnDiscount` runs a few lines later reading `o.Discounts`, which had
// never been told. A restaurant took 585 000 so'm off two bills; the database
// had it, the response had it, the receipt printed it, and the one thing meant
// to notice was handed an empty slice.
//
// The alert is raised from the in-memory order because that is what the close
// handler holds, so the two have to be kept in step at the point they diverge.
func TestTheDiscountReachesTheOrderNotOnlyTheUpdate(t *testing.T) {
	src := readLossSource(t, "tillclose.go")
	setAt := strings.Index(src, `set["discounts"] = []models.OrderDiscount{{`)
	mirror := strings.Index(src, "o.Discounts = d")
	alert := strings.Index(src, "h.alertOnDiscount(o, aset)")
	if setAt < 0 || mirror < 0 || alert < 0 {
		t.Fatal("the discount, its mirror or the alert is gone")
	}
	if mirror < setAt {
		t.Fatal("the mirror runs before the discount is built")
	}
	if alert < mirror {
		t.Fatal("the alert reads the order before the discount was put on it")
	}
}

// A rule-driven discount still carries no name and still measures nobody — the
// mirror must not have changed that.
func TestTheMirrorDoesNotInventAnActor(t *testing.T) {
	src := readLossSource(t, "alerts.go")
	if !strings.Contains(src, `if d.ByID.IsZero() && d.By == "" {`) {
		t.Fatal("a promotion that matched would now be reported as somebody's judgement")
	}
}
