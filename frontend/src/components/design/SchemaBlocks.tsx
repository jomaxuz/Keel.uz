// ⚠️ **Plain `<img>`, never `next/image`, for our own uploads.**
//
// Two reasons, and the second one is a bug this file shipped with:
//
//   • The size is already right. `imageUrl(path, w)` asks the backend for the width this
//     card shows (`?w=600`), the result is cached on disk and served `immutable` — so
//     Next's optimiser would resize an already-resized picture and add a hop to the
//     render tier, which is the platform's bottleneck.
//   • ⚠️ **It cannot work here at all.** `/_next/image` fetches the source from the Next
//     server's own origin, and in this deployment `/uploads/*` is routed by the edge
//     rather than by Next (`rewrites()` is sealed at build time — see CLAUDE.md). So the
//     optimiser asks itself for a path it does not serve, gets a 404, and answers 400:
//     every image drawn this way was invisible while the file itself served fine.

// Sections that read their own settings.
//
// ⚠️ **This is the half that makes a schema real.** A settings panel whose values
// nothing consumes is a form, and that is exactly what the constructor had become:
// the console could set a heading and the page would keep showing the restaurant's
// name. Each section here reads the keys its schema declares, and falls back to the
// restaurant's own data when a key is empty — so a band added and left untouched
// still renders something true rather than a gap.

import LocaleLink from "@/components/site/LocaleLink";
import { imageUrl } from "@/lib/api";
import { siteWords } from "@/lib/siteWords";
import { localized } from "@/lib/i18n/site-content";
import type { Dict, Lang } from "@/lib/i18n/dictionaries";
import BannerCarousel from "@/components/site/BannerCarousel";
import type { DesignSection } from "@/lib/types";
import { PerksBlock, type BlockData } from "./blocks";
import { PerkIcon } from "./perkIcons";
import { TONE } from "./tokens";

const ROUND: Record<string, string> = { "": "", lg: "rounded-3xl", full: "rounded-full" };

type Bag = Record<string, unknown>;

/** A three-language value, or "". Settings may also hold a plain string — an older
 *  document, or a value typed before the field was localised. */
// ⚠️ **`data-keel-set` marks where a setting's words are printed.**
//
// The console's live preview writes a changed heading straight into the page
// rather than reloading the customer's site for every letter (see
// PreviewBridge). It finds the node by the settings key that produced it, so a
// band whose heading is not marked falls back to a full reload — which works,
// and is the thing this attribute exists to avoid. Move the text, move the
// attribute; it is not styling and not a test hook.
function text(bag: Bag, key: string, lang: Lang): string {
  const v = bag[key];
  if (typeof v === "string") return v;
  if (v && typeof v === "object") return localized(v as never, lang);
  return "";
}

function str(bag: Bag, key: string, fallback = ""): string {
  const v = bag[key];
  return typeof v === "string" && v !== "" ? v : fallback;
}

function num(bag: Bag, key: string, fallback: number): number {
  const v = bag[key];
  return typeof v === "number" ? v : fallback;
}

function bool(bag: Bag, key: string, fallback: boolean): boolean {
  const v = bag[key];
  return typeof v === "boolean" ? v : fallback;
}

/** A setting holding a list of record ids — which categories a band draws from.
 *  Empty means "all", which is also what an absent key means: a band added and
 *  left alone shows the restaurant's whole menu, not nothing. */
function ids(bag: Bag, key: string): string[] {
  const v = bag[key];
  if (!Array.isArray(v)) return [];
  return v.filter((x): x is string => typeof x === "string");
}

/** ⚠️ A dark overlay is drawn as a layer rather than by dimming the photograph:
 *  dimming the image dims the text over it too, which is the one thing the overlay
 *  exists to keep readable. */
function Overlay({ percent }: { percent: number }) {
  if (percent <= 0) return null;
  return <div className="absolute inset-0 bg-charcoal" style={{ opacity: percent / 100 }} />;
}

