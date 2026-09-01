package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"

	"restaurant-backend/internal/i18n"
)

var validate = validator.New()

// errBadJSON is what a malformed body is answered with, in place of Go's raw
// parser message.
//
// ⚠️ Small on purpose, but it is an information leak: echoing
// "invalid character 'b' looking for beginning of value" back to the client
// tells a prober the backend is Go with encoding/json, which is one free hint
// toward everything else. The validator's messages are kept — they name the
// caller's own missing field, which is theirs to see and helps them fix the
// request — but the decoder's internal position errors are not.
var errBadJSON = errors.New("so'rov formati noto'g'ri")

// JSON writes v as a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// LangOf is the language this response is being written in.
//
// ⚠️ **For the answers that carry a sentence in a field rather than in an
// error** — `permissionName`, the line the till puts beside its PIN pad. Those
// never pass through `Error`, which is why they stayed Uzbek on a Russian till
// long after every label around them had been translated.
//
// The assertion is against a method so this package does not import the
// middleware that supplies it; an unwrapped writer answers Uzbek, which is what
// every response did before any of this existed.
func LangOf(w http.ResponseWriter) string {
	if lw, ok := w.(interface{ Lang() string }); ok {
		return lw.Lang()
	}
	return i18n.UZ
}

// T is the message a response carries in a **field** rather than in an error.
//
// ⚠️ **The second way out, and the one that kept staying Uzbek.** A connection
// check does not fail — "the domain does not point here yet" is the answer that
// screen exists to give — so it is written with `JSON`, and `JSON` translates
// nothing. Every one of those sentences (POS, PBX, Telegram, fiscal, domain,
// import) sat beside labels that were translated, and only appeared on the day
// something was wrong, which is the only day anybody reads them.
//
// Same catalogue, same key, same fallback: what changes is that the writer is
// asked for the language here instead of inside `Error`.
func T(w http.ResponseWriter, msg string) string {
	return i18n.Localize(LangOf(w), msg)
}

// Error writes a JSON error envelope, in the language the caller reads.
//
// ⚠️ **The translation happens here, in the one place a message is written,
// and every call site is untouched.** There are more than a thousand of them;
// an error code beside each would be a thousand edits and any one missed would
// stay Uzbek silently. See internal/i18n — the Uzbek text is the key, and a
// test fails when a handler grows a message the catalogue does not have.
//
// ⚠️ The assertion is against a method, not against a concrete type: this
// package must not import the middleware that supplies it, and a writer that
// does not answer simply gets the message as written — which is what every
// response did before this existed.
func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": i18n.Localize(LangOf(w), msg)})
}

// Decode parses the JSON request body into dst and validates it.
// DecodeOptional reads a body that may not be there.
//
// ⚠️ **Not `if r.ContentLength > 0`, which is what every caller used to write.**
// A chunked request reports a length of **-1**, so that test skips a body that
// is genuinely present — and the till's own requests arrive chunked once they
// have been through the Windows app's proxy. The symptom is a field silently
// arriving empty: "bill" printed nothing and answered "which receipt is this?",
// on the one screen that had said exactly which.
//
// An absent body leaves the destination at its zero value, which is what
// "optional" means; anything else is refused as a malformed request.
func DecodeOptional(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return errBadJSON
	}
	return nil
}

func Decode(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		// The parser's message names byte offsets and Go's own types; the
		// caller learns nothing useful from it and a prober learns the stack.
		return errBadJSON
	}
	return validate.Struct(dst)
}
