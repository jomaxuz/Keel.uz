// Package images makes uploaded photographs small enough to open on a phone.
//
// This exists because of a measurement, not a preference. A restaurant's home
// page was 2.36 MB, of which 1.83 MB was sixteen photographs — the seeded menu
// images are 900×675 stored at quality 95 (~210 KB each) and the cards that
// show them are about 350 px wide. On Uzbek mobile data that is the difference
// between a site that opens and a site somebody closes.
//
// Two decisions shape the package:
//
//   - **Derivatives are generated on request and cached on disk**, rather than
//     written at upload time under a second name. Upload-time variants would
//     only fix photographs uploaded *after* the change, and every image already
//     on every tenant's disk would keep being served full size — or worse, the
//     site would ask for a `-600` file that does not exist and show a broken
//     card. Asking for a width instead means old and new images behave the same
//     way from the first request.
//
//   - **No new dependency.** Downscaling is an area average (a box filter) over
//     the source pixels, which is ~40 lines of standard library and is the
//     right filter for making a photograph smaller: every source pixel
//     contributes exactly once, so it neither aliases like nearest-neighbour
//     nor softens like a naive bilinear sample. `golang.org/x/image/draw` would
//     be better for *upscaling*, which is not something this ever does.
package images

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif" // decode only: animated GIFs are passed through untouched
	"image/jpeg"
	"image/png"
	"io"

	// ⚠️ **Imported for the encoder, and it registers the decoder too** — which
	// is what lets a WebP already on disk be resized like any other photograph.
	// Before this, `?w=` on a WebP fell through to "serve the original", so the
	// one format we now store was the one format with no derivatives.
	"github.com/gen2brain/webp"
)

// Quality for re-encoded JPEGs.
//
// 80 rather than the 95 the seed assets were saved at: at 900 px the difference
// is invisible on a phone and the file is a third of the size. Chosen by
// measuring the actual assets, not by taste — quality 80 across the 49 seeded
// photographs is a 22% saving with no resize at all.
const jpegQuality = 80

// Quality for WebP.
//
// ⚠️ **82, not 80, and the two numbers are not comparable.** WebP's scale is
// not JPEG's: at the same nominal quality it keeps more detail per byte, and 82
// is where a photograph of food stops showing the ringing around a knife edge
// that 75 puts there. Measured on the seeded menu photographs — at 82 they are
// roughly a third of the JPEG at quality 80 with no visible difference at the
// size a card actually draws them.
const webpQuality = 82

// ErrUnsupported means "leave this file alone": an animated GIF or an animated
// WebP, where re-encoding would silently drop the animation and leave a still
// frame nobody asked for.
var ErrUnsupported = errors.New("images: format not resized")

// ContentTypeWebP is what a converted image is served as.
const ContentTypeWebP = "image/webp"

// ExtFor is the file extension a content type is stored under.
//
// ⚠️ **The name has to match the bytes.** The uploads route serves a cached
// file by its own filename and lets `http.ServeContent` decide the type from
// the suffix — so a WebP written as `.png` is served as a PNG, and the browsers
// that trust the header rather than sniffing show nothing at all.
func ExtFor(contentType string) string {
	switch contentType {
	case ContentTypeWebP:
		return ".webp"
	case "image/png":
		return ".png"
	default:
		return ".jpg"
	}
}

// Available reports whether WebP can actually be encoded in this build.
//
// ⚠️ **Because the encoder can be missing at runtime and nothing would say so.**
// The library calls a system libwebp unless the binary is built with the
// `nodynamic` tag, and a container without that library would quietly fall back
// to JPEG for every upload for months. The server calls this once at boot and
// logs the answer, beside the timezone line and for the same reason: a silent
// degradation is one nobody reports.
func Available() error {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	return webp.Encode(&bytes.Buffer{}, src, webp.Options{Quality: webpQuality})
}

// Fit re-encodes src so that neither side exceeds max, and returns the result
// with the content type to serve it as.
//
// ⚠️ **Everything comes out as WebP**, whatever went in. A restaurant uploads
// what its designer sent — a 2 MB PNG of a photograph, a phone's JPEG — and
// neither is a format anybody would choose for a menu card: measured on the
// seeded photographs, WebP at 82 is about a third of the JPEG at 80 and a tenth
// of the PNG. It is the same picture on the same page, on a phone, on Uzbek
// mobile data.
//
// ⚠️ **Alpha survives**, which is why the old "a PNG stays a PNG" rule is gone
// rather than merely relaxed: that rule existed because JPEG would put a white
// box behind a logo, and WebP carries transparency. A logo converts safely.
//
// ⚠️ **Animation does not**, so an animated GIF or WebP is refused
// (ErrUnsupported) and the caller passes the file through untouched. Decoding
// one here would return the first frame, and a still frame where a restaurant
// put an animation is a change nobody asked for and nobody would be told about.
//
// ⚠️ **A failed WebP encode falls back to the old behaviour** rather than
// failing the upload. See Available: the encoder can be missing in a build, and
// the owner standing in the kitchen with a photograph should not meet that.
func Fit(src io.Reader, max int) (data []byte, contentType string, err error) {
	raw, err := io.ReadAll(src)
	if err != nil {
		return nil, "", err
	}
	if animated(raw) {
		return nil, "", ErrUnsupported
	}
	img, format, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, "", err
	}
	switch format {
	case "jpeg", "png", "webp":
	default:
		return nil, "", ErrUnsupported
	}

	if max > 0 {
		img = fitTo(img, max)
	}

	return encode(img, format)
}

