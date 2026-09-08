#!/usr/bin/env python3
"""Turn one restaurant's logo into a launcher icon and a splash mark.

⚠️ **The two are not the same picture, and treating them as one is the mistake
this file exists to avoid.** A launcher icon is cropped by whatever mask the
phone's launcher applies — a circle, a squircle, a rounded square — and anything
in the outer eighteenth of the canvas is gone. A splash mark is shown whole on a
white field. A logo scaled to fill the icon looks right in a preview and loses
its own name on half the phones in the country.

⚠️ **White ground, in both schemes, for both.** Restaurant logos are drawn for
paper: dark ink, no light variant, almost every time. On a dark splash such a
mark is invisible, which is indistinguishable from an app that failed to start.

⚠️ **Transparency is flattened onto white rather than kept.** A PNG with an
alpha channel is fine in the foreground layer of an adaptive icon, but the
legacy 512×512 the store listing needs must not have one — Play refuses it — and
a logo that was drawn as white-on-transparent would otherwise arrive as a blank
square nobody notices until the listing is live.
"""

import sys
from pathlib import Path

from PIL import Image

# The adaptive icon's canvas is 108dp and the launcher may crop everything
# outside the middle 72dp. Anything drawn past this fraction of the width can be
# cut, so the mark is fitted inside it with room to spare.
SAFE = 0.60

# What the splash screen shows. Android 12+ masks the animated icon to a circle
# and only the inner two-thirds is reliably visible, so the same rule applies
# with a slightly kinder margin.
SPLASH_SAFE = 0.66

DENSITIES = {"mdpi": 48, "hdpi": 72, "xhdpi": 96, "xxhdpi": 144, "xxxhdpi": 192}


def load(path: Path) -> Image.Image:
    img = Image.open(path)
    # ⚠️ Converted rather than assumed: a logo arrives as a palette PNG, a
    # greyscale JPEG or a CMYK TIFF often enough, and `paste` with a mask throws
    # on all three.
    return img.convert("RGBA")


def fitted(logo: Image.Image, canvas: int, safe: float) -> Image.Image:
    """The logo centred on a transparent square, scaled to the safe area.

    ⚠️ **Fitted by its longest side, never stretched.** A wordmark is four times
    as wide as it is tall, and a logo squeezed into a square is a restaurant's
    name spelled wrongly on every phone that installs it.
    """
    out = Image.new("RGBA", (canvas, canvas), (0, 0, 0, 0))
    box = int(canvas * safe)
    scaled = logo.copy()
    scaled.thumbnail((box, box), Image.LANCZOS)
    out.paste(
        scaled,
        ((canvas - scaled.width) // 2, (canvas - scaled.height) // 2),
        scaled,
    )
    return out


def on_white(img: Image.Image) -> Image.Image:
    ground = Image.new("RGBA", img.size, (255, 255, 255, 255))
    ground.paste(img, (0, 0), img)
    return ground.convert("RGB")


def main() -> int:
    if len(sys.argv) != 3:
        print("usage: brandimages.py <logo.png> <res-dir>", file=sys.stderr)
        return 2
    logo = load(Path(sys.argv[1]))
    res = Path(sys.argv[2])

    # ---- The placeholders go first ----
    #
    # ⚠️ **Deleted, not written around.** Two files of the same resource name in
    # the same qualifier — `drawable/brand_logo.xml` and a PNG beside it — is a
    # duplicate-resource error, and relying on a qualifier to out-rank the
    # default is the kind of thing that works until somebody adds a density.
    # The vectors exist so a fresh checkout runs; a branded build replaces them.
    for placeholder in ("brand_logo.xml", "ic_launcher_foreground.xml"):
        (res / "drawable" / placeholder).unlink(missing_ok=True)

    # ---- The splash mark ----
    (res / "drawable").mkdir(parents=True, exist_ok=True)
    fitted(logo, 768, SPLASH_SAFE).save(res / "drawable" / "brand_logo.png")

    # ---- The adaptive icon's foreground ----
    #
    # ⚠️ **`drawable`, not `mipmap`, because that is what the icon asks for.**
    # `mipmap-anydpi-v26/ic_launcher.xml` names `@drawable/ic_launcher_foreground`
    # — a foreground written to `mipmap-*` is a different resource entirely, so
    # the build succeeds, the icon is generated, and every phone shows the
    # placeholder. Nothing errors and nobody notices until an app is installed.
    #
    # 432px is the 108dp canvas at xxxhdpi; every density scales down from it.
    fg = fitted(logo, 432, SAFE)
    for name, size in DENSITIES.items():
        drawable = res / f"drawable-{name}"
        drawable.mkdir(parents=True, exist_ok=True)
        # The foreground layer keeps its transparency: the white comes from the
        # background layer, which is a solid colour and needs no file.
        edge = size * 108 // 48
        fg.resize((edge, edge), Image.LANCZOS).save(
            drawable / "ic_launcher_foreground.png"
        )

        # ⚠️ **The legacy square and round icons as well.** minSdk is 26, so the
        # adaptive icon covers every phone that can install this — but a few
        # launchers and the recents switcher still read the legacy names, and
        # the one that finds them missing draws a grey android instead.
        mipmap = res / f"mipmap-{name}"
        mipmap.mkdir(parents=True, exist_ok=True)
        square = on_white(fitted(logo, size, SAFE))
        square.save(mipmap / "ic_launcher.png")
        square.save(mipmap / "ic_launcher_round.png")

    # ---- What the store listing needs ----
    # ⚠️ 512×512, no alpha: Play refuses an icon with a transparent channel, and
    # the refusal arrives at the end of an upload somebody has waited for.
    on_white(fitted(logo, 512, SAFE)).save(res / "play_icon.png")
    print("icons written to", res)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
