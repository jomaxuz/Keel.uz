package escpos

import (
	"bytes"
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
