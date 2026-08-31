// Package i18n translates the messages the API sends back to a person.
//
// ⚠️ **The whole panel, till and staff app are trilingual and the errors were
// not.** A Russian cashier could read every label on the till and then be told
// "ochiq smena yo'q" when the shift was not open — at a counter, mid-queue,
// with a guest waiting. The screens had been translated; the one sentence that
// only ever appears when something is already going wrong had not.
//
// ⚠️ **The Uzbek message is the key.** The alternative — an error code beside
// every message — means editing 1295 call sites, and any one of them missed
// stays Uzbek silently. Keying on the text itself leaves those call sites
// exactly as they are: `httpx.Error(w, 409, "ochiq smena yo'q")` reads the same
// and now answers in the caller's language.
//
// ⚠️ **The obvious risk of that is guarded by a test.** Rewording a message in
// a handler would quietly drop its translation, so `errors_test.go` reads every
// literal out of `internal/handlers` and fails when one has no entry here. The
// failure is a red test, not a Russian screen showing an Uzbek sentence.
//
// ⚠️ **Not everything here is translated, and the untranslated ones are
// listed rather than forgotten.** "invalid id", "forbidden", "bad request" and
// their kin answer a malformed request, not a person: they are read in a
// network panel by whoever wrote the caller. Translating them would suggest a
// cashier is meant to act on one.
package i18n

// Lang is one of the three the product speaks. Anything else is Uzbek, which
// is the language every message is already written in.
const (
	UZ = "uz"
	RU = "ru"
	EN = "en"
)

type pair struct{ ru, en string }

// Localize returns msg in lang, or msg itself when there is nothing better.
//
// ⚠️ **Falling back to the Uzbek text is deliberate and is the old behaviour.**
// A missing entry then shows what it always showed rather than an empty string
// or a raw key — the reader loses the translation, not the sentence.
func Localize(lang, msg string) string {
	if lang == UZ || lang == "" {
		return msg
	}
	t, ok := messages[msg]
	if !ok {
		return msg
	}
	if lang == RU {
		return t.ru
	}
	if lang == EN {
		return t.en
	}
	return msg
}

// Untranslated are the messages that answer a broken request rather than a
// person, and are left in place on purpose. Kept as a list so the test can
// tell "decided against" apart from "not done yet" — the difference between
// those two is the whole value of the test.
var Untranslated = map[string]bool{
	"bad request":              true,
	"check not found":          true,
	"file too large":           true,
	"forbidden":                true,
	"invalid adminId":          true,
	"invalid courier id":       true,
	"invalid credentials":      true,
	"invalid id":               true,
	"invalid operatorId":       true,
	"invalid provider id":      true,
	"invalid serverId":         true,
	"invalid token":            true,
	"invalid userId":           true,
	"missing file field":       true,
	"not found":                true,
	"order not found":          true,
	"restaurant not configured": true,
	"unauthorized":             true,
	"unsupported file type":    true,
	"unsupported language":     true,
	"user not found":           true,
}
