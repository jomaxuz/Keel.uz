package telegram

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

const testToken = "123456:AAH-fake-bot-token-for-tests"

// The signature is the login, so every test here is a way somebody could try to
// be somebody else.

func TestVerifyAcceptsWhatTelegramSigned(t *testing.T) {
	fields := map[string]string{
		"auth_date": fmt.Sprint(time.Now().Unix()),
		"user":      `{"id":12345,"first_name":"Ali","username":"ali"}`,
		"query_id":  "AAH123",
	}
	got, err := Verify(Sign(fields, testToken), testToken)
	if err != nil {
		t.Fatalf("haqiqiy imzo rad etildi: %v", err)
	}
	if got["query_id"] != "AAH123" {
		t.Errorf("maydonlar yo'qoldi: %#v", got)
	}
	u, err := ParseUser(got)
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != 12345 || u.FirstName != "Ali" {
		t.Errorf("foydalanuvchi noto'g'ri o'qildi: %+v", u)
	}
}

func TestVerifyRefusesTamperingAndTheWrongBot(t *testing.T) {
	fields := map[string]string{
		"auth_date": fmt.Sprint(time.Now().Unix()),
		"user":      `{"id":12345,"first_name":"Ali"}`,
	}
	signed := Sign(fields, testToken)

	// The attack this exists to stop: keep the signature, change the person.
	swapped := strings.Replace(signed, "12345", "99999", 1)
	if _, err := Verify(swapped, testToken); err == nil {
		t.Error("boshqa foydalanuvchi id'si bilan o'zgartirilgan payload o'tdi")
	}

	// Signed by a different bot — a real case on a shared platform, where a
	// payload from one restaurant's bot must not log anybody into another's.
	if _, err := Verify(Sign(fields, "999:OTHER-BOT"), testToken); err == nil {
		t.Error("boshqa botning imzosi qabul qilindi")
	}

	// No hash at all.
	if _, err := Verify("user=%7B%22id%22%3A1%7D&auth_date=1", testToken); err == nil {
		t.Error("imzosiz payload o'tdi")
	}

	// And with no bot connected nothing is ever accepted, whatever the payload
	// looks like: an install without a token has no Telegram login, rather than
	// one that trusts everybody.
	if _, err := Verify(signed, ""); err != ErrNoToken {
		t.Errorf("tokensiz holatda: %v", err)
	}
}

// ⚠️ Without an age check a captured initData is a permanent password for that
// account — and it lives in the browser, where a screenshot or an extension can
// take it.
func TestCheckFreshBoundsHowLongALoginLives(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	fresh := map[string]string{"auth_date": fmt.Sprint(now.Add(-time.Minute).Unix())}
	if err := CheckFresh(fresh, now, MaxAuthAge); err != nil {
		t.Errorf("yangi payload rad etildi: %v", err)
	}

	old := map[string]string{"auth_date": fmt.Sprint(now.Add(-48 * time.Hour).Unix())}
	if err := CheckFresh(old, now, MaxAuthAge); err != ErrStale {
		t.Errorf("eski payload o'tdi: %v", err)
	}

	// A date in the future is a forged one somebody hopes will still work next
	// year. Small skew is normal and allowed.
	future := map[string]string{"auth_date": fmt.Sprint(now.Add(time.Hour).Unix())}
	if err := CheckFresh(future, now, MaxAuthAge); err != ErrStale {
		t.Errorf("kelajakdagi sana o'tdi: %v", err)
	}
	skew := map[string]string{"auth_date": fmt.Sprint(now.Add(time.Minute).Unix())}
	if err := CheckFresh(skew, now, MaxAuthAge); err != nil {
		t.Errorf("kichik soat farqi rad etildi: %v", err)
	}

	// No date cannot be aged out, so it is refused rather than trusted.
	if err := CheckFresh(map[string]string{}, now, MaxAuthAge); err != ErrStale {
		t.Error("sanasiz payload qabul qilindi")
	}
}

// Sorting is not cosmetic: Telegram signs the fields in sorted order, so a
// verifier that joined them in map order would reject every real login — and it
// would do so intermittently, because Go randomises map iteration.
func TestVerifyIsOrderIndependent(t *testing.T) {
	fields := map[string]string{
		"auth_date": "1800000000",
		"user":      `{"id":7}`,
		"zebra":     "last",
		"alpha":     "first",
		"query_id":  "q",
	}
	signed := Sign(fields, testToken)
	for i := 0; i < 20; i++ {
		if _, err := Verify(signed, testToken); err != nil {
			t.Fatalf("%d-urinishda yiqildi: %v", i, err)
		}
	}
}

func TestParseContactReadsBothShapes(t *testing.T) {
	// The shape Telegram documents.
	c, err := ParseContact(map[string]string{
		"contact": `{"user_id":42,"phone_number":"998901234567"}`,
	})
	if err != nil || c.Phone != "998901234567" || c.UserID != 42 {
		t.Errorf("JSON ko'rinish o'qilmadi: %+v %v", c, err)
	}

	// And the flat one some clients send. Read leniently on purpose: the
	// signature is what makes the number trustworthy, and being strict about the
	// wrapper would reject a real, signed phone number.
	c, err = ParseContact(map[string]string{
		"phone_number": "998907654321", "user_id": "7",
	})
	if err != nil || c.Phone != "998907654321" || c.UserID != 7 {
		t.Errorf("tekis ko'rinish o'qilmadi: %+v %v", c, err)
	}

	if _, err := ParseContact(map[string]string{"user": `{"id":1}`}); err == nil {
		t.Error("telefonsiz payload qabul qilindi")
	}
}

// ParseUser must never be reachable with unverified input, and it must refuse a
// payload with no identity rather than inventing user 0.
func TestParseUserRefusesNonsense(t *testing.T) {
	for _, f := range []map[string]string{
		{},
		{"user": "not json"},
		{"user": `{"first_name":"Ali"}`}, // no id
	} {
		if _, err := ParseUser(f); err == nil {
			t.Errorf("qabul qilindi: %#v", f)
		}
	}
}

// ⚠️ Which update types the bot asks Telegram for.
//
// Sealed because omitting one is the bug with no symptom: the greeting's language
// buttons were drawn, tapping them did nothing at all, nothing was logged, and
// the settings page showed a perfectly healthy bot — the stale registration was
// held by Telegram, where none of our own state can see it.
func TestWebhookAsksForCallbacks(t *testing.T) {
	want := map[string]bool{"message": true, "callback_query": true}
	got := map[string]bool{}
	for _, u := range WebhookUpdates {
		got[u] = true
	}
	for u := range want {
		if !got[u] {
			t.Fatalf("%q is not in WebhookUpdates: a feature that needs it will "+
				"silently do nothing (got %v)", u, WebhookUpdates)
		}
	}
}