export function HeroSection({ d, section }: { d: BlockData; section: DesignSection }) {
  const s = (section.settings ?? {}) as Bag;
  const rest = d.data?.restaurant;
  const heading = text(s, "heading", d.lang) || rest?.name || "";
  const sub = text(s, "subheading", d.lang) || rest?.description || "";
  const eyebrow = text(s, "eyebrow", d.lang);
  const image = str(s, "image") || rest?.coverUrl || "";
  const height = num(s, "height", 80);
  const centred = str(s, "align", "left") === "center";

  return (
    <section
      className={`relative flex w-full items-center ${TONE[str(s, "tone")] ?? ""}`}
      style={{ minHeight: `${height}vh` }}
    >
      {image && (
        <>
          <img
            src={imageUrl(image, 1200) ?? ""}
            alt=""
            className="absolute inset-0 h-full w-full object-cover"
          />
          <Overlay percent={num(s, "overlay", 40)} />
        </>
      )}
      <div
        className={`container-page relative py-16 ${
          centred ? "mx-auto text-center" : ""
        } ${image ? "text-white" : ""}`}
      >
        {eyebrow && <p className="eyebrow" data-keel-set="eyebrow">{eyebrow}</p>}
        <h1
          className="mt-2 font-display text-4xl font-black leading-tight sm:text-6xl"
          data-keel-set="heading"
        >
          {heading}
        </h1>
        {sub && (
          <p
            className={`mt-4 max-w-xl text-base ${centred ? "mx-auto" : ""} ${image ? "text-white/85" : "text-ink-muted"}`}
            data-keel-set="subheading"
          >
            {sub}
          </p>
        )}
        <div className={`mt-8 flex flex-wrap gap-3 ${centred ? "justify-center" : ""}`}>
          <Buttons s={s} d={d} />
        </div>
      </div>
    </section>
  );
}

/** The buttons a band offers.
 *
 *  `fallback` is for the bands whose whole purpose is the button — a call to
 *  action with no button is a heading. Bands that are only *sometimes* about a
 *  button (a hero, an image-and-text) pass none, and stay silent until somebody
 *  types a label. */
function Buttons({
  s,
  d,
  fallback,
}: {
  s: Bag;
  d: BlockData;
  fallback?: {
    primary?: { label: string; href: string };
    secondary?: { label: string; href: string };
  };
}) {
  const one = text(s, "primaryLabel", d.lang) || fallback?.primary?.label || "";
  const two = text(s, "secondaryLabel", d.lang) || fallback?.secondary?.label || "";
  return (
    <>
      {one && (
        <LocaleLink
          href={str(s, "primaryLink", fallback?.primary?.href ?? "/menu")}
          className="btn btn-primary px-6 py-3"
          data-keel-set="primaryLabel"
        >
          {one}
        </LocaleLink>
      )}
      {two && (
        <LocaleLink
          href={str(s, "secondaryLink", fallback?.secondary?.href ?? "/bron")}
          className="btn btn-ghost px-6 py-3"
          data-keel-set="secondaryLabel"
        >
          {two}
        </LocaleLink>
      )}
    </>
  );
}

/** The restaurant's own strip. ⚠️ Its own band rather than something bolted onto the
 *  hero: the hero is the site's face and this is this week, and merging them would mean
 *  editing the identity every time a promotion runs. It renders nothing at all when there
 *  are no banners, so the band can sit in every layout without leaving a gap. */
export function BannersSection({ d }: { d: BlockData; section: DesignSection }) {
  const banners = (d.data?.banners ?? []).filter((b) => b.imageUrl);
  if (banners.length === 0) return null;
  return <BannerCarousel banners={banners} />;
}

export function RichTextSection({ d, section }: { d: BlockData; section: DesignSection }) {
  const s = (section.settings ?? {}) as Bag;
  const heading = text(s, "heading", d.lang);
  const body = text(s, "body", d.lang);
  if (!heading && !body) return null;
  const centred = str(s, "align", "center") === "center";
  return (
    <section className={`w-full ${TONE[str(s, "tone", "surface")] ?? ""}`}>
      <div className={`container-page py-14 ${centred ? "text-center" : ""}`}>
        {heading && <h2 className="section-title" data-keel-set="heading">{heading}</h2>}
        {body && (
          <p
            className={`mt-4 max-w-2xl whitespace-pre-line text-ink-muted ${centred ? "mx-auto" : ""}`}
            data-keel-set="body"
          >
            {body}
          </p>
        )}
      </div>
    </section>
  );
}

