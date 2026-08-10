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
import { localized } from "@/lib/i18n/site-content";
import type { Dict, Lang } from "@/lib/i18n/dictionaries";
import BannerCarousel from "@/components/site/BannerCarousel";
import type { DesignSection } from "@/lib/types";
import type { BlockData } from "./blocks";

const TONE: Record<string, string> = {
  "": "",
  surface: "bg-surface",
  raised: "bg-raised",
  charcoal: "bg-charcoal text-white",
  brand: "bg-brand text-white",
};

const ROUND: Record<string, string> = { "": "", lg: "rounded-3xl", full: "rounded-full" };

type Bag = Record<string, unknown>;

/** A three-language value, or "". Settings may also hold a plain string — an older
 *  document, or a value typed before the field was localised. */
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
        {eyebrow && <p className="eyebrow">{eyebrow}</p>}
        <h1 className="mt-2 font-display text-4xl font-black leading-tight sm:text-6xl">
          {heading}
        </h1>
        {sub && (
          <p className={`mt-4 max-w-xl text-base ${centred ? "mx-auto" : ""} ${image ? "text-white/85" : "text-ink-muted"}`}>
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

function Buttons({ s, d }: { s: Bag; d: BlockData }) {
  const one = text(s, "primaryLabel", d.lang);
  const two = text(s, "secondaryLabel", d.lang);
  return (
    <>
      {one && (
        <LocaleLink href={str(s, "primaryLink", "/menu")} className="btn btn-primary px-6 py-3">
          {one}
        </LocaleLink>
      )}
      {two && (
        <LocaleLink href={str(s, "secondaryLink", "/bron")} className="btn btn-ghost px-6 py-3">
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
        {heading && <h2 className="section-title">{heading}</h2>}
        {body && (
          <p className={`mt-4 max-w-2xl whitespace-pre-line text-ink-muted ${centred ? "mx-auto" : ""}`}>
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
          {heading && <h2 className="section-title">{heading}</h2>}
          {body && <p className="mt-4 whitespace-pre-line text-ink-muted">{body}</p>}
          {text(s, "buttonLabel", d.lang) && (
            <LocaleLink href={str(s, "buttonLink", "/menu")} className="btn btn-primary mt-6 px-6 py-3">
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
          <h2 className="font-display text-3xl font-black leading-tight sm:text-4xl">{heading}</h2>
        )}
        {body && <p className="mx-auto mt-3 max-w-xl whitespace-pre-line opacity-85">{body}</p>}
        {text(s, "buttonLabel", d.lang) && (
          <LocaleLink href={str(s, "buttonLink", "/menu")} className="btn btn-primary mt-7 px-6 py-3">
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
  if (shots.length === 0) return null;
  return (
    <section className={`w-full ${TONE[str(s, "tone")] ?? ""}`}>
      <div className="container-page py-14">
        {text(s, "heading", d.lang) && (
          <h2 className="section-title mb-6">{text(s, "heading", d.lang)}</h2>
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
  const cards = (section.blocks ?? []).filter((b) => b.type === "perk" && !b.hidden);
  if (cards.length === 0) return null;
  return (
    <section className={`w-full ${TONE[str(s, "tone", "surface")] ?? ""}`}>
      <div className="container-page py-14">
        {text(s, "heading", d.lang) && (
          <h2 className="section-title mb-8">{text(s, "heading", d.lang)}</h2>
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

const PERK_PATHS: Record<string, string> = {
  star: "M12 3l2.9 5.9 6.5.9-4.7 4.6 1.1 6.5L12 18l-5.8 3 1.1-6.5L2.6 9.8l6.5-.9L12 3Z",
  clock: "M12 7v5l3 2M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z",
  truck: "M3 7h11v9H3zM14 10h4l3 3v3h-7zM7 19a2 2 0 1 0 0-4 2 2 0 0 0 0 4Zm10 0a2 2 0 1 0 0-4 2 2 0 0 0 0 4Z",
  leaf: "M20 4C10 4 4 9 4 17v3M20 4c0 8-5 12-12 12",
  fire: "M12 22c4 0 7-2.7 7-6.5 0-4.5-4.5-6.5-4-11.5-3 1.5-5 4-5 7 0-1.5-1-2.5-2-3-.7 1.5-3 3.4-3 7.5C5 19.3 8 22 12 22Z",
  check: "m5 13 4 4L19 7",
  heart: "M12 21s-8-4.8-8-10a4.5 4.5 0 0 1 8-2.8A4.5 4.5 0 0 1 20 11c0 5.2-8 10-8 10Z",
  chef: "M7 21h10M6 17h12v-2a6 6 0 0 0-12 0v2Z",
  phone: "M22 16.9v3a2 2 0 0 1-2.2 2 19.8 19.8 0 0 1-8.6-3.1 19.5 19.5 0 0 1-6-6A19.8 19.8 0 0 1 2.1 4.2 2 2 0 0 1 4.1 2h3a2 2 0 0 1 2 1.7c.1 1 .4 1.9.7 2.8a2 2 0 0 1-.5 2.1L8.1 9.9a16 16 0 0 0 6 6l1.3-1.2a2 2 0 0 1 2.1-.5c.9.3 1.8.6 2.8.7a2 2 0 0 1 1.7 2Z",
  pin: "M12 22s7-5.6 7-12a7 7 0 1 0-14 0c0 6.4 7 12 7 12ZM12 11a2 2 0 1 0 0-4 2 2 0 0 0 0 4Z",
};

function PerkIcon({ name }: { name: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6"
      strokeLinecap="round" strokeLinejoin="round" className="h-7 w-7 text-brand" aria-hidden>
      <path d={PERK_PATHS[name] ?? PERK_PATHS.star} />
    </svg>
  );
}

/** Category tiles. The tiles themselves come from the menu — a section that let
 *  somebody type category names would let them type names the menu does not have. */
export function CategoriesSection({ d, section }: { d: BlockData; section: DesignSection }) {
  const s = (section.settings ?? {}) as Bag;
  const limit = num(s, "limit", 6);
  const groups = (d.menu ?? []).slice(0, limit);
  if (groups.length === 0) return null;
  return (
    <section className={`w-full ${TONE[str(s, "tone")] ?? ""}`}>
      <div className="container-page py-14">
        {text(s, "heading", d.lang) && (
          <h2 className="section-title mb-6">{text(s, "heading", d.lang)}</h2>
        )}
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
          {groups.map((g) => (
            <LocaleLink
              key={g.category.id}
              href={`/menu#cat-${g.category.slug || g.category.id}`}
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
  const heading = text(s, "heading", d.lang);
  const body = text(s, "body", d.lang);
  if (!heading && !body) return null;
  const centred = str(s, "align", "center") === "center";
  return (
    <section className={`w-full ${TONE[str(s, "tone", "brand")] ?? ""}`}>
      <div className={`container-page py-14 ${centred ? "text-center" : ""}`}>
        {heading && (
          <h2 className="font-display text-2xl font-black leading-tight sm:text-3xl">{heading}</h2>
        )}
        {body && <p className={`mt-3 max-w-xl opacity-90 ${centred ? "mx-auto" : ""}`}>{body}</p>}
        <div className={`mt-7 flex flex-wrap gap-3 ${centred ? "justify-center" : ""}`}>
          <Buttons s={s} d={d} />
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
  const content = d.data?.restaurant?.content;
  const heading = text(s, "heading", d.lang) || localized(content?.aboutTitle, d.lang);
  const body = text(s, "body", d.lang) || localized(content?.aboutText, d.lang);
  if (!heading && !body) return null;
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
