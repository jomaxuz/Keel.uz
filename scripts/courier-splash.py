#!/usr/bin/env python3
"""A phone app's splash image — the mark, and the app's own word under it.

⚠️ **Generated, not drawn by hand, and that is the point.** The waiter app's
splash came out of an image tool and cannot be re-derived: the next person who
needs it a size larger, or in a second wordmark, starts from a PNG. This one is
a script — the mark is the same two paths as `logos/keel-mark.svg`, so a change
to the brand is a change in one place rather than a redraw.

⚠️ **The icon is deliberately not generated here.** The apps share it
(`assets/icon.png` and the Android adaptive set are copied from the waiter),
because on one phone they have to read as one product. Only the splash says
which of them you opened, which is exactly when that question is asked.

    python3 scripts/courier-splash.py mobile/courier/assets/splash-icon.png
    python3 scripts/courier-splash.py mobile/team/assets/splash-icon.png Team

Needs Pillow and Space Grotesk Bold (the brand's heading face; see
logos/README.txt). Point SPACE_GROTESK at the .ttf, or pass it as argv[2].
"""

import os
import sys

from PIL import Image, ImageDraw, ImageFont

# Sampled from the waiter app's splash so the pair matches exactly. ⚠️ Not the
# theme's accent (#e2590d): that is the restaurant template's orange, and this
# is Keel's own — the two live on different screens and are not the same colour.
NAVY = (0, 11, 28)
ORANGE = (254, 162, 4)
WHITE = (255, 255, 255)

SIZE = 1024
SS = 4  # supersampling; the mark is one thick stroke and edges show


def guess_word(path: str) -> str:
    """"mobile/team/assets/splash-icon.png" → "Team"."""
    parts = [p for p in path.split(os.sep) if p]
    if "mobile" in parts:
        i = parts.index("mobile")
        if i + 1 < len(parts):
            return parts[i + 1].capitalize()
    return "Keel"


def bezier(p0, p1, p2, p3, steps=160):
    for i in range(steps + 1):
        t = i / steps
        u = 1 - t
        yield (
            u * u * u * p0[0] + 3 * u * u * t * p1[0] + 3 * u * t * t * p2[0] + t * t * t * p3[0],
            u * u * u * p0[1] + 3 * u * u * t * p1[1] + 3 * u * t * t * p2[1] + t * t * t * p3[1],
        )


def stroke(draw, pts, width, colour):
    """A round-capped, round-joined polyline, stamped rather than drawn.

    ⚠️ **Pillow's `joint="curve"` is not a round join.** On a curve sampled
    this finely it leaves a spike at every vertex — the mark came out looking
    frayed, like a rope rather than a hull. Stamping a disc along the path has
    no joins to get wrong, and the caps come free: the ends of these two
    strokes are half the mark's character, and squared ones read as a bracket.
    """
    r = width / 2
    prev = None
    for x, y in pts:
        if prev is not None:
            # Fill the gap between stamps so a coarse sampling cannot show
            # through as scalloping.
            draw.line([prev, (x, y)], fill=colour, width=int(width))
        draw.ellipse([x - r, y - r, x + r, y + r], fill=colour)
        prev = (x, y)


def keel_mark(draw, cx, cy, size):
    """The mark from logos/keel-mark.svg, centred on (cx, cy).

    The source is a 32-unit viewBox whose ink runs from y=6 to y=29 and x=5 to
    x=27 — so it is placed by its own ink rather than by the box, which is why
    the numbers below are offsets rather than the viewBox's centre.
    """
    unit = size / 32
    ox = cx - 16 * unit
    oy = cy - 17.5 * unit  # (6 + 29) / 2

    def P(x, y):
        return (ox + x * unit, oy + y * unit)

    hull = [P(*p) for p in bezier((5, 6), (5, 15.5), (9.4, 20.5), (16, 20.5))]
    # The `S` command: the first control point mirrors the previous one.
    hull += [P(*p) for p in bezier((16, 20.5), (22.6, 20.5), (27, 15.5), (27, 6))]
    width = 2.6 * unit

    stroke(draw, hull, width, ORANGE)
    stroke(draw, [P(16, 20.5), P(16, 29)], width, ORANGE)


def main() -> None:
    out = sys.argv[1] if len(sys.argv) > 1 else "mobile/courier/assets/splash-icon.png"
    # The word under the mark. ⚠️ Taken from the path when it is not given, so
    # a new app's splash is one argument rather than an edit to this file —
    # which is how the second one would end up saying the first one's name.
    word = sys.argv[2] if len(sys.argv) > 2 else guess_word(out)
    font_path = os.environ.get("SPACE_GROTESK", "SpaceGrotesk-Bold.ttf")
    if len(sys.argv) > 3:
        font_path = sys.argv[3]

    im = Image.new("RGB", (SIZE * SS, SIZE * SS), NAVY)
    d = ImageDraw.Draw(im)

    keel_mark(d, SIZE * SS / 2, SIZE * SS * 0.40, SIZE * SS * 0.50)

    font = ImageFont.truetype(font_path, int(SIZE * SS * 0.155))
    box = d.textbbox((0, 0), word, font=font)
    d.text(
        ((SIZE * SS - (box[2] - box[0])) / 2 - box[0], SIZE * SS * 0.645),
        word,
        font=font,
        fill=WHITE,
    )

    im.resize((SIZE, SIZE), Image.LANCZOS).save(out)
    print(out)


if __name__ == "__main__":
    main()
