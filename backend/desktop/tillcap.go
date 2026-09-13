package main

// What the till says when a branch has bound every register its plan allows.
//
// ⚠️ **Its own file, with no build tag, so it can be argued with in a test.**
// The pairing code is Windows-only — it has to be — and a sentence a restaurant
// reads at the worst possible moment should not be beyond the reach of the test
// suite because of where it happens to be called from.

import (
	"encoding/json"
	"fmt"
)

// capMessage turns the server's refusal into the sentence shown on the setup
// screen when a branch has bound every register its plan allows.
//
// ⚠️ **It says what to do, because the person reading it can do both things.**
// A till that is no longer on the counter still holds its slot until somebody
// unbinds it in the panel, and that is the usual cause — far more often than a
// restaurant genuinely outgrowing its plan. "Limit reached" alone leaves one
// action, which is ringing us.
//
// ⚠️ The count comes from the body rather than being left out when it is
// missing: an older server answers 402 with nothing but a message, and the
// sentence has to read correctly then too.
func capMessage(raw []byte) string {
	var body struct {
		Error     string `json:"error"`
		Registers int    `json:"registers"`
		Limit     int    `json:"limit"`
	}
	_ = json.Unmarshal(raw, &body)
	msg := "kassa ekranlari limiti tugadi"
	if body.Limit > 0 {
		msg = fmt.Sprintf("%s (%d / %d)", msg, body.Registers, body.Limit)
	}
	return msg + " — panelda ishlatilmayotgan ekranni uzing yoki tarifni ko'taring"
}
