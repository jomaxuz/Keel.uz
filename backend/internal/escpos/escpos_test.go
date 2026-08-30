package escpos

import (
	"bytes"
	"image"
	"image/color"
	"strings"
	"testing"
)

// ⚠️ **The failure this file exists for is invisible from the server.** The
// bytes leave, the printer prints *something*, and a restaurant discovers a
// page of question marks — or worse, a price with the wrong digits in it.
func TestUzbekLatinSurvivesTheCodePage(t *testing.T) {
	// The receipt designer's own default text contains these: correctly written
	// o‘ and g‘ use characters that exist in no printer code page.
	got := string(Latin.encode("Lag‘mon oʻzbekcha — 45 000 so'm"))

	if strings.ContainsRune(got, '?') {
		t.Fatalf("something was dropped as unprintable: %q", got)
	}
	if !strings.Contains(got, "Lag'mon o'zbekcha - 45 000 so'm") {
		t.Fatalf("folded wrongly: %q", got)
	}
}

// ⚠️ formatPrice groups thousands with a **non-breaking** space, and that
// character is not in CP437 either — so every total on every receipt would have
// printed with a "?" in the middle of the number.
func TestThousandsSeparatorPrints(t *testing.T) {
	got := string(Latin.encode("JAMI   92 000"))
	if strings.ContainsRune(got, '?') {
		t.Fatalf("the price separator did not survive: %q", got)
	}
	if !strings.Contains(got, "92 000") {
		t.Fatalf("got %q", got)
	}
}

func TestCyrillicUsesTheRussianPage(t *testing.T) {
	// А is 0x80 on CP866, я is 0xEF, ё is 0xF1 — the three edges of the map.
	got := Cyrillic.encode("Аяё")
	want := []byte{0x80, 0xEF, 0xF1}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
	if !bytes.Contains(Cyrillic.selectCmd(), []byte{0x1B, 0x74, 17}) {
		t.Fatal("the printer was never told to switch pages")
	}
}

// ⚠️ Order matters on paper: the drawer is kicked before the cut. On a slow
// printer the cut is the last thing to happen, and a drawer that opens after
// the receipt is torn off is one the cashier has already pulled at.
func TestDrawerIsKickedBeforeTheCut(t *testing.T) {
	out := Encode([]string{"x"}, Options{Cut: true, OpenDrawer: true})
	drawer := bytes.Index(out, Drawer)
	cut := bytes.Index(out, cutPartial)
	if drawer < 0 || cut < 0 {
		t.Fatalf("drawer=%d cut=%d — one of them was not written", drawer, cut)
	}
	if drawer > cut {
		t.Fatal("the drawer opens after the paper is cut")
	}
}

// ⚠️ A printer with no cutter **prints** the cut command rather than ignoring
// it, so it has to be possible to leave it out.
func TestNoCutMeansNoCutBytes(t *testing.T) {
	out := Encode([]string{"x"}, Options{})
	if bytes.Contains(out, cutPartial) || bytes.Contains(out, cutFull) {
		t.Fatal("a cut was sent to a printer that was not asked to cut")
	}
	if bytes.Contains(out, Drawer) {
		t.Fatal("the drawer was kicked without being asked")
	}
	if !bytes.HasPrefix(out, initPrinter) {
		t.Fatal("the job does not start by resetting the printer")
	}
}

// The QR is the one thing the text cannot carry — the guest's right to check
// that the sale was registered.
func TestQRCarriesItsPayloadAndItsLength(t *testing.T) {
	out := EncodeQR("https://ofd.soliq.uz/check?t=UZ21")
	if !bytes.Contains(out, []byte("https://ofd.soliq.uz/check?t=UZ21")) {
		t.Fatal("the payload is missing")
	}
	n := len("https://ofd.soliq.uz/check?t=UZ21") + 3
	if !bytes.Contains(out, []byte{byte(n % 256), byte(n / 256), 0x31, 0x50, 0x30}) {
		t.Fatalf("the stored length is wrong for a %d-byte payload", n)
	}
	if EncodeQR("") != nil {
		t.Fatal("an empty payload still sent a QR command")
	}
}

// The restaurant's logo, as dots.
func TestLogoIsWholeBytesWideAndCentred(t *testing.T) {
	// A picture wider than the head, so it has to be scaled down.
	src := image.NewRGBA(image.Rect(0, 0, 1000, 250))
	for y := 0; y < 250; y++ {
		for x := 0; x < 1000; x++ {
			// A dark half and a light half: something must burn, something must
			// not, or the test proves nothing about the dithering.
			if x < 500 {
				src.Set(x, y, color.RGBA{0, 0, 0, 255})
			} else {
				src.Set(x, y, color.RGBA{255, 255, 255, 255})
			}
		}
	}
	out := Logo(src, Dots80)
	if len(out) == 0 {
		t.Fatal("nothing was produced for a perfectly ordinary picture")
	}
	// GS v 0 — raster bit image.
	i := bytes.Index(out, []byte{0x1D, 0x76, 0x30, 0x00})
	if i < 0 {
		t.Fatalf("the raster command is missing: % X", out[:16])
	}
	// ⚠️ The width is counted in **bytes**, and a width that is not a multiple
	// of eight dots shifts every row after the first — which prints as a
	// diagonal smear rather than as a slightly narrow logo.
	widthBytes := int(out[i+4]) + int(out[i+5])*256
	if widthBytes*8 > Dots80 {
		t.Fatalf("the logo is %d dots wide on a %d dot head", widthBytes*8, Dots80)
	}
	height := int(out[i+6]) + int(out[i+7])*256
	if height == 0 {
		t.Fatal("the logo has no height")
	}
	if len(out) < i+8+widthBytes*height {
		t.Fatal("fewer bytes than the header promises — the printer would hang")
	}
	if !bytes.Contains(out[:i], alignCenter) {
		t.Fatal("the logo is not centred")
	}
}

