#!/usr/bin/env python3
"""Every image this application ships, made from the three Keel masters.

⚠️ **Run again rather than edited by hand.** The launcher icon, the adaptive
foreground, the splash mark and the leanback banner are four sizes of one
identity; touching up one of them in an editor is how a television ends up with
a slightly different orange from the phone next to it. The masters live in
`mobile/tv/assets` — the same files the phone applications draw from, which is
why the mark is identical on all of them.

    python3 tools/icons.py

⚠️ **The banner exists only here.** A leanback home screen draws a 320x180 tile
for every application and there is no icon fallback: without one the entry is a
blank rectangle with a label under it, and Play review refuses the listing.
"""

import os
from PIL import Image, ImageDraw, ImageFont

HERE = os.path.dirname(os.path.abspath(__file__))
SRC = os.path.abspath(os.path.join(HERE, "..", "..", "tv", "assets"))
RES = os.path.abspath(os.path.join(HERE, "..", "app", "src", "main", "res"))

NAVY = (0, 11, 28, 255)
ORANGE = (226, 89, 13, 255)

# The five densities Android asks for, as multiples of a baseline pixel.
DENSITIES = {"mdpi": 1, "hdpi": 1.5, "xhdpi": 2, "xxhdpi": 3, "xxxhdpi": 4}


def out(folder, name, img):
    d = os.path.join(RES, folder)
    os.makedirs(d, exist_ok=True)
    img.save(os.path.join(d, name))
    print(os.path.join(folder, name), img.size)


def scaled(img, size):
    return img.resize((size, size), Image.LANCZOS)


def circle_mask(img):
    """The round launcher icon, actually round.

    ⚠️ A square PNG under `roundIcon` is drawn square by the launchers that ask
    for it — which is most of the ones shipped on cheap Android TV boxes."""
    mask = Image.new("L", img.size, 0)
    ImageDraw.Draw(mask).ellipse((0, 0, img.size[0] - 1, img.size[1] - 1), fill=255)
    result = img.copy()
    result.putalpha(mask)
    return result


def main():
    icon = Image.open(os.path.join(SRC, "icon.png")).convert("RGBA")
    fg = Image.open(os.path.join(SRC, "android-icon-foreground.png")).convert("RGBA")
    mono = Image.open(os.path.join(SRC, "android-icon-monochrome.png")).convert("RGBA")

    # ---- The launcher icon, legacy and round ----
    for dpi, mult in DENSITIES.items():
        size = int(round(48 * mult))
        square = scaled(icon, size)
        out(f"mipmap-{dpi}", "ic_launcher.png", square)
        out(f"mipmap-{dpi}", "ic_launcher_round.png", circle_mask(square))

    # ---- The adaptive icon's two layers ----
    #
    # ⚠️ 108dp at xxxhdpi, which is 432 px. The visible circle is the middle
    # 72dp; the master already carries that padding, which is why it is scaled
    # rather than re-composed here.
    out("drawable-xxxhdpi", "ic_launcher_foreground.png", scaled(fg, 432))
    out("drawable-xxxhdpi", "ic_launcher_monochrome.png", scaled(mono, 432))

    # ---- The splash mark ----
    #
    # ⚠️ **The same geometry the owner application ships**, worked out from its
    # generated file rather than re-invented: the canvas is 288dp and the mark
    # sits well inside it. A splash whose mark is a different size from the
    # phone's is the sort of difference nobody reports and everybody sees.
    for dpi, mult in DENSITIES.items():
        canvas = int(round(288 * mult))
        mark = int(round(canvas / 1.4922))
        sheet = Image.new("RGBA", (canvas, canvas), (0, 0, 0, 0))
        art = scaled(fg, mark)
        sheet.paste(art, ((canvas - mark) // 2, (canvas - mark) // 2), art)
        out(f"drawable-{dpi}", "splash_icon.png", sheet)

    # ---- The leanback banner ----
    #
    # ⚠️ **The mark and the name, on the app's own navy.** A banner is read
    # from four metres on a home row beside Netflix and YouTube, so it carries
    # the two things that identify it at that distance and nothing else. The
    # xhdpi size (320x180) is the one the platform documents; the larger one is
    # for the sets that draw the row bigger.
    font_path = "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf"
    for dpi, mult in (("xhdpi", 1), ("xxhdpi", 1.5), ("xxxhdpi", 2)):
        w, h = int(round(320 * mult)), int(round(180 * mult))
        banner = Image.new("RGBA", (w, h), NAVY)
        art_size = int(round(h * 0.62))
        art = scaled(fg, art_size)
        banner.paste(art, (int(h * 0.10), (h - art_size) // 2), art)
        draw = ImageDraw.Draw(banner)
        font = ImageFont.truetype(font_path, int(round(h * 0.20)))
        x = int(h * 0.10) + art_size + int(h * 0.02)
        box = draw.textbbox((0, 0), "Keel TV", font=font)
        draw.text((x, (h - (box[3] - box[1])) // 2 - box[1]), "Keel TV",
                  font=font, fill=(255, 255, 255, 255))
        out(f"drawable-{dpi}", "tv_banner.png", banner)


if __name__ == "__main__":
    main()
