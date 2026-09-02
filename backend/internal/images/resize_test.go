package images

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// A photograph-ish source: smooth gradients with soft shading, which is what a
// camera actually produces.
//
// ⚠️ **Smooth on purpose.** An earlier version of this helper was
// high-frequency noise, and it made a real point by accident: lossy WebP stores
// noise *worse* than PNG stores it, so the "conversion" inflated the file. That
// is why Fit keeps whichever encoding is smaller — but it is not what a
// photograph of a cake looks like, and a test whose input is noise measures the
// wrong thing.
func photo(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{
				R: uint8(180 - x*120/w),
				G: uint8(60 + (x*90/w+y*60/h)/2),
				B: uint8(40 + y*140/h),
				A: 255,
			})
		}
	}
	return img
}

func jpegBytes(t *testing.T, img image.Image, q int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func pngBytes(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// ⚠️ **A real photograph, not a synthetic one**, and the difference is the
// whole point of this test. Every image a restaurant uploads is a photograph of
// food, and lossy WebP is built for exactly that: measured here against the
// seeded menu photographs, which are the same pictures the product ships with.
//
// A generated gradient would prove nothing — it compresses in PNG better than
// any lossy codec can manage, which is true and irrelevant to a menu card.
func realPhotograph(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "seed", "assets", "brauni.jpg"))
	if err != nil {
		t.Skipf("no seeded photograph to measure against: %v", err)
	}
	return raw
}

// Whatever a restaurant uploads, what lands on disk is smaller than what they
// sent — and for a photograph it is WebP.
func TestAPhotographBecomesWebP(t *testing.T) {
	if err := Available(); err != nil {
		t.Skipf("no webp encoder in this build: %v", err)
	}
	jpg := realPhotograph(t)

	// The same photograph as a PNG, which is what a designer's export looks
	// like — and the case that pays for this whole change: it arrives as
	// megabytes and leaves as a fraction of them.
	decoded, _, err := image.Decode(bytes.NewReader(jpg))
	if err != nil {
		t.Fatal(err)
	}
	asPNG := pngBytes(t, decoded)

	for _, tc := range []struct {
		name string
		in   []byte
	}{
		{"jpeg", jpg},
		{"png", asPNG},
	} {
		out, ct, err := Fit(bytes.NewReader(tc.in), 1600)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if ct != ContentTypeWebP {
			t.Errorf("%s came out as %q, want WebP", tc.name, ct)
		}
		if len(out) >= len(tc.in) {
			t.Errorf("%s got bigger: %d → %d bytes", tc.name, len(tc.in), len(out))
		}
		t.Logf("%s: %d → %d bytes (%d%%)", tc.name, len(tc.in), len(out),
			len(out)*100/len(tc.in))

		back, format, err := image.Decode(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("%s: the result does not decode: %v", tc.name, err)
		}
		if format != "webp" {
			t.Errorf("%s: decoded as %q", tc.name, format)
		}
		if back.Bounds() != decoded.Bounds() {
			t.Errorf("%s: size changed to %v", tc.name, back.Bounds())
		}
	}
}

// ⚠️ **Never bigger than what came in.** A flat graphic or a noisy pattern can
// come out *larger* as WebP — found here rather than in a restaurant — and a
// "conversion" that inflates a file would be worse than doing nothing at all.
func TestAConversionNeverInflatesAFile(t *testing.T) {
	if err := Available(); err != nil {
		t.Skipf("no webp encoder in this build: %v", err)
	}
	// High-frequency noise: the worst case for a lossy codec and the best case
	// for PNG's filters.
	noise := image.NewNRGBA(image.Rect(0, 0, 400, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 400; x++ {
			noise.Set(x, y, color.NRGBA{
				R: uint8((x*x + y*7) % 256),
				G: uint8((y*y + x*13) % 256),
				B: uint8((x * y) % 256),
				A: 255,
			})
		}
	}
	in := pngBytes(t, noise)
	out, ct, err := Fit(bytes.NewReader(in), 1600)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) > len(in) {
		t.Errorf("the file grew: %d → %d bytes (%s)", len(in), len(out), ct)
	}
}

// ⚠️ **The extension has to match the bytes**, because the uploads route serves
// a file by its own name and lets `http.ServeContent` read the type off the
// suffix. A WebP stored as `.png` is served as a PNG, and a browser that
// believes the header shows nothing at all.
func TestExtensionFollowsTheBytes(t *testing.T) {
	cases := map[string]string{
		ContentTypeWebP: ".webp",
		"image/png":     ".png",
		"image/jpeg":    ".jpg",
		"":              ".jpg",
	}
	for ct, want := range cases {
		if got := ExtFor(ct); got != want {
			t.Errorf("ExtFor(%q) = %q, want %q", ct, got, want)
		}
	}
}

