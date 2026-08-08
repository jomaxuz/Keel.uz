package handlers

import (
	"strings"
	"testing"

	"restaurant-backend/internal/models"
)

// A guest who gets five pings for one order stops reading the one that matters,
// so which statuses speak at all is a product decision worth pinning.
func TestOnlyTheStatusesAGuestActsOnAreNotified(t *testing.T) {
	for _, s := range []models.OrderStatus{
		models.StatusConfirmed, models.StatusOnTheWay,
		models.StatusDelivered, models.StatusCancelled,
	} {
		if !notifiableStatus(s) {
			t.Errorf("%s haqida xabar berilmaydi", s)
		}
	}
	// `preparing` is the kitchen's own step and `pending` is an order nobody has
	// accepted yet — telling a guest about either is noise at best.
	for _, s := range []models.OrderStatus{
		models.StatusPending, models.StatusPreparing,
	} {
		if notifiableStatus(s) {
			t.Errorf("%s haqida xabar berilmasligi kerak", s)
		}
	}
}

func TestOrderStatusMessageSpeaksTheGuestsLanguage(t *testing.T) {
	cases := []struct{ lang, want string }{
		{"uz", "qabul qilindi"},
		{"ru", "принят"},
		{"en", "accepted"},
		// Telegram sends region codes and things we do not speak; Uzbek is the
		// base everywhere else in this app, so it is the base here too.
		{"ru-RU", "принят"},
		{"en-GB", "accepted"},
		{"de", "qabul qilindi"},
		{"", "qabul qilindi"},
	}
	for _, c := range cases {
		got := orderStatusMessage(normalizeLang(c.lang), "Osh Markazi", "AB12-3456",
			models.StatusConfirmed, "")
		if !strings.Contains(got, c.want) {
			t.Errorf("lang=%q: %q ichida %q yo'q", c.lang, got, c.want)
		}
		// The number and the restaurant are in every message: a guest with two
		// orders open needs to know which one this is about.
		if !strings.Contains(got, "AB12-3456") || !strings.Contains(got, "Osh Markazi") {
			t.Errorf("lang=%q: raqam yoki restoran nomi yo'q: %q", c.lang, got)
		}
	}
}

// A cancellation without its reason is the message most likely to produce a
// phone call, so the reason travels with it — in all three languages.
func TestCancellationCarriesTheReason(t *testing.T) {
	for _, lang := range []string{"uz", "ru", "en"} {
		got := orderStatusMessage(lang, "Osh", "X1", models.StatusCancelled,
			"taomlar tugadi")
		if !strings.Contains(got, "taomlar tugadi") {
			t.Errorf("%s: sabab yo'qoldi: %q", lang, got)
		}
		// And with no reason recorded the message still reads as a sentence.
		bare := orderStatusMessage(lang, "Osh", "X1", models.StatusCancelled, "")
		if bare == "" || strings.HasSuffix(strings.TrimSpace(bare), ":") {
			t.Errorf("%s: sababsiz xabar buzuq: %q", lang, bare)
		}
	}
}

func TestUnknownStatusSaysNothing(t *testing.T) {
	if got := orderStatusMessage("uz", "Osh", "X1", models.StatusPreparing, ""); got != "" {
		t.Errorf("kutilmagan xabar: %q", got)
	}
}
