// Package marking is what a scanned Asl Belgisi code is allowed to be.
//
// ⚠️ **A wrong code is refused by the tax register, in front of a guest.** The
// receipt is filed at the moment the money is taken; a code the register will
// not accept turns a payment into an argument at the counter, and the cashier's
// only remedy is to scan again — which is the right remedy and needs to happen
// *before* the sale, not after the refusal. So the shape is checked here, on
// the way in, and the same rules run on the till (`lib/marking.ts`) so the
// refusal lands where somebody is holding the bottle.
//
// ⚠️ **The shape only.** Whether the code has actually been issued, and whether
// it is still in circulation, is the national system's answer and it is given
// through the fiscal receipt itself — there is no separate Asl Belgisi API in
// this pipeline (docs/markirovka.md). Checking the shape catches the mistakes
// that happen at a counter: a scanner set to the wrong symbology, a barcode
// read instead of a DataMatrix, a half-read code, the same bottle scanned
// twice.
package marking

import (
	"errors"
	"strings"
)

// ⚠️ **GS1 element separators, and there are two of them in the wild.** A
// DataMatrix carries its fields separated by ASCII 29 (GS); many scanners in
// HID mode emit it as-is, and some replace it with the printable sequence
// "\x1d" or with the ASCII 232 substitute their configuration was set to. All
// three mean the same boundary, and a code rejected because the scanner's
// firmware chose a different one is a code the cashier cannot fix.
const (
	gs        = "\x1d"
	gsAlt     = "è"
	gsPrinted = "\\x1d"
)

var (
	// ErrEmpty is a line that should carry a code and does not.
	ErrEmpty = errors.New("markirovka kodi yo'q")
	// ErrShape is a string that cannot be a marking code.
	ErrShape = errors.New("markirovka kodi noto'g'ri")
	// ErrDuplicate is the same code twice on one receipt.
	ErrDuplicate = errors.New("bu kod chekda allaqachon bor")
)

// ⚠️ **The shortest real code is longer than a barcode and shorter than a URL,
// and those are the two things that get scanned by mistake.** A GS1 DataMatrix
// for a drink carries at least a GTIN (14 digits behind "01") and a serial
// (behind "21"), which is 31 characters before any separator. The upper bound
// is the standard's own limit for the data area.
const (
	minLen = 20
	maxLen = 200
)

// Normalize turns what the scanner typed into what is stored.
//
// ⚠️ **A scanner is a keyboard**, so what arrives is whatever the pistol typed
// into the focused field: the code, sometimes wrapped in whitespace, ending in
// the Enter it sends. Nothing here tries to be clever about the contents — the
// separators are unified and the edges are trimmed, because those are the two
// differences that come from the hardware rather than from the bottle.
func Normalize(raw string) string {
	s := strings.ReplaceAll(raw, gsPrinted, gs)
	s = strings.ReplaceAll(s, gsAlt, gs)
	s = strings.TrimSpace(s)
	// ⚠️ A trailing separator is what a scanner adds before the Enter, and it
	// carries no field after it. Kept, it makes two scans of one bottle look
	// like two different codes.
	return strings.TrimRight(s, gs)
}

// Check says whether a normalized code may be filed.
func Check(code string) error {
	if code == "" {
		return ErrEmpty
	}
	if len(code) < minLen || len(code) > maxLen {
		return ErrShape
	}
	// ⚠️ **"01" is the GS1 application identifier for a GTIN**, and every drink
	// code in this system starts with it. This is the check that separates a
	// DataMatrix from the EAN-13 printed beside it — the two look alike on a
	// bottle and a scanner will read whichever it is pointed at, so without
	// this the cashier's mistake reaches the tax register instead of the
	// screen.
	if !strings.HasPrefix(code, "01") {
		return ErrShape
	}
	// A code is printable ASCII plus the separator; anything else is a scanner
	// in the wrong mode or a keyboard layout translating as it types.
	for _, r := range code {
		if r == '\x1d' {
			continue
		}
		if r < 0x20 || r > 0x7e {
			return ErrShape
		}
	}
	return nil
}

// Line is what a caller needs to say about one line for CheckAll.
type Line struct {
	// Whether the menu item behind this line carries a code.
	Marked bool
	// What was scanned, already normalized.
	Code string
	// How many of this line. ⚠️ Read only to refuse: one code cannot cover two
	// bottles, and a quantity of two with one code files one and hands over two.
	Qty int
}

// CheckAll validates every line of one receipt together.
//
// ⚠️ **Together, because duplicates are the failure a per-line check cannot
// see.** The way a code gets scanned twice is entirely ordinary: the pistol
// beeps, nobody is sure it took, it is scanned again — and two lines then
// withdraw one bottle from circulation while the second bottle stays in it. The
// register accepts that receipt, so the only place it can be caught is here.
func CheckAll(lines []Line) error {
	seen := make(map[string]bool, len(lines))
	for _, l := range lines {
		if !l.Marked {
			// ⚠️ A code on an unmarked line is dropped by the caller rather
			// than refused here: it is somebody scanning at the wrong moment,
			// and stopping a sale over it helps nobody.
			continue
		}
		if err := Check(l.Code); err != nil {
			return err
		}
		if l.Qty > 1 {
			return ErrShape
		}
		if seen[l.Code] {
			return ErrDuplicate
		}
		seen[l.Code] = true
	}
	return nil
}
