// Package escpos turns a laid-out receipt into the bytes a thermal printer
// understands.
//
// ⚠️ **It formats nothing.** internal/receipt has already counted characters
// and wrapped the lines for the paper width; this only adds the control codes
// around them — initialise, choose a character set, feed, cut, kick the drawer.
// Two layout engines would drift, and the drift would be found by a guest
// holding a receipt that does not match the one the owner approved.
//
// ⚠️ **ESC/POS is a family, not a standard.** Xprinter, Rongta, Epson, Bixolon
// and the no-name units sold in Tashkent all speak the Epson command set for
// the things below — init, feed, cut, drawer, code page — and differ in the
// extras (logos, fonts, barcode variants). Only the common core is used here,
// deliberately: a command a printer does not know is not ignored, it is printed
// as garbage in the middle of a bill.
package escpos

import (
	"bytes"
	"strings"
	"unicode"
)

// The commands. Named rather than inlined because a stray byte in this file is
// a page of mojibake in a restaurant.
var (
	initPrinter = []byte{0x1B, 0x40}       // ESC @
	lineFeed    = []byte{0x0A}             // LF
	cutFull     = []byte{0x1D, 0x56, 0x00} // GS V 0
	cutPartial  = []byte{0x1D, 0x56, 0x01} // GS V 1
	alignLeft   = []byte{0x1B, 0x61, 0x00} // ESC a 0
	alignCenter = []byte{0x1B, 0x61, 0x01} // ESC a 1

	// ⚠️ **Emphasis and size are two different commands, and only one of them
	// is safe everywhere.** `ESC E` (bold) is honoured by every printer this
	// product has met. `GS !` (character size) is honoured by most, and a
	// printer that ignores it prints the line at normal size — which is a
	// receipt that still reads correctly. Neither is allowed to leave state
	// behind: every marked line is closed before the next one begins, because a
	// printer left in double height prints the rest of the roll that way.
	boldOn   = []byte{0x1B, 0x45, 0x01} // ESC E 1
	boldOff  = []byte{0x1B, 0x45, 0x00} // ESC E 0
	sizeBig  = []byte{0x1D, 0x21, 0x11} // GS ! — double width and height
	sizeNorm = []byte{0x1D, 0x21, 0x00} // GS ! 0
)

// Drawer opens the cash drawer wired into the printer.
//
// ⚠️ **Pin 2, 50 ms on, 200 ms off.** The drawer is a solenoid on a phone-style
// plug and the pulse is what throws the bolt; too short and it does not open,
// too long and the coil cooks. These are the values every printer's manual
// gives, and the reason they are not configurable is that a restaurant cannot
// debug a drawer that opens "sometimes".
var Drawer = []byte{0x1B, 0x70, 0x00, 0x19, 0xFA} // ESC p 0 25 250

// Options is how one job differs from another.
type Options struct {
	// The character set the paper will be printed in.
	Charset Charset
	// Feed this many blank lines before cutting, so the tear-off is below the
	// last line rather than through it.
	FeedLines int
	// Cut the paper. ⚠️ Off for printers without a cutter: the command is not
	// ignored by all of them — some print it.
	Cut bool
	// Full cut rather than partial. Partial leaves a tab holding the receipt on
	// the roll, which is what most restaurants want.
	FullCut bool
	// Kick the cash drawer open. Only ever true for the till's own copy.
	OpenDrawer bool
	// The restaurant's logo, already rastered by Logo(). Printed above
	// everything else, which is where a letterhead goes.
	Banner []byte
}

// Encode wraps rendered lines in the control codes for one job.
func Encode(lines []string, o Options) []byte {
	var b bytes.Buffer
	b.Write(initPrinter)
	b.Write(o.Charset.selectCmd())
	// ⚠️ After the reset, before the text: `ESC @` clears the print position and
	// the character set, so a logo written before it comes out with whatever the
	// last job left behind.
	b.Write(o.Banner)
	b.Write(alignLeft)

	for _, line := range lines {
		// ⚠️ **The marker is a control character at the head of the line, and
		// it is stripped here.** The renderer works in plain strings — the same
		// strings the browser prints when no printer of the restaurant's took
		// the job — so emphasis travels as a byte nothing else displays rather
		// than as a second data structure every caller would have to learn.
		text, bold, big := readMark(line)
		if bold {
			b.Write(boldOn)
		}
		if big {
			b.Write(sizeBig)
		}
		b.Write(o.Charset.encode(text))
		// ⚠️ Closed on the same line it was opened on, always. A printer left
		// emphasised prints everything after it that way, and the line that
		// turned it on is long past by the time anybody notices.
		if big {
			b.Write(sizeNorm)
		}
		if bold {
			b.Write(boldOff)
		}
		b.Write(lineFeed)
	}
	for i := 0; i < o.FeedLines; i++ {
		b.Write(lineFeed)
	}
	// ⚠️ The drawer is kicked **before** the cut, not after. On a slow printer
	// the cut is the last thing to happen and the cashier's hand is already
	// moving; a drawer that opens after the paper is torn is a drawer they have
	// already pulled at.
	if o.OpenDrawer {
		b.Write(Drawer)
	}
	if o.Cut {
		if o.FullCut {
			b.Write(cutFull)
		} else {
			b.Write(cutPartial)
		}
	}
	return b.Bytes()
}

