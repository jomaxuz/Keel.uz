"use client";

// A band with freely placed elements: the answer to "make it look like this
// picture".
//
// Every other block in this folder has a fixed inner layout — a hero is a hero.
// This one is an empty box somebody composed inside, and it exists because a real
// design brief is a screenshot from Pinterest, not a choice between three heroes.
//
// Three things make free placement survive contact with a phone:
//
//   • **Percent, never pixels.** A box at `x=24%` has exactly one position on any
//     screen; a box at `x=340px` has none on a 380px phone. Same reason the band
//     heights are in `vh`.
//
//   • **A separate phone layout, per element.** ⚠️ There is no arithmetic that
//     turns a desktop composition into a phone one: elements side by side have to
//     become one column, and only a person knows in what order. So each element
//     may carry a `mobile` box, and when none of them do the whole band falls back
//     to **stacked flow** — elements in drawn order, full width, no overlap. That
//     fallback is not a compromise, it is the thing that stops this feature
//     shipping beautiful pages that are unusable on the screen most guests use.
//
//   • **Functional widgets keep their own insides.** A menu placed here is
//     positioned and sized freely and still renders the real menu, with prices,
//     translations, options and a working cart. A "free" editor that made people
//     rebuild those out of text and rectangles would produce a page that cannot
//     take an order.

import Image from "next/image";
import LocaleLink from "@/components/site/LocaleLink";
import { imageUrl } from "@/lib/api";
import { localized } from "@/lib/i18n/site-content";
import type { Dict, Lang } from "@/lib/i18n/dictionaries";
import type { DesignBox, DesignCanvas, DesignElement } from "@/lib/types";

const TONE_CLASS: Record<string, string> = {
  "": "",
  surface: "bg-surface",
  raised: "bg-raised",
  charcoal: "bg-charcoal text-white",
  brand: "bg-brand text-white",
};

const COLOR_CLASS: Record<string, string> = {
  "": "",
  ink: "text-ink",
  soft: "text-ink-soft",
  muted: "text-ink-muted",
  white: "text-white",
  brand: "text-brand",
  surface: "text-surface",
  charcoal: "text-charcoal",
};

// Font size as a step on the type scale rather than a pixel value: a headline
// stays proportional when the theme's root size changes, which is a setting the
// restaurant owns.
const SIZE_CLASS: Record<number, string> = {
  [-2]: "text-[0.7rem]",
  [-1]: "text-xs",
  0: "text-base",
  1: "text-lg",
  2: "text-xl",
  3: "text-2xl",
  4: "text-3xl",
  5: "text-4xl",
  6: "text-5xl",
  7: "text-6xl",
  8: "text-7xl",
};

export interface CanvasWidgets {
  /** The functional blocks, handed in rather than imported: this file must not
   *  decide what a menu grid is, and the widgets already exist. */
  [key: string]: React.ReactNode;
}

export default function CanvasBlock({
  canvas,
  lang,
  t,
  widgets,
  popup = false,
}: {
  canvas?: DesignCanvas | null;
  lang: Lang;
  t: Dict;
  widgets?: CanvasWidgets;
  popup?: boolean;
}) {
  const elements = (canvas?.elements ?? []).filter((e) => !e.hidden);
  if (elements.length === 0) return null;

  // Whether anybody drew a phone layout at all. Decided for the band, not per
  // element: a half-drawn phone layout — two elements placed, five falling back
  // to flow — would overlap in a way nobody chose.
  const hasMobile = elements.some((e) => e.mobile);

  const height = canvas?.height ?? 60;
  const mobileHeight = canvas?.heightMobile ?? 0;

  return (
    <>
      {/* ⚠️ Two renderings of the same elements, one hidden by CSS.
          Not a duplicate for its own sake: the desktop version is absolutely
          positioned and the phone version is flow, and no single DOM structure is
          both. Doing it with CSS rather than JavaScript keeps it correct during
          server rendering, which is where this page is drawn. */}
      <div
        className={`relative hidden w-full lg:block ${TONE_CLASS[canvas?.background ?? ""] ?? ""}`}
        style={{ height: `${height}vh` }}
      >
        {elements.map((el, i) => (
          <Element
            key={i}
            el={el}
            box={el.box}
            lang={lang}
            t={t}
            widgets={widgets}
            absolute
          />
        ))}
      </div>

      {hasMobile ? (
        <div
          className={`relative w-full lg:hidden ${TONE_CLASS[canvas?.background ?? ""] ?? ""}`}
          style={{ height: `${mobileHeight || height}vh` }}
        >
          {elements
            .filter((e) => !e.hiddenMobile)
            .map((el, i) => (
              <Element
                key={i}
                el={el}
                box={el.mobile ?? el.box}
                lang={lang}
                t={t}
                widgets={widgets}
                absolute
              />
            ))}
        </div>
      ) : (
        // The fallback: drawn order, full width, generous spacing. A phone
        // showing a squashed desktop composition is the failure this avoids.
        <div
          className={`flex w-full flex-col gap-6 px-4 py-10 lg:hidden ${
            TONE_CLASS[canvas?.background ?? ""] ?? ""
          } ${popup ? "" : ""}`}
        >
          {elements
            .filter((e) => !e.hiddenMobile)
            .sort((a, b) => a.box.y - b.box.y)
            .map((el, i) => (
              <Element
                key={i}
                el={el}
                box={el.box}
                lang={lang}
                t={t}
                widgets={widgets}
                absolute={false}
              />
            ))}
        </div>
      )}
    </>
  );
}

