package handlers

import (
	"strings"
	"testing"
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
	real := smsCodeText("123456")
	test := smsTestText()

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
	if eskizProbeText == smsTestText() {
		t.Fatal("the probe must stay a different question from the real test")
	}
	if !strings.Contains(eskizProbeText, "Eskiz") {
		t.Fatal("the probe must be one of Eskiz's accepted texts verbatim")
	}
}
