// What the site's header looks like, read out of the drawn design.
//
// ⚠️ **The `navbar` band is configuration, not a band, and that is why it draws
// nothing in the page flow.** The header is rendered by the site shell, above
// `<main>`, on every page — not only on the home page the design describes. A
// `navbar` block rendered inline would put a second bar under the real one on
// the home page and leave every other page with the built-in bar, which is the
// worst of both. So the band carries the settings and this file turns them into
// the shell's own props.
//
// ⚠️ **It used to draw nothing at all, silently.** All five built-in templates
// open with a `navbar` band; the renderer has no entry for it, so `if (!Block)
// return null` quietly skipped every one of them. An operator could pick
// "Markazda" or "Shaffof" in the console, save, publish, and watch the site not
// change — with nothing anywhere saying why.
//
// ⚠️ **Every default is today's header**, field by field. A design that has no
// navbar band, or one left untouched, must produce the bar every customer has
// now — the same precondition `DEFAULT_SECTIONS` exists for, one level up.

import type { PageDesign } from "@/lib/types";

/** Which controls the bar carries on a desktop. */
export interface ChromeIcons {
  /** ⚠️ Off by default because today's header has none. A shop wants one — the
   *  magnifier is the second thing a guest reaches for in a catalogue — and a
   *  restaurant would be given a control it has no use for. */
  search: boolean;
  cart: boolean;
  account: boolean;
  lang: boolean;
  theme: boolean;
}

export interface SiteChrome {
  variant: "classic" | "centered" | "minimal" | "transparent";
  tone: string;
  /** How wide the bar's contents run — the same three steps a band has. */
  width: "" | "wide" | "full";
  sticky: boolean;
  icons: ChromeIcons;
  /** How the phone menu opens: the panel under the bar (today), or a drawer
   *  over the page. */
  burger: "panel" | "drawer";
  /** Lowercase nav with wide letter spacing — the fashion-site convention, and
   *  the one thing that makes a bar copied from a lookbook read as one.
   *
   *  ⚠️ Named for what it does. It was `uppercase` for an hour, which is the
   *  opposite of the effect and exactly the kind of setting somebody switches
   *  on twice before reading the code. */
  lowercase: boolean;
}

/** The bar every site has today. */
export const DEFAULT_CHROME: SiteChrome = {
  variant: "classic",
  tone: "",
  width: "",
  sticky: true,
  icons: { search: false, cart: true, account: true, lang: true, theme: true },
  burger: "panel",
  lowercase: false,
};

const VARIANTS = new Set(["classic", "centered", "minimal", "transparent"]);
const TONES = new Set(["", "surface", "raised", "charcoal", "brand", "accent", "ink"]);
const WIDTHS = new Set(["", "wide", "full"]);
const BURGERS = new Set(["panel", "drawer"]);

function bool(bag: Record<string, unknown>, key: string, fallback: boolean): boolean {
  const v = bag[key];
  return typeof v === "boolean" ? v : fallback;
}

/** Reads the header's settings out of a design.
 *
 *  ⚠️ **Every unknown value falls back rather than being refused**, exactly as
 *  the business-type predicates do. A design written by a newer console must
 *  lose the tailoring it asked for, never the header — a site whose bar failed
 *  to render is a site with no way to reach the catalogue. */
export function siteChrome(design?: PageDesign | null): SiteChrome {
  const band = design?.sections?.find((s) => s.type === "navbar" && !s.hidden);
  if (!band) return DEFAULT_CHROME;
  const s = (band.settings ?? {}) as Record<string, unknown>;

  // ⚠️ The variant lives on the band itself, not in `settings` — that is where
  // the schema puts it and where the five built-in templates already write it.
  const variant = typeof band.variant === "string" && VARIANTS.has(band.variant)
    ? (band.variant as SiteChrome["variant"])
    : DEFAULT_CHROME.variant;

  const tone = typeof s.tone === "string" && TONES.has(s.tone) ? s.tone : "";
  const width =
    typeof band.style?.width === "string" && WIDTHS.has(band.style.width)
      ? (band.style.width as SiteChrome["width"])
      : "";
  const burger =
    typeof s.burger === "string" && BURGERS.has(s.burger)
      ? (s.burger as SiteChrome["burger"])
      : DEFAULT_CHROME.burger;

  return {
    variant,
    tone,
    width,
    sticky: bool(s, "sticky", DEFAULT_CHROME.sticky),
    icons: {
      search: bool(s, "iconSearch", DEFAULT_CHROME.icons.search),
      cart: bool(s, "iconCart", DEFAULT_CHROME.icons.cart),
      account: bool(s, "iconAccount", DEFAULT_CHROME.icons.account),
      lang: bool(s, "iconLang", DEFAULT_CHROME.icons.lang),
      theme: bool(s, "iconTheme", DEFAULT_CHROME.icons.theme),
    },
    burger,
    lowercase: bool(s, "lowercase", DEFAULT_CHROME.lowercase),
  };
}

/** The bar's background, from the design system's own tokens.
 *
 *  ⚠️ Never a colour: a bar painted `#fffbe6` would stay that colour in dark
 *  mode and ignore the accent the shop chose. `transparent` is not a tone —
 *  it is the absence of one, and it is the variant's job. */
export const CHROME_TONE: Record<string, string> = {
  "": "border-b border-line bg-cream/85 backdrop-blur-md",
  surface: "border-b border-line bg-surface",
  raised: "border-b border-line bg-raised",
  charcoal: "border-b border-white/10 bg-charcoal text-white",
  brand: "border-b border-white/10 bg-brand text-white",
  accent: "border-b border-black/10 bg-accent text-accent-ink",
  ink: "border-b border-white/10 bg-ink text-cream",
};

/** The width variables the bar's own container reads. Same mechanism the bands
 *  use — see DesignRenderer's WIDTH_VARS and `.container-page` in globals.css. */
export const CHROME_WIDTH: Record<string, React.CSSProperties> = {
  "": {},
  wide: { "--band-max": "96rem" } as React.CSSProperties,
  full: { "--band-max": "100%", "--band-pad": "1.5rem" } as React.CSSProperties,
};