function Element({
  el,
  box,
  lang,
  t,
  widgets,
  absolute,
}: {
  el: DesignElement;
  box: DesignBox;
  lang: Lang;
  t: Dict;
  widgets?: CanvasWidgets;
  absolute: boolean;
}) {
  const style = el.style ?? {};
  const position: React.CSSProperties = absolute
    ? {
        position: "absolute",
        left: `${box.x}%`,
        top: `${box.y}%`,
        width: `${box.w}%`,
        height: `${box.h}%`,
        zIndex: box.z ?? 0,
      }
    : {};

  const classes = [
    TONE_CLASS[style.tone ?? ""] ?? "",
    COLOR_CLASS[style.color ?? ""] ?? "",
    SIZE_CLASS[style.size ?? 0] ?? "",
    style.font === "display" ? "font-display" : "",
    style.weight === "bold" ? "font-bold" : style.weight === "black" ? "font-black" : "",
    style.align === "center" ? "text-center" : "",
    style.rounded ? "overflow-hidden rounded-2xl" : "",
    style.shadow ? "shadow-card" : "",
  ]
    .filter(Boolean)
    .join(" ");

  // ⚠️ Opacity is applied to the element's own surface, never to text: half
  // transparent words are unreadable, and this control exists to dim a photograph
  // behind a headline.
  const opacity = style.opacity != null && style.opacity < 100 ? style.opacity / 100 : undefined;

  if (el.type === "widget-menu" || el.type === "widget-categories" ||
      el.type === "widget-hours" || el.type === "widget-map" || el.type === "widget-cart") {
    return (
      <div style={position} className={`${classes} overflow-auto`}>
        {widgets?.[el.type] ?? null}
      </div>
    );
  }

  if (el.type === "box") {
    return <div style={{ ...position, opacity }} className={classes} />;
  }

  if (el.type === "divider") {
    return (
      <div style={position} className={classes}>
        <hr className="border-line-strong" />
      </div>
    );
  }

  if (el.type === "image") {
    const src = el.image ? imageUrl(el.image, 1200) : "";
    if (!src) return null;
    return (
      <div style={{ ...position, opacity }} className={`relative ${classes}`}>
        <Image
          src={src}
          alt=""
          fill
          // The designer chose the box; the picture fills it. `contain` would
          // leave letterboxing nobody drew.
          className="object-cover"
          sizes="100vw"
        />
      </div>
    );
  }

  const text = localized(el.text, lang);

  if (el.type === "button") {
    if (!text) return null;
    return (
      <div style={position} className={classes}>
        <LocaleLink href={el.link || "/menu"} className="btn btn-primary w-full">
          {text}
        </LocaleLink>
      </div>
    );
  }

  // Text. `whitespace-pre-line` because a designer types line breaks where they
  // want them and a headline reflowed by the browser is a different headline.
  if (!text) return null;
  return (
    <div style={position} className={`${classes} whitespace-pre-line leading-tight`}>
      {text}
    </div>
  );
}

// Kept for the popup, which is the same drawing shown over the page.
export { TONE_CLASS as canvasTones };
