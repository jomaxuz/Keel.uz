// Sections that read their own settings.
//
// ⚠️ **This is the half that makes a schema real.** A settings panel whose values
// nothing consumes is a form, and that is exactly what the constructor had become:
// the console could set a heading and the page would keep showing the restaurant's
// name. Each section here reads the keys its schema declares, and falls back to the
// restaurant's own data when a key is empty — so a band added and left untouched
// still renders something true rather than a gap.

import Image from "next/image";
import LocaleLink from "@/components/site/LocaleLink";
import { imageUrl } from "@/lib/api";
import { localized } from "@/lib/i18n/site-content";
import type { Dict, Lang } from "@/lib/i18n/dictionaries";
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
          <Image
            src={imageUrl(image, 1200) ?? ""}
            alt=""
            fill
            priority
            className="object-cover"
            sizes="100vw"
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
            <Image src={imageUrl(image, 1200) ?? ""} alt="" fill className="object-cover" sizes="50vw" />
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
          <Image src={imageUrl(image, 1200) ?? ""} alt="" fill className="object-cover" sizes="100vw" />
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
                <Image src={imageUrl(src, 600) ?? ""} alt="" fill className="object-cover" sizes="25vw" />
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

export type { Dict };
