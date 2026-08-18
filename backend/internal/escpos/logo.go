package escpos

import (
	"bytes"
	"image"

	"restaurant-backend/internal/images"
)

// The restaurant's logo, on the paper.
//
// ⚠️ **Sent with the job, not stored in the printer.** ESC/POS has an "NV logo"
// kept in the printer's own flash, and it is the wrong choice here: it is
// uploaded with a vendor tool, per printer, per model, by somebody standing in
// front of it — and a restaurant that changes its logo would have to find that
// tool again. A raster image costs a few kilobytes per receipt and works the
// same on every unit.
//
// ⚠️ **One bit per dot, and that is the whole problem with logos.** A thermal
// head burns a dot or it does not; there is no grey. Flat artwork — a wordmark,
// a line drawing — comes out clean, and a photograph comes out as a smudge.
// Dithering is what makes the difference bearable, and it is why this does not
// simply threshold.

// Dots across the head, by paper width. ⚠️ These are the physical resolutions
// of 58 mm and 80 mm heads at 203 dpi and they are not a preference: an image
// wider than the head is silently cropped by the printer.
const (
	Dots58 = 384
	Dots80 = 576
)

// DotsFor is how many dots wide a logo may be on this paper.
func DotsFor(widthMM int) int {
	if widthMM == 58 {
		return Dots58
	}
	return Dots80
}

// Logo turns an image into the raster command for one receipt.
//
// The image is scaled to `width` dots (never up: a 120-pixel logo blown up to
// 576 is a blurred one), centred by padding, and dithered to black and white.
func Logo(src image.Image, width int) []byte {
	if src == nil || width <= 0 {
		return nil
	}
	b := src.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 {
		return nil
	}
	// ⚠️ Rounded down to a whole byte: the raster command counts bytes across,
	// and a width that is not a multiple of eight shifts every row after the
	// first — which prints as a diagonal smear, not as a slightly narrow logo.
	w := width / 8 * 8
	if b.Dx() < w {
		w = b.Dx() / 8 * 8
	}
	if w == 0 {
		return nil
	}
	h := b.Dy() * w / b.Dx()
	if h <= 0 {
		return nil
	}
	// ⚠️ Capped: a tall logo is a metre of paper per receipt, and the person who
	// uploaded a poster never sees the roll it costs.
	if h > 480 {
		h = 480
	}
	small := images.Scale(src, w, h)
	bits := dither(small, w, h)

	var out bytes.Buffer
	out.Write(alignCenter)
	// GS v 0 — print raster bit image. Mode 0: normal size.
	bytesPerRow := w / 8
	out.Write([]byte{0x1D, 0x76, 0x30, 0x00,
		byte(bytesPerRow % 256), byte(bytesPerRow / 256),
		byte(h % 256), byte(h / 256)})
	out.Write(bits)
	out.Write(alignLeft)
	out.Write(lineFeed)
	return out.Bytes()
}

// dither turns the scaled image into one bit per dot.
//
// ⚠️ **Floyd–Steinberg, carrying the error forward.** Plain thresholding turns
// every soft edge into a hard staircase and every mid-grey into either a solid
// block or nothing — which for a logo with a gradient means it disappears. The
// error diffusion is twelve lines and is the difference between a mark somebody
// recognises and a stain.
func dither(img image.Image, w, h int) []byte {
	// Luminance, 0..255, one float row of error carried into the next.
	grey := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			// ⚠️ Transparent pixels are **white**, not black. A logo saved as a
			// PNG with a transparent background would otherwise print as a
			// solid rectangle with the mark knocked out of it.
			if a < 0x8000 {
				grey[y*w+x] = 255
				continue
			}
			grey[y*w+x] = 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
		}
	}

	out := make([]byte, w/8*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			old := grey[y*w+x]
			var newVal float64
			if old < 128 {
				// Dark: burn the dot. The bit is 1 for black — the opposite of
				// how the pixel reads.
				out[y*(w/8)+x/8] |= 0x80 >> uint(x%8)
			} else {
				newVal = 255
			}
			err := old - newVal
			spread := func(dx, dy int, f float64) {
				nx, ny := x+dx, y+dy
				if nx < 0 || nx >= w || ny >= h {
					return
				}
				grey[ny*w+nx] += err * f
			}
			spread(1, 0, 7.0/16)
			spread(-1, 1, 3.0/16)
			spread(0, 1, 5.0/16)
			spread(1, 1, 1.0/16)
		}
	}
	return out
}
