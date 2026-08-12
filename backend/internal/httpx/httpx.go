package httpx

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
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

// Error writes a JSON error envelope.
func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}

// Decode parses the JSON request body into dst and validates it.
func Decode(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		// The parser's message names byte offsets and Go's own types; the
		// caller learns nothing useful from it and a prober learns the stack.
		return errBadJSON
	}
	return validate.Struct(dst)
}