export function ImageTextSection({ d, section }: { d: BlockData; section: DesignSection }) {
  const s = (section.settings ?? {}) as Bag;
  const image = str(s, "image") || d.data?.restaurant?.coverUrl || "";
  const heading = text(s, "heading", d.lang);
  const body = text(s, "body", d.lang);
  const right = bool(s, "imageRight", false);
  const round = ROUND[str(s, "round", "lg")] ?? "rounded-3xl";

  return (
    <section className={`w-full ${TONE[str(s, "tone")] ?? ""}`}>
      <div className="container-page grid grid-cols-1 items-center gap-10 py-14 lg:grid-cols-2">
        <div className={`relative aspect-square w-full overflow-hidden ${round} ${right ? "lg:order-2" : ""}`}>
          {image && (
            <img src={imageUrl(image, 1200) ?? ""} alt="" className="absolute inset-0 h-full w-full object-cover" />
          )}
        </div>
        <div>
          {heading && <h2 className="section-title" data-keel-set="heading">{heading}</h2>}
          {body && (
            <p className="mt-4 whitespace-pre-line text-ink-muted" data-keel-set="body">
              {body}
            </p>
          )}
          {text(s, "buttonLabel", d.lang) && (
            <LocaleLink
              href={str(s, "buttonLink", siteWords(d.t, d.data?.brand?.businessType).href)}
              className="btn btn-primary mt-6 px-6 py-3"
              data-keel-set="buttonLabel"
            >
              {text(s, "buttonLabel", d.lang)}
            </LocaleLink>
          )}
        </div>
      </div>
    </section>
  );
}

export function BannerSection({ d, section }: { d: BlockData; section: DesignSection }) {
  const s = (section.settings ?? {}) as Bag;
  const heading = text(s, "heading", d.lang);
  const body = text(s, "body", d.lang);
  const image = str(s, "image");
  if (!heading && !body) return null;
  return (
    <section className={`relative w-full overflow-hidden ${TONE[str(s, "tone", "charcoal")] ?? ""}`}>
      {image && (
        <>
          <img src={imageUrl(image, 1200) ?? ""} alt="" className="absolute inset-0 h-full w-full object-cover" />
          <Overlay percent={num(s, "overlay", 55)} />
        </>
      )}
      <div className="container-page relative py-16 text-center">
        {heading && (
          <h2
            className="font-display text-3xl font-black leading-tight sm:text-4xl"
            data-keel-set="heading"
          >
            {heading}
          </h2>
        )}
        {body && (
          <p
            className="mx-auto mt-3 max-w-xl whitespace-pre-line opacity-85"
            data-keel-set="body"
          >
            {body}
          </p>
        )}
        {text(s, "buttonLabel", d.lang) && (
          <LocaleLink
            href={str(s, "buttonLink", siteWords(d.t, d.data?.brand?.businessType).href)}
            className="btn btn-primary mt-7 px-6 py-3"
            data-keel-set="buttonLabel"
          >
            {text(s, "buttonLabel", d.lang)}
          </LocaleLink>
        )}
      </div>
    </section>
  );
}

/** Photographs, one per block. ⚠️ Blocks rather than a list of URLs in one
 *  setting: a repeatable item is the thing an operator adds, removes and reorders,
 *  and a newline-separated field cannot be reordered without retyping it. */
export function GallerySection({ d, section }: { d: BlockData; section: DesignSection }) {
  const s = (section.settings ?? {}) as Bag;
  const shots = (section.blocks ?? []).filter((b) => b.type === "photo" && !b.hidden);
  // ⚠️ With no photographs added, fall back to the dish images the restaurant has
  // already uploaded — the only pool of pictures every tenant has, and what this
  // band drew from before it became schema-driven. Requiring hand-added photos
  // made it render nothing on every site, including one where it was already live.
  if (shots.length === 0) {
    const items = d.menu
      .flatMap((g) => g.items)
      .filter((i) => i.imageUrl)
      .slice(0, num(s, "limit", 8));
    if (items.length === 0) return null;
    return (
      <section className={`w-full ${TONE[str(s, "tone")] ?? ""}`}>
        <div className="container-page py-14">
          {text(s, "heading", d.lang) && (
            <h2 className="section-title mb-6" data-keel-set="heading">{text(s, "heading", d.lang)}</h2>
          )}
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
            {items.map((it) => (
              <figure key={it.id} className="relative aspect-square overflow-hidden rounded-2xl">
                <img
                  src={imageUrl(it.imageUrl, 600) ?? ""}
                  alt=""
                  className="absolute inset-0 h-full w-full object-cover"
                />
              </figure>
            ))}
          </div>
        </div>
      </section>
    );
  }
  return (
    <section className={`w-full ${TONE[str(s, "tone")] ?? ""}`}>
      <div className="container-page py-14">
        {text(s, "heading", d.lang) && (
          <h2 className="section-title mb-6" data-keel-set="heading">{text(s, "heading", d.lang)}</h2>
        )}
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
          {shots.map((b, i) => {
            const bag = (b.settings ?? {}) as Bag;
            const src = str(bag, "image");
            if (!src) return null;
            return (
              <figure key={i} className="relative aspect-square overflow-hidden rounded-2xl">
                <img src={imageUrl(src, 600) ?? ""} alt="" className="absolute inset-0 h-full w-full object-cover" />
                {text(bag, "caption", d.lang) && (
                  <figcaption className="absolute inset-x-0 bottom-0 bg-charcoal/70 px-2 py-1 text-[11px] text-white">
                    {text(bag, "caption", d.lang)}
                  </figcaption>
                )}
              </figure>
            );
          })}
        </div>
      </div>
    </section>
  );
}