// EncodeQR prints a QR code, centred, at the end of a receipt.
//
// ⚠️ **The one thing text cannot do.** The fiscal QR is the guest's right to
// check that the sale was registered, and a printer that cannot draw it turns a
// receipt into a piece of paper. Model 2, the size and correction level every
// unit in this price range supports.
func EncodeQR(text string) []byte {
	if text == "" {
		return nil
	}
	var b bytes.Buffer
	b.Write(alignCenter)
	// Model 2
	b.Write([]byte{0x1D, 0x28, 0x6B, 0x04, 0x00, 0x31, 0x41, 0x32, 0x00})
	// Module size 6 — big enough for a phone camera at arm's length on 58 mm.
	b.Write([]byte{0x1D, 0x28, 0x6B, 0x03, 0x00, 0x31, 0x43, 0x06})
	// Error correction M: survives a thermal roll that has been in a pocket.
	b.Write([]byte{0x1D, 0x28, 0x6B, 0x03, 0x00, 0x31, 0x45, 0x31})
	// Store the data. The length is the payload + 3, little-endian.
	n := len(text) + 3
	b.Write([]byte{0x1D, 0x28, 0x6B, byte(n % 256), byte(n / 256), 0x31, 0x50, 0x30})
	b.WriteString(text)
	// Print it.
	b.Write([]byte{0x1D, 0x28, 0x6B, 0x03, 0x00, 0x31, 0x51, 0x30})
	b.Write(alignLeft)
	b.Write(lineFeed)
	return b.Bytes()
}

// ---- Character sets ----

// Charset is which code page the printer is told to use.
//
// ⚠️ **This is where receipts actually go wrong**, and it is invisible from
// here: the bytes leave, the printer prints *something*, and the restaurant
// discovers a page of question marks. Thermal printers have no Unicode — they
// hold a handful of 256-character pages and the text has to be converted into
// the one that is selected.
type Charset string

const (
	// Latin. Uzbek written in Latin script, which is what the default receipt
	// text and most menus are.
	Latin Charset = "latin"
	// Cyrillic (CP866), for a restaurant whose menu and receipts are Russian.
	Cyrillic Charset = "cyrillic"
)

func (c Charset) selectCmd() []byte {
	// ESC t n — select character code table.
	if c == Cyrillic {
		return []byte{0x1B, 0x74, 17} // CP866
	}
	return []byte{0x1B, 0x74, 0} // CP437, the universal default
}

// encode converts one line into the selected page's bytes.
func (c Charset) encode(s string) []byte {
	s = Fold(s)
	if c == Cyrillic {
		return cp866(s)
	}
	return ascii(s)
}

// Markers a rendered line may begin with. They are control characters so that
// nothing in a restaurant's own text can be mistaken for one — a dish called
// "**Osh**" prints its asterisks.
const (
	// MarkBold asks for ESC E on this line.
	MarkBold = "\x01"
	// MarkBig asks for double width and height.
	MarkBig = "\x02"
	// MarkBoldBig asks for both.
	MarkBoldBig = "\x03"
)

// readMark takes the emphasis off the front of a line.
//
// ⚠️ Only the first character, and only these three. A line that begins with
// anything else is text, including text that begins with a control character
// somebody pasted from a spreadsheet.
func readMark(line string) (text string, bold, big bool) {
	if line == "" {
		return line, false, false
	}
	switch line[0] {
	case 0x01:
		return line[1:], true, false
	case 0x02:
		return line[1:], false, true
	case 0x03:
		return line[1:], true, true
	}
	return line, false, false
}

// Fold rewrites the characters a thermal printer has never heard of.
//
// ⚠️ **Uzbek Latin is the problem, not Cyrillic.** Correctly written o‘ and g‘
// use U+2018/U+02BB, which exists in no printer code page — and the receipt
// designer's own default text contains them. Unfolded they print as a random
// glyph or as nothing, in the middle of the restaurant's name. Folding to the
// ASCII apostrophe is what every Uzbek POS does, and it is legible.
//
// The dashes and ellipsis are here for the same reason: they arrive from menus
// typed in Word.
func Fold(s string) string {
	return folder.Replace(s)
}

var folder = strings.NewReplacer(
	"ʻ", "'", // U+02BB — the correct one
	"ʼ", "'", // U+02BC
	"‘", "'", // U+2018
	"’", "'", // U+2019
	"“", "\"", "”", "\"",
	"–", "-", "—", "-", // en/em dash
	"…", "...",
	"№", "N",
	" ", " ", // the non-breaking space formatPrice groups thousands with
)

// ascii drops anything the Latin page cannot show, rather than emitting a byte
// that means something else there.
//
// ⚠️ A dropped character is a gap; a wrong byte is a different letter, and on a
// price that is a different number.
func ascii(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		switch {
		case r < 0x80:
			out = append(out, byte(r))
		case unicode.IsSpace(r):
			out = append(out, ' ')
		default:
			// Unknown to this page. A question mark rather than silence: a
			// receipt with a hole in a dish name reads as a broken printer, and
			// a visible "?" tells the owner to switch the code page.
			out = append(out, '?')
		}
	}
	return out
}

// cp866 maps Cyrillic into the page Russian-speaking printers ship with.
func cp866(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		switch {
		case r < 0x80:
			out = append(out, byte(r))
		case r >= 'А' && r <= 'п': // А..п — contiguous at 0x80
			out = append(out, byte(r-'А'+0x80))
		case r >= 'р' && r <= 'я': // р..я — resumes at 0xE0
			out = append(out, byte(r-'р'+0xE0))
		case r == 'Ё':
			out = append(out, 0xF0)
		case r == 'ё':
			out = append(out, 0xF1)
		case unicode.IsSpace(r):
			out = append(out, ' ')
		default:
			out = append(out, '?')
		}
	}
	return out
}
