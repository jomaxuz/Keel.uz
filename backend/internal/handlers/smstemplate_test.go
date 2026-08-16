package handlers

import (
	"strings"
	"testing"

	"restaurant-backend/internal/models"
)

// ⚠️ The test message must be the **real** template with a dummy code.
//
// Gateways moderate an exact wording, not an account: Eskiz and Play Mobile
// approve one string and refuse everything else. This used to be two separately
// written strings — the test sent "Test: SMS sozlamalari tekshirilmoqda…" while
// logins sent "Tasdiqlash kodi: …" — which meant the test could pass on an
// account whose real login codes were still being rejected, i.e. the page whose
// whole purpose is catching that failure certified it as working.
//
// Sealed as a relationship rather than a literal, so rewording the message
// stays a one-line change and keeps the two in step.
func TestTestSMSUsesTheRealTemplate(t *testing.T) {
	real := smsCodeText(defaultSMSTemplate, "123456")
	test := smsTestText(defaultSMSTemplate)

	if strings.Contains(test, "123456") {
		t.Fatal("the test must not carry a real code")
	}
	// Same sentence, different digits: strip the codes and they must match.
	if strip(real, "123456") != strip(test, "000000") {
		t.Fatalf("test text drifted from the login text:\n  login: %s\n  test:  %s",
			real, test)
	}
	if !strings.Contains(test, "000000") {
		t.Fatal("the test text should carry a visibly dummy code")
	}
}

func strip(s, code string) string { return strings.ReplaceAll(s, code, "") }

// The refusal every fresh Eskiz account hits. Recognised so the panel can say
// what it means and what to do, instead of showing the owner a Russian JSON
// blob that reads as a broken integration rather than an unfinished signup.
func TestEskizModerationRefusalIsRecognised(t *testing.T) {
	body := `eskiz send: 400 {"id":"6bf81cd4","message":"Для теста можно ` +
		`использовать только один из этих текстов: Bu Eskiz dan test, ` +
		`Это тест от Eskiz, This is test from Eskiz","status":"error"}`

	if !eskizNeedsModeration("eskiz", body) {
		t.Fatal("the moderation refusal must be recognised")
	}
	// Not claimed for anybody else: the other gateways refuse for their own
	// reasons and telling their owners to visit an Eskiz cabinet is worse than
	// showing them the raw error.
	if eskizNeedsModeration("playmobile", body) {
		t.Fatal("only Eskiz has an Eskiz cabinet")
	}
	// An ordinary Eskiz failure — no balance, wrong sender — must keep its own
	// wording. Overwriting it with moderation advice would send the owner to
	// fix something that is not wrong.
	if eskizNeedsModeration("eskiz", `eskiz send: 400 {"message":"balance"}`) {
		t.Fatal("an unrelated failure must not be relabelled as moderation")
	}
}

// The probe text is Eskiz's fixed wording, and it is deliberately not the
// template: it answers "do the credentials reach Eskiz", not "will a login code
// arrive". If these ever converged, the probe would start certifying the thing
// it cannot check.
func TestProbeTextIsNotTheTemplate(t *testing.T) {
	if eskizProbeText == smsTestText(defaultSMSTemplate) {
		t.Fatal("the probe must stay a different question from the real test")
	}
	if !strings.Contains(eskizProbeText, "Eskiz") {
		t.Fatal("the probe must be one of Eskiz's accepted texts verbatim")
	}
}

// The setting resolves to the built-in whenever it is empty — the zero value has
// to be today's behaviour, the same rule as an empty mapProvider meaning 2GIS.
// Every install that has never opened the page has this field absent.
func TestTemplateFallsBackToDefault(t *testing.T) {
	if smsTemplateOf(nil) != defaultSMSTemplate {
		t.Fatal("no settings at all must still send something")
	}
	if smsTemplateOf(&models.SMSSettings{}) != defaultSMSTemplate {
		t.Fatal("an empty template must mean the built-in wording")
	}
	if smsTemplateOf(&models.SMSSettings{CodeTemplate: "   "}) != defaultSMSTemplate {
		t.Fatal("whitespace is empty")
	}

	custom := "Kod: {code} — Osh Markazi"
	if smsTemplateOf(&models.SMSSettings{CodeTemplate: custom}) != custom {
		t.Fatal("a real template must be used as written")
	}
}

// ⚠️ A template with no placeholder sends a message with no code in it: the
// gateway accepts it, the SMS arrives, and the guest simply cannot sign in.
// Nothing errors anywhere. AdminUpdateSMS refuses to store one; this is the
// second guard, for a document written before that check existed or by hand.
func TestTemplateWithoutPlaceholderIsIgnored(t *testing.T) {
	broken := &models.SMSSettings{CodeTemplate: "Tasdiqlash kodi keldi."}
	if smsTemplateOf(broken) != defaultSMSTemplate {
		t.Fatal("a template with no {code} must fall back, not ship")
	}
	out := smsCodeText("Tasdiqlash kodi keldi.", "424242")
	if !strings.Contains(out, "424242") {
		t.Fatalf("the code must reach the guest whatever is stored, got %q", out)
	}
}

// The default is Latin Uzbek on purpose, and both halves matter: Uzbek is
// understood in the regions where Russian thins out, and plain Latin stays
// inside GSM-7 — a 160-character message rather than the 70 a single Cyrillic
// letter or `oʻ` cuts it to. At 48 characters the cost is one part either way;
// what Latin buys is the headroom an owner spends adding their restaurant name.
func TestDefaultTemplateStaysInGSM7(t *testing.T) {
	msg := smsTestText(defaultSMSTemplate)
	if !isGSM7(msg) {
		t.Fatalf("the default must not leave GSM-7: %q", msg)
	}
	if n := smsParts(msg); n != 1 {
		t.Fatalf("the default must cost one part, got %d", n)
	}
	// The cliff itself, sealed so the reasoning above stays checkable: one
	// Cyrillic word is enough to halve the limit.
	if isGSM7("Код: {code}") {
		t.Fatal("Cyrillic is not GSM-7 — the headroom argument depends on this")
	}
}