/** Three or four cards. ⚠️ Blocks, not settings named `perk1Title`…`perk3Body`:
 *  numbered settings are how a section ends up with a fixed count and a panel full
 *  of empty fields, and they cannot be reordered. */
export function PerksSection({ d, section }: { d: BlockData; section: DesignSection }) {
  const s = (section.settings ?? {}) as Bag;
  // ⚠️ The owner's switch wins even over a console-composed band. What the
  // cards claim — that this place delivers, that it takes cards — is a
  // statement about the business, not styling, and the restaurant is the one
  // answerable for it. The words themselves still come from the design when one
  // was drawn: that is the part somebody paid us for.
  if (d.data?.restaurant?.content?.hidePerks) return null;
  const cards = (section.blocks ?? []).filter((b) => b.type === "perk" && !b.hidden);
  // ⚠️ **The widest instance of the missing-fallback bug, and the one CLAUDE.md
  // warned about by name.** `perks` is in the default layout, so requiring
  // hand-added cards did not empty one band on one design — it removed the three
  // cards under the hero from **every restaurant's home page**, including the
  // ones that never opened the constructor. "Existing customers see no change"
  // is the precondition of the whole feature; this was the line that broke it.
  if (cards.length === 0) return <PerksBlock d={d} section={section} />;
  return (
    <section className={`w-full ${TONE[str(s, "tone", "surface")] ?? ""}`}>
      <div className="container-page py-14">
        {text(s, "heading", d.lang) && (
          <h2 className="section-title mb-8" data-keel-set="heading">{text(s, "heading", d.lang)}</h2>
        )}
        <div className="grid grid-cols-2 gap-3 sm:gap-6 lg:grid-cols-3">
          {cards.map((b, i) => {
            const bag = (b.settings ?? {}) as Bag;
            return (
              <div key={i} className="card p-6">
                <PerkIcon name={str(bag, "icon", "star")} />
                <h3 className="mt-4 font-display text-lg font-bold">
                  {text(bag, "title", d.lang)}
                </h3>
                <p className="mt-2 text-sm text-ink-muted">{text(bag, "body", d.lang)}</p>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}

/** Category tiles. The tiles themselves come from the menu — a section that let
 *  somebody type category names would let them type names the menu does not have. */
export function CategoriesSection({ d, section }: { d: BlockData; section: DesignSection }) {
  const s = (section.settings ?? {}) as Bag;
  const limit = num(s, "limit", 6);
  // A chosen subset, in the *menu's* order rather than the order they were
  // clicked: the tiles sit next to a menu page that uses the restaurant's own
  // sort, and two orders for the same list is the kind of difference a guest
  // reads as the site being wrong without being able to say why.
  const chosen = ids(s, "categories");
  const all = (d.menu ?? []).filter(
    (g) => chosen.length === 0 || chosen.includes(g.category.id),
  );
  // Every chosen category deleted since leaves nothing to draw. Falling back to
  // the whole list keeps the band alive, which is the honest failure: the tiles
  // are a way into the menu, and no way in is worse than the wrong way in.
  const groups = (all.length > 0 ? all : (d.menu ?? [])).slice(0, limit);
  if (groups.length === 0) return null;
  return (
    <section className={`w-full ${TONE[str(s, "tone")] ?? ""}`}>
      <div className="container-page py-14">
        {text(s, "heading", d.lang) && (
          <h2 className="section-title mb-6" data-keel-set="heading">{text(s, "heading", d.lang)}</h2>
        )}
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
          {groups.map((g) => (
            <LocaleLink
              key={g.category.id}
              href={`${siteWords(d.t, d.data?.brand?.businessType).href}#cat-${g.category.slug || g.category.id}`}
              className="card flex flex-col items-center gap-2 p-4 text-center"
            >
              {g.category.imageUrl && (
                <span className="relative h-14 w-14 overflow-hidden rounded-full">
                  <img src={imageUrl(g.category.imageUrl, 300) ?? ""} alt="" className="absolute inset-0 h-full w-full object-cover" />
                </span>
              )}
              <span className="text-sm font-semibold">{g.category.name}</span>
            </LocaleLink>
          ))}
        </div>
      </div>
    </section>
  );
}

export function CtaSection({ d, section }: { d: BlockData; section: DesignSection }) {
  const s = (section.settings ?? {}) as Bag;
  // ⚠️ Falls back to the dictionary's own wording, which is what this band said
  // before it became schema-driven. Without it the band rendered **nothing** until
  // somebody typed a heading — and "add a band, see no change" is indistinguishable
  // from a broken editor. It was live that way on a real customer's home page.
  // ⚠️ The fallback follows the business, not the kitchen: on a shop these read
  // "Savatga soling", not "Ochlik kutib turmaydi" (lib/siteWords.ts).
  const words = siteWords(d.t, d.data?.brand?.businessType);
  const heading = text(s, "heading", d.lang) || words.orderTitle;
  const body = text(s, "body", d.lang) || words.orderText;
  const centred = str(s, "align", "center") === "center";
  return (
    <section className={`w-full ${TONE[str(s, "tone", "brand")] ?? ""}`}>
      <div className={`container-page py-14 ${centred ? "text-center" : ""}`}>
        {heading && (
          <h2
            className="font-display text-2xl font-black leading-tight sm:text-3xl"
            data-keel-set="heading"
          >
            {heading}
          </h2>
        )}
        {body && (
          <p
            className={`mt-3 max-w-xl opacity-90 ${centred ? "mx-auto" : ""}`}
            data-keel-set="body"
          >
            {body}
          </p>
        )}
        <div className={`mt-7 flex flex-wrap gap-3 ${centred ? "justify-center" : ""}`}>
          {/* The second button only when the restaurant takes bookings: offering a
              table to a place that does not seat people is the band actively
              lying, which is worse than a band with one button. */}
          <Buttons
            s={s}
            d={d}
            fallback={{
              primary: { label: d.t.home.orderBtn, href: words.href },
              secondary: d.data?.restaurant?.booking?.enabled
                ? { label: d.t.nav.booking, href: "/bron" }
                : undefined,
            }}
          />
        </div>
      </div>
    </section>
  );
}

/** About: the same shape as image-with-text, but it falls back to the restaurant's
 *  own `content.aboutTitle` / `aboutText` — which is where that copy already lives,
 *  and which the owner edits in their own panel. */
export function AboutSection({ d, section }: { d: BlockData; section: DesignSection }) {
  const s = (section.settings ?? {}) as Bag;
  const rest = d.data?.restaurant;
  const content = rest?.content;
  // The same chain the /about page walks, and the same one the pre-schema band
  // used: what the design says, then what the owner wrote in their panel, then
  // the restaurant's own name and description. Two links of it had been dropped,
  // which is why a restaurant that had not filled in "Biz haqimizda" got an
  // invisible band.
  const heading =
    text(s, "heading", d.lang) || localized(content?.aboutTitle, d.lang) || rest?.name || "";
  const body =
    text(s, "body", d.lang) ||
    localized(content?.aboutText, d.lang) ||
    rest?.description ||
    "";
  // Still nothing when the restaurant has no description at all: a heading with
  // no text under it is a band that looks unfinished rather than one that says
  // something true.
  if (!body) return null;
  const merged: DesignSection = {
    ...section,
    settings: {
      ...s,
      heading: { uz: heading, ru: heading, en: heading },
      body: { uz: body, ru: body, en: body },
      round: "lg",
    },
  };
  return <ImageTextSection d={d} section={merged} />;
}

export type { Dict };