// ⚠️ **An animation is left alone**, and this is checked on the container
// rather than after decoding — decoding is exactly what loses the answer.
// `image.Decode` hands back the first frame and says nothing about the rest, so
// a restaurant that uploaded a moving logo would get a still one, silently, and
// find out from the site.
func TestAnimationIsNotFlattened(t *testing.T) {
	var buf bytes.Buffer
	small := image.NewPaletted(image.Rect(0, 0, 4, 4), []color.Color{
		color.Black, color.White,
	})
	if err := gif.EncodeAll(&buf, &gif.GIF{
		Image: []*image.Paletted{small, small},
		Delay: []int{10, 10},
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Fit(bytes.NewReader(buf.Bytes()), 1600); err != ErrUnsupported {
		t.Fatalf("an animated GIF was re-encoded: %v", err)
	}

	// An animated WebP is the same promise, read out of the RIFF chunks.
	anim := append([]byte("RIFF\x00\x00\x00\x00WEBPVP8X"), make([]byte, 20)...)
	copy(anim[20:], "ANIM")
	if !animated(anim) {
		t.Fatal("an animated WebP would be flattened to its first frame")
	}
	if animated(jpegBytes(t, photo(8, 8), 80)) {
		t.Fatal("a plain photograph was read as an animation")
	}
}

// ⚠️ **Transparency survives, which is why "a PNG stays a PNG" could go.** That
// rule existed because JPEG would put a white box behind a logo; WebP carries
// alpha, so a logo converts safely — and a logo is the image an owner notices
// fastest.
func TestALogoKeepsItsTransparency(t *testing.T) {
	if err := Available(); err != nil {
		t.Skipf("no webp encoder in this build: %v", err)
	}
	src := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			// A red disc on a fully transparent field.
			if (x-32)*(x-32)+(y-32)*(y-32) < 20*20 {
				src.Set(x, y, color.NRGBA{R: 220, A: 255})
			}
		}
	}
	out, ct, err := Fit(bytes.NewReader(pngBytes(t, src)), 1600)
	if err != nil || ct != ContentTypeWebP {
		t.Fatalf("logo: %v (%s)", err, ct)
	}
	back, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, a := back.At(2, 2).RGBA(); a > 0x2000 {
		t.Errorf("the corner is no longer transparent (alpha %d) — a logo would "+
			"come back with a box behind it", a)
	}
}

// A WebP already on disk can now be resized like anything else. ⚠️ Before this
// it could not: `?w=` on a WebP fell through to "serve the original", so the one
// format we now store was the one format with no derivatives.
func TestAWebPCanBeResized(t *testing.T) {
	if err := Available(); err != nil {
		t.Skipf("no webp encoder in this build: %v", err)
	}
	full, _, err := Fit(bytes.NewReader(jpegBytes(t, photo(1200, 900), 90)), 0)
	if err != nil {
		t.Fatal(err)
	}
	small, ct, err := Fit(bytes.NewReader(full), 300)
	if err != nil {
		t.Fatalf("a stored WebP could not be resized: %v", err)
	}
	if ct != ContentTypeWebP {
		t.Errorf("resized WebP came out as %q", ct)
	}
	img, _, err := image.Decode(bytes.NewReader(small))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 300 {
		t.Errorf("width = %d, want 300", img.Bounds().Dx())
	}
}

// ⚠️ **An already-optimised file is left as it is**, and this was found on a
// real menu rather than in a test: twenty product cards went in at 2.1 MB and
// came out at 2.3 MB, because the comparison only ever looked at our own two
// encodings and never at the file that arrived. "Convert everything" was never
// the promise — smaller was.
func TestAnAlreadySmallFileIsLeftAlone(t *testing.T) {
	if err := Available(); err != nil {
		t.Skipf("no webp encoder in this build: %v", err)
	}
	// A photograph squeezed hard: nothing we do can beat it.
	tiny := jpegBytes(t, photo(600, 400), 35)

	out, ct, err := Fit(bytes.NewReader(tiny), 1600)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) > len(tiny) {
		t.Errorf("the file grew: %d → %d bytes (%s)", len(tiny), len(out), ct)
	}

	// ⚠️ But a picture that is too big for a phone is still resized, whatever
	// that costs in bytes: the bound is the whole point of this package, and an
	// image kept at 4000 px because the file happened to be small is an image
	// nobody on mobile data can open.
	huge := jpegBytes(t, photo(3000, 2000), 92)
	out, _, err = Fit(bytes.NewReader(huge), 1600)
	if err != nil {
		t.Fatal(err)
	}
	img, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 1600 {
		t.Errorf("width = %d, want 1600", img.Bounds().Dx())
	}
}
