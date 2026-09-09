package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"strings"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/i18n"
)

// ⚠️ The one thing that cannot be checked by reading the code: whether the
// wrapper actually reaches `httpx.Error`. Added before the logger it would be
// buried, the assertion would find nothing, and every message would stay Uzbek
// with no other symptom.
func TestErrorsAreAnsweredInTheCallersLanguage(t *testing.T) {
	h := Lang(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.Error(w, http.StatusConflict, "ochiq smena yo'q")
	}))

	for _, tc := range []struct{ header, cookie, accept, want string }{
		// ⚠️ **The header alone, with no cookie at all** — the shape every
		// cross-origin call and every phone request actually has. This is the
		// case that was broken: the panel set a `lang` cookie the API never
		// received, and a dashboard read in Russian was answered in Uzbek with
		// nothing anywhere reporting a fault.
		{"ru", "", "", "нет открытой смены"},
		{"en", "", "", "no shift is open"},
		// The header outranks both: only a client that knows which language it
		// is drawing in sends it, and a cookie left by another tab is staler.
		{"ru", "uz", "en-US", "нет открытой смены"},
		// Something we do not answer in falls through rather than winning.
		{"tr", "ru", "", "нет открытой смены"},
		{"", "ru", "", "нет открытой смены"},
		{"", "en", "", "no shift is open"},
		{"", "uz", "", "ochiq smena yo'q"},
		{"", "", "", "ochiq smena yo'q"},
		// No cookie yet — the first request from a mini app or a tracking link.
		{"", "", "ru-RU,ru;q=0.9", "нет открытой смены"},
		// ⚠️ The cookie wins over the machine's own preference: a cashier on a
		// Russian Windows who set the till to Uzbek gets what they chose.
		{"", "uz", "ru-RU", "ochiq smena yo'q"},
	} {
		req := httptest.NewRequest("GET", "/", nil)
		if tc.header != "" {
			req.Header.Set(LangHeader, tc.header)
		}
		if tc.cookie != "" {
			req.AddCookie(&http.Cookie{Name: LangCookie, Value: tc.cookie})
		}
		if tc.accept != "" {
			req.Header.Set("Accept-Language", tc.accept)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		var body struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Error != tc.want {
			t.Errorf("header=%q cookie=%q accept=%q: %q kutilgan, %q keldi",
				tc.header, tc.cookie, tc.accept, tc.want, body.Error)
		}
	}
}

// ⚠️ The other half of the same wrapper: some answers put a sentence in a
// *field*, not in an error. `permissionName` is the one the till shows beside
// its PIN pad — "Chegirma berish uchun ruxsat kerak" — and it never passes
// through httpx.Error, which is how it stayed Uzbek on a Russian till after
// everything around it had been translated.
func TestFieldsCarryingSentencesAreTranslatedToo(t *testing.T) {
	h := Lang(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusConflict, map[string]any{
			"permissionName": i18n.Localize(httpx.LangOf(w), "Chegirma berish"),
		})
	}))

	for _, tc := range []struct{ lang, want string }{
		{"uz", "Chegirma berish"},
		{"ru", "Скидка"},
		{"en", "Giving a discount"},
	} {
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: LangCookie, Value: tc.lang})
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		var body struct {
			Name string `json:"permissionName"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Name != tc.want {
			t.Errorf("%s: %q kutilgan, %q keldi", tc.lang, tc.want, body.Name)
		}
	}
}

// ⚠️ An unwrapped writer — a handler called directly from a test — has to keep
// answering Uzbek rather than panicking or returning an empty string. That is
// what every handler test in this repository already asserts against.
func TestAnUnwrappedWriterStaysUzbek(t *testing.T) {
	rec := httptest.NewRecorder()
	if got := httpx.LangOf(rec); got != i18n.UZ {
		t.Fatalf("o'ralmagan yozuvchi %q qaytardi", got)
	}
	httpx.Error(rec, http.StatusConflict, "chek bo'sh")
	if !strings.Contains(rec.Body.String(), "chek bo'sh") {
		t.Fatalf("xabar o'zgardi: %s", rec.Body.String())
	}
}