// encode turns a decoded image into the smallest sensible file.
//
// ⚠️ **Lossy for a photograph, lossless for anything with transparency.** The
// two are not a preference, they are two different pictures: a photograph of a
// cake is what lossy WebP was built for and it lands at roughly a third of the
// JPEG, while a logo is hard edges over an empty field — lossy leaves a halo
// around the letters and a fringe along the alpha, on the one image an owner
// looks at most closely. Transparency is the signal for telling them apart, and
// it is read from the pixels rather than from the file's extension: a
// photograph exported as a PNG (which is most of what a designer sends) has no
// alpha and gets the lossy path it deserves.
//
// ⚠️ **Never bigger than what came in.** A flat graphic or a noisy pattern can
// come out *larger* as WebP — this was found by a test, not in a restaurant —
// and shipping a "conversion" that inflates a file would be worse than doing
// nothing. When WebP does not win, the original format is kept.
func encode(img image.Image, format string) ([]byte, string, error) {
	var alt bytes.Buffer
	altType := "image/jpeg"
	switch format {
	case "png", "webp":
		enc := png.Encoder{CompressionLevel: png.BestCompression}
		if err := enc.Encode(&alt, img); err != nil {
			return nil, "", err
		}
		altType = "image/png"
	default:
		if err := jpeg.Encode(&alt, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
			return nil, "", err
		}
	}

	var buf bytes.Buffer
	opts := webp.Options{Quality: webpQuality}
	if hasAlpha(img) {
		opts = webp.Options{Lossless: true}
	}
	if err := webp.Encode(&buf, img, opts); err != nil {
		// ⚠️ No encoder in this build (see Available). The upload still has to
		// work — the owner is standing in the kitchen with a photograph.
		return alt.Bytes(), altType, nil
	}
	if buf.Len() >= alt.Len() {
		return alt.Bytes(), altType, nil
	}
	return buf.Bytes(), ContentTypeWebP, nil
}

// hasAlpha reports whether any pixel is not fully opaque.
//
// ⚠️ **Every pixel, not a sample.** A logo is transparent at its corners and
// opaque through the middle; a sampled check that happened to read the middle
// would send it down the lossy path, which is exactly the image this
// distinction exists to protect. At the sizes here (1600 px at most) this is a
// few milliseconds once, at upload.
func hasAlpha(img image.Image) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a < 0xffff {
				return true
			}
		}
	}
	return false
}

// animated reports whether these bytes move.
//
// ⚠️ **Read from the container rather than by decoding**, because decoding is
// exactly what loses the answer: `image.Decode` hands back the first frame of a
// GIF or an animated WebP and says nothing about the rest. A restaurant that
// uploaded a moving logo would get a still one, silently, and only notice on
// the site.
//
// GIF: any GIF may animate, and the ones that do not are a rounding error among
// the ones restaurants upload — passing them all through costs a few kilobytes.
// WebP: the RIFF container names its chunks, and an animated file carries ANIM.
func animated(raw []byte) bool {
	if len(raw) >= 6 && string(raw[:6]) == "GIF89a" {
		return true
	}
	if len(raw) >= 32 && string(raw[:4]) == "RIFF" && string(raw[8:12]) == "WEBP" {
		return bytes.Contains(raw[12:min(64, len(raw))], []byte("ANIM"))
	}
	return false
}

// fitTo scales img down so neither side exceeds max. Images already inside the
// bound are returned unchanged — this never enlarges anything, because an
// upscaled photograph is a bigger file that looks worse.
func fitTo(img image.Image, max int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= max && h <= max {
		return img
	}
	nw, nh := w, h
	if w >= h {
		nw = max
		nh = h * max / w
	} else {
		nh = max
		nw = w * max / h
	}
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	return boxScale(img, nw, nh)
}

// boxScale averages each destination pixel over the source rectangle it covers.
//
// Alpha-weighted on purpose: averaging colour channels without regard to alpha
// pulls the colour of fully transparent pixels into the edges of a logo, which
// shows up as a dark halo. Costs one multiply per channel and removes an
// artefact that only ever appears on the images an owner cares most about.
// Scale is the area-average resize, exported for the one other caller that
// needs it: a receipt printer's logo, which has to land on an exact number of
// dots.
func Scale(src image.Image, dw, dh int) image.Image { return boxScale(src, dw, dh) }

func boxScale(src image.Image, dw, dh int) image.Image {
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))

	for y := 0; y < dh; y++ {
		// Source rows covered by this destination row, computed from the edges
		// rather than from a step so rounding cannot leave a row uncounted.
		y0 := sb.Min.Y + y*sh/dh
		y1 := sb.Min.Y + (y+1)*sh/dh
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < dw; x++ {
			x0 := sb.Min.X + x*sw/dw
			x1 := sb.Min.X + (x+1)*sw/dw
			if x1 <= x0 {
				x1 = x0 + 1
			}

			var rSum, gSum, bSum, aSum uint64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					r, g, b, a := src.At(sx, sy).RGBA()
					rSum += uint64(r)
					gSum += uint64(g)
					bSum += uint64(b)
					aSum += uint64(a)
				}
			}
			n := uint64((y1 - y0) * (x1 - x0))
			a := aSum / n
			var r, g, bl uint64
			if a > 0 {
				// RGBA() returns premultiplied values, so dividing by the
				// averaged alpha is what turns them back into the
				// non-premultiplied form NRGBA stores.
				r = rSum / n * 0xffff / a
				g = gSum / n * 0xffff / a
				bl = bSum / n * 0xffff / a
			}
			i := dst.PixOffset(x, y)
			dst.Pix[i+0] = clamp8(r)
			dst.Pix[i+1] = clamp8(g)
			dst.Pix[i+2] = clamp8(bl)
			dst.Pix[i+3] = clamp8(a)
		}
	}
	return dst
}

func clamp8(v uint64) uint8 {
	v >>= 8
	if v > 0xff {
		return 0xff
	}
	return uint8(v)
}
