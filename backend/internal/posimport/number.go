// Package posimport reads a nomenclature, a set of tech cards or a stock sheet
// exported from another till system.
//
// # Why a file and not an API
//
// A restaurant running iiko, r_keeper, Clopos, Poster or Jowi does not stay
// because it likes the software. It stays because the ingredient list, the tech
// cards and the stock are in there, and moving them by hand is weeks of work
// somebody has to be paid for. That sentence — "hamma narsa boshqa POS'da" — is
// the actual objection, and it is a real one.
//
// The answer is a file rather than an integration, for three reasons and the
// third is the one that decides it:
//
//  1. Every one of those systems exports to Excel. Not one of them documents an
//     API for reading tech cards, and the ones that have one hand it over after
//     a contract — which the restaurant is in the middle of leaving.
//  2. A restaurant on its way out often no longer has API access, or never had
//     it: the licence is the reseller's.
//  3. **Guessing a wire format here is worse than useless.** An adapter written
//     against a documentation-free API compiles, reviews cleanly, and imports a
//     tech card with the quantities in the wrong unit — which is a food cost
//     that looks computed and is a thousand times wrong. See the fiscal
//     package's note on the same rule.
package posimport

import (
	"errors"
	"strconv"
	"strings"
)

// ParseQty reads a number out of a cell exported by a Russian-locale program.
//
// ⚠️ **The decimal comma, and it is the most expensive character in this whole
// import.** Every one of these systems exports from a Russian locale: a recipe
// line of 180 grams is written `0,180`, and `strconv.ParseFloat` on that fails.
// The tempting recovery — treat a failed parse as zero — imports the tech card
// with every quantity at nothing, which makes every dish cost nothing, which
// makes the food cost report say the kitchen is free. It is arithmetically
// consistent, appears on a screen full of correct-looking dishes, and is
// discovered at a stocktake.
//
// ⚠️ **And the thousands separator is a space**, sometimes a non-breaking one:
// `1 234,56`. Stripped, not treated as a delimiter — a cell that reads
// `1 234,56` is one number, and reading it as two is how a stock balance
// becomes 1.
//
// ⚠️ **A cell that cannot be read is an error, never a zero.** The caller shows
// it as a line to fix. Silence here is the difference between an import
// somebody corrects and an import somebody trusts.
func ParseQty(raw string) (float64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, errUnreadable
	}
	// Every space a spreadsheet or a locale can put between thousands.
	for _, sp := range []string{" ", " ", " ", " ", "'"} {
		s = strings.ReplaceAll(s, sp, "")
	}
	// ⚠️ Both separators present means the comma is the decimal and the dot is
	// the thousands mark, or the reverse — decided by which comes last, because
	// the decimal separator is always the rightmost one.
	comma, dot := strings.LastIndex(s, ","), strings.LastIndex(s, ".")
	switch {
	case comma >= 0 && dot >= 0:
		if comma > dot {
			s = strings.ReplaceAll(s, ".", "")
			s = strings.Replace(s, ",", ".", 1)
		} else {
			s = strings.ReplaceAll(s, ",", "")
		}
	case comma >= 0:
		// ⚠️ A lone comma is a decimal comma, not a thousands mark. `1,5` is one
		// and a half kilos on every export these systems produce; reading it as
		// fifteen would put ten times the meat in the dish.
		s = strings.Replace(s, ",", ".", 1)
	}
	// Trailing currency or unit text an export sometimes leaves in the cell.
	s = strings.TrimRight(s, " ")
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, errUnreadable
	}
	return n, nil
}

// ParseMoney reads a price and returns whole so'm.
//
// ⚠️ Rounded rather than truncated, and rounded here rather than by the caller:
// an export writes 42 000,50 and every ingredient in the list would otherwise
// lose its half so'm in the same direction. The amount is trivial; a rule that
// lives in one place and is applied once is not.
func ParseMoney(raw string) (int, error) {
	n, err := ParseQty(raw)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, errUnreadable
	}
	return int(n + 0.5), nil
}

var errUnreadable = errors.New("raqamni o'qib bo'lmadi")

// Unreadable reports whether a cell could not be read as a number.
func Unreadable(err error) bool { return errors.Is(err, errUnreadable) }
