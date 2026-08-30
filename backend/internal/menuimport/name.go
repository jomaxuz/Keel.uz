package menuimport

import (
	"strings"
	"unicode"
)

// NormalName is what "the same dish" means when comparing an imported name
// against one already on the menu.
//
// ⚠️ **Exact comparison does not work here, and the reason is the apostrophe.**
// Uzbek writes `Lag'mon`, and every source spells that mark differently: the
// typographic ʻ (U+02BB) an aggregator's CMS produces, the ʼ a phone keyboard
// produces, a backtick, and the plain ASCII quote somebody typed. Four strings,
// one dish — and a check comparing them literally imports a second Lag'mon that
// reads as identical in the list and is a separate row in every report for ever
// afterwards.
//
// ⚠️ **Case and spacing too**, for the same reason: "LAG'MON" off a page
// written in capitals, and "Lag'mon " with the space a copy-paste left on it.
//
// ⚠️ **And no further.** It does not strip words, stem, or compare loosely:
// "Lag'mon" and "Lag'mon qovurma" are two dishes on every menu in the country,
// and a rule that merged them would silently refuse to import the second.
func NormalName(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	space := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch r {
		// Every mark that gets written where an Uzbek apostrophe belongs.
		case 'ʻ', 'ʼ', '‘', '’', '`', '´', '\'':
			b.WriteRune('\'')
			space = false
		default:
			if unicode.IsSpace(r) {
				// Collapsed rather than removed: "osh palov" and "oshpalov"
				// are not obviously the same thing, and guessing costs a dish.
				if !space && b.Len() > 0 {
					b.WriteRune(' ')
					space = true
				}
				continue
			}
			b.WriteRune(r)
			space = false
		}
	}
	return strings.TrimSpace(b.String())
}