// ⚠️ A PNG with a transparent background is the ordinary way a logo is saved,
// and reading "no colour" as black prints a solid rectangle with the mark
// knocked out of it.
func TestTransparentBackgroundPrintsWhite(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 64, 64)) // all zero: transparent
	out := Logo(src, Dots58)
	i := bytes.Index(out, []byte{0x1D, 0x76, 0x30, 0x00})
	if i < 0 {
		t.Fatal("no raster")
	}
	// ⚠️ Only the raster itself: the alignment reset and the feed after it are
	// commands, not dots, and reading past the image would fail every time.
	widthBytes := int(out[i+4]) + int(out[i+5])*256
	height := int(out[i+6]) + int(out[i+7])*256
	for _, b := range out[i+8 : i+8+widthBytes*height] {
		if b != 0 {
			t.Fatal("a transparent picture burned dots")
		}
	}
}

func TestNoLogoIsNoBytes(t *testing.T) {
	if Logo(nil, Dots80) != nil {
		t.Fatal("a missing picture produced a command")
	}
	if Logo(image.NewRGBA(image.Rect(0, 0, 4, 4)), 0) != nil {
		t.Fatal("a zero width produced a command")
	}
}

// ⚠️ **Emphasis must never outlive its line.** A printer left bold or in double
// height prints the whole rest of the roll that way, and the line that switched
// it on is long gone by the time anybody looks — so what is asserted here is
// not that the command was sent but that it was closed.
func TestEmphasisIsClosedOnTheLineThatOpenedIt(t *testing.T) {
	out := Encode([]string{MarkBoldBig + "JAMI", "oddiy qator"}, Options{})

	if !bytes.Contains(out, boldOn) || !bytes.Contains(out, sizeBig) {
		t.Fatal("the marked line was printed plain")
	}
	if !bytes.Contains(out, boldOff) || !bytes.Contains(out, sizeNorm) {
		t.Fatal("emphasis was never turned off — the rest of the roll inherits it")
	}
	// The marker itself must not reach the paper.
	if bytes.Contains(out, []byte{0x03}) {
		t.Fatal("the control character was printed as text")
	}
	if !bytes.Contains(out, []byte("JAMI")) {
		t.Fatal("the line lost its text")
	}
}

func TestAPlainLineIsUnchanged(t *testing.T) {
	// ⚠️ Every receipt printed before this existed is a plain line, and none of
	// them may gain a byte.
	plain := Encode([]string{"Osh"}, Options{})
	if bytes.Contains(plain, boldOn) || bytes.Contains(plain, sizeBig) {
		t.Fatal("an unmarked line came out emphasised")
	}
}

func TestTextThatMerelyLooksLikeAMarkerIsText(t *testing.T) {
	// A dish somebody typed with asterisks, and a line beginning with a
	// printable character, are both text.
	out := Encode([]string{"**Osh**"}, Options{})
	if !bytes.Contains(out, []byte("**Osh**")) {
		t.Fatal("a dish name was eaten by the marker check")
	}
}

// ⚠️ **The sign on every line of every receipt.** "2 × Osh" is how a quantity is
// printed, and U+00D7 exists in no thermal printer's code page — so the one
// character guaranteed to appear on every check came out as a question mark.
// Found on paper in a restaurant, which is where a character set is actually
// tested.
func TestTheMultiplicationSignSurvivesTheCodePage(t *testing.T) {
	for _, c := range []Charset{Latin, Cyrillic} {
		got := string(c.encode("2 × Osh"))
		if strings.Contains(got, "?") {
			t.Fatalf("%s: %q still has a hole in it", c, got)
		}
		if !strings.Contains(got, "x") {
			t.Fatalf("%s: %q lost the quantity marker", c, got)
		}
	}
}

// ⚠️ **A Russian receipt on a printer nobody re-ticked.** The charset is a
// per-printer setting and the receipt language is a per-kind one; nothing joins
// them, so the second printer in a restaurant prints a page of question marks
// with a perfectly correct layout. The text decides the page.
func TestCyrillicTextPromotesTheCodePage(t *testing.T) {
	out := Encode([]string{"Официант"}, Options{Charset: Latin})
	if !bytes.Contains(out, []byte{0x1B, 0x74, 17}) {
		t.Fatal("Cyrillic text did not select CP866")
	}
	if bytes.Contains(out, []byte("?")) {
		t.Fatal("Cyrillic text came out as question marks")
	}
}

// And a Latin job is left exactly where it was: promoting every receipt would
// move restaurants that never asked onto a page their printer may not hold.
func TestLatinTextStaysOnTheLatinPage(t *testing.T) {
	out := Encode([]string{"Ofitsiant"}, Options{Charset: Latin})
	if !bytes.Contains(out, []byte{0x1B, 0x74, 0}) {
		t.Fatal("a Latin job did not select CP437")
	}
}
