package main

import (
	"encoding/binary"
	"os"
	"testing"
)

// ⚠️ **The installer wore Wails' logo and the build said nothing.**
//
// The icon was written by an image library that compresses every entry as PNG.
// The ICO format allows that for 256x256 only; NSIS reads the smaller sizes as
// device-independent bitmaps, finds a PNG signature, and quietly keeps its own
// default icon. Wails' own loader accepts the file either way, so the
// executable looked correct while the installer did not — and the only symptom
// was a logo, on an artefact nobody rebuilds twice.
//
// This test reads the file the way NSIS does. It cannot run makensis, so it
// checks the one property that decides the outcome.
func TestIconEntriesAreBitmapsNotPNG(t *testing.T) {
	raw, err := os.ReadFile("build/windows/icon.ico")
	if err != nil {
		t.Fatalf("icon.ico: %v", err)
	}
	if len(raw) < 6 {
		t.Fatal("icon.ico is truncated")
	}
	if kind := binary.LittleEndian.Uint16(raw[2:4]); kind != 1 {
		t.Fatalf("type %d, want 1 (icon)", kind)
	}
	n := int(binary.LittleEndian.Uint16(raw[4:6]))
	if n == 0 {
		t.Fatal("icon.ico holds no images")
	}

	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	var sizes []int
	for i := range n {
		e := 6 + 16*i
		if e+16 > len(raw) {
			t.Fatalf("directory entry %d runs past the end of the file", i)
		}
		w := int(raw[e])
		if w == 0 {
			w = 256 // 0 means 256 in the ICO directory
		}
		off := int(binary.LittleEndian.Uint32(raw[e+12 : e+16]))
		if off+8 > len(raw) {
			t.Fatalf("%dx%d image points past the end of the file", w, w)
		}
		if string(raw[off:off+8]) == string(png) {
			t.Errorf("the %dx%d image is PNG-compressed — NSIS will ignore this "+
				"icon and ship the installer with its own", w, w)
		}
		sizes = append(sizes, w)
	}

	// ⚠️ 16 is the one that matters most and the easiest to leave out: it is the
	// taskbar and the title of every window, and it is where a downscaled 256
	// turns a thin stroke into grey mush.
	for _, want := range []int{16, 32, 48, 256} {
		found := false
		for _, s := range sizes {
			if s == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("icon.ico has no %dx%d image; sizes present: %v", want, want, sizes)
		}
	}
}
