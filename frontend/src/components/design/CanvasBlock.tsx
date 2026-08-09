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
// ⚠️ **A server component, deliberately, and it was a client one for an hour.**
//
// It needs no state — it is pure rendering — and being a client component meant
// the page had to hand it the dictionary, which contains functions
// (`dishes(n)`, `priceFrom(price)`). React refuses to serialise those across the
// boundary, and the way it refuses is the worst part: the band came back as a
// Suspense placeholder that never resolved, so the drawn layout simply **was not
// there**, while every log line was an opaque digest. The site looked like it had
// ignored the design.
//
//   • **Functional widgets keep their own insides.** A menu placed here is
//     positioned and sized freely and still renders the real menu, with prices,
//     translations, options and a working cart. A "free" editor that made people
//     rebuild those out of text and rectangles would produce a page that cannot
//     take an order.

import LocaleLink from "@/components/site/LocaleLink";
import { imageUrl } from "@/lib/api";
import { localized } from "@/lib/i18n/site-content";
import type { Lang } from "@/lib/i18n/dictionaries";
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
/** Radius as a step. `full` is what makes a circular photograph possible, which
 *  three of the starting templates are built around. */
const RADIUS_CLASS: Record<string, string> = {
  "": "",
  md: "rounded-xl",
  lg: "rounded-3xl",
  full: "rounded-full",
};

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
  bandIndex = 0,
  lang,
  widgets,
  popup = false,
}: {
  canvas?: DesignCanvas | null;
  /** Which band this is, published for the console's editing overlay. */
  bandIndex?: number;
  /** Only the language code crosses into this component — never the dictionary.
   *  See the note above on why. */
  lang: Lang;
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
        data-keel-band={bandIndex}
        className={`relative hidden w-full lg:block ${TONE_CLASS[canvas?.background ?? ""] ?? ""}`}
        style={{ height: `${height}vh` }}
      >
        <BandImage canvas={canvas} />
        {elements.map((el, i) => (
          <Element
            key={i}
            el={el}
            box={el.box}
            index={indexOf(canvas, el)}
            lang={lang}
            widgets={widgets}
            absolute
          />
        ))}
      </div>

      {hasMobile ? (
        <div
          data-keel-band={bandIndex}
          className={`relative w-full lg:hidden ${TONE_CLASS[canvas?.background ?? ""] ?? ""}`}
          style={{ height: `${mobileHeight || height}vh` }}
        >
          <BandImage canvas={canvas} />
          {elements
            .filter((e) => !e.hiddenMobile)
            .map((el, i) => (
              <Element
                key={i}
                el={el}
                box={el.mobile ?? el.box}
                index={indexOf(canvas, el)}
                lang={lang}
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
                index={indexOf(canvas, el)}
                lang={lang}
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
  index,
  lang,
  widgets,
  absolute,
}: {
  el: DesignElement;
  box: DesignBox;
  /** ⚠️ The element's index **in the stored canvas**, not in the filtered list
   *  being rendered. It is published as a data attribute so the console's editing
   *  overlay can match a box on screen to the element in the document it is
   *  editing — and a hidden element shifting the numbering would make the overlay
   *  move the wrong thing. */
  index: number;
  lang: Lang;
  widgets?: CanvasWidgets;
  absolute: boolean;
}) {
  // Published on every element, in every branch, so the overlay does not have to
  // know which kind it is looking at.
  const mark = { "data-keel-el": index } as Record<string, unknown>;
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
    style.radius ? `overflow-hidden ${RADIUS_CLASS[style.radius] ?? ""}` : "",
    style.shadow ? "shadow-card" : "",
  ]
    .filter(Boolean)
    .join(" ");

  // ⚠️ Opacity is applied to the element's own surface, never to text: half
  // transparent words are unreadable, and this control exists to dim a photograph
  // behind a headline.
  const opacity = style.opacity != null && style.opacity < 100 ? style.opacity / 100 : undefined;

  if (el.type === "widget-social" || el.type === "widget-menu" || el.type === "widget-categories" ||
      el.type === "widget-hours" || el.type === "widget-map" || el.type === "widget-cart") {
    return (
      <div style={position} {...mark} className={`${classes} overflow-auto`}>
        {widgets?.[el.type] ?? null}
      </div>
    );
  }

  if (el.type === "box") {
    return <div style={{ ...position, opacity }} {...mark} className={classes} />;
  }

  if (el.type === "divider") {
    return (
      <div style={position} {...mark} className={classes}>
        <hr className="border-line-strong" />
      </div>
    );
  }

  if (el.type === "image") {
    const src = el.image ? imageUrl(el.image, 1200) : "";
    if (!src) return null;
    return (
      <div style={{ ...position, opacity }} {...mark} className={`relative ${classes}`}>
        <img
          src={src}
          alt=""
          // The designer chose the box; the picture fills it. `contain` would
          // leave letterboxing nobody drew.
          className="absolute inset-0 h-full w-full object-cover"
        />
      </div>
    );
  }

  const text = localized(el.text, lang);
  const subtext = localized(el.subtext, lang);

  // ⚠️ A scroll-snap strip rather than a scripted slider.
  //
  // A carousel is the element restaurants ask for most, and the usual answer is a
  // JavaScript library — on the site whose page weight was the thing worth fixing.
  // CSS scroll-snap gives the same gesture with no script at all, works before
  // hydration, and is what a phone already does natively. The trade is no
  // auto-advance, which is the part guests dislike anyway.
  if (el.type === "carousel") {
    const shots = (el.images ?? []).filter(Boolean);
    if (shots.length === 0) return null;
    return (
      <div style={position} {...mark} className={`${classes} flex snap-x snap-mandatory gap-3 overflow-x-auto`}>
        {shots.map((src, i) => (
          <div key={i} className="relative h-full w-full shrink-0 snap-center overflow-hidden rounded-2xl">
            <img src={imageUrl(src, 1200) ?? ""} alt="" className="absolute inset-0 h-full w-full object-cover" />
          </div>
        ))}
      </div>
    );
  }

  if (el.type === "icon") {
    return (
      <div style={position} {...mark} className={`${classes} flex items-center justify-center`}>
        <Icon name={el.icon ?? "star"} />
      </div>
    );
  }

  if (el.type === "badge") {
    if (!text) return null;
    return (
      <div style={position} {...mark} className={`${classes} flex items-center`}>
        <span className="badge-brand">{text}</span>
      </div>
    );
  }

  if (el.type === "quote") {
    if (!text) return null;
    return (
      <div style={position} {...mark} className={`${classes} flex flex-col justify-center gap-2`}>
        <p className="font-display italic leading-snug">“{text}”</p>
        {subtext && <p className="text-sm text-ink-muted">— {subtext}</p>}
      </div>
    );
  }

  if (el.type === "rating") {
    const filled = Math.max(0, Math.min(5, el.value ?? 5));
    return (
      <div style={position} {...mark} className={`${classes} flex items-center gap-1`}>
        {[1, 2, 3, 4, 5].map((i) => (
          <svg key={i} viewBox="0 0 24 24" className="h-full w-auto max-h-8" aria-hidden
            fill={i <= filled ? "currentColor" : "none"} stroke="currentColor" strokeWidth="1.5">
            <path d="M12 3l2.9 5.9 6.5.9-4.7 4.6 1.1 6.5L12 18l-5.8 3 1.1-6.5L2.6 9.8l6.5-.9L12 3Z" />
          </svg>
        ))}
      </div>
    );
  }

  // A number and what it counts. Two fields, because a stat with no label is a
  // number nobody can interpret.
  if (el.type === "stat") {
    if (!text) return null;
    return (
      <div style={position} {...mark} className={`${classes} flex flex-col justify-center`}>
        <span className="font-display text-4xl font-black leading-none sm:text-5xl">{text}</span>
        {subtext && <span className="mt-1 text-sm text-ink-muted">{subtext}</span>}
      </div>
    );
  }

  // One line per typed line. The lines come from the text field rather than from a
  // repeating editor: typing three lines is faster than adding three rows, and this
  // is content a designer writes in one go.
  if (el.type === "list") {
    const lines = text.split("\n").map((l) => l.trim()).filter(Boolean);
    if (lines.length === 0) return null;
    return (
      <ul style={position} {...mark} className={`${classes} space-y-1.5 overflow-auto`}>
        {lines.map((line, i) => (
          <li key={i} className="flex gap-2">
            <span className="mt-1 h-1.5 w-1.5 shrink-0 rounded-full bg-brand" />
            <span>{line}</span>
          </li>
        ))}
      </ul>
    );
  }

  if (el.type === "button") {
    if (!text) return null;
    return (
      <div style={position} {...mark} className={classes}>
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
    <div style={position} {...mark} className={`${classes} whitespace-pre-line leading-tight`}>
      {text}
    </div>
  );
}

/** The photograph behind a whole band.
 *
 *  ⚠️ Its own component so both surfaces (desktop and phone) get it from one place.
 *  `object-cover` and not `contain`: a photo hero is a photo that fills the band,
 *  and letterboxing is never what somebody drew. */
function BandImage({ canvas }: { canvas?: DesignCanvas | null }) {
  if (!canvas?.image) return null;
  const opacity =
    canvas.backgroundOpacity != null && canvas.backgroundOpacity < 100
      ? canvas.backgroundOpacity / 100
      : undefined;
  return (
    <div className="absolute inset-0" style={{ opacity }}>
      <img
        src={imageUrl(canvas.image, 1200) ?? ""}
        alt=""
        className="absolute inset-0 h-full w-full object-cover"
      />
    </div>
  );
}

/** An element's index in the stored canvas, by identity.
 *
 *  ⚠️ Not the index in the list being rendered: hidden elements are filtered out
 *  first, so the two disagree exactly when something is hidden — and the console's
 *  overlay would then move a different element than the one being dragged. */
function indexOf(canvas: DesignCanvas | null | undefined, el: DesignElement): number {
  return (canvas?.elements ?? []).indexOf(el);
}

/** The fixed icon set. Inline paths, so there is no icon font to load and no
 *  request to make — the same reason the flags are SVG. */
function Icon({ name }: { name: string }) {
  const paths: Record<string, string> = {
    star: "M12 3l2.9 5.9 6.5.9-4.7 4.6 1.1 6.5L12 18l-5.8 3 1.1-6.5L2.6 9.8l6.5-.9L12 3Z",
    clock: "M12 7v5l3 2M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z",
    phone: "M22 16.9v3a2 2 0 0 1-2.2 2 19.8 19.8 0 0 1-8.6-3.1 19.5 19.5 0 0 1-6-6A19.8 19.8 0 0 1 2.1 4.2 2 2 0 0 1 4.1 2h3a2 2 0 0 1 2 1.7c.1 1 .4 1.9.7 2.8a2 2 0 0 1-.5 2.1L8.1 9.9a16 16 0 0 0 6 6l1.3-1.2a2 2 0 0 1 2.1-.5c.9.3 1.8.6 2.8.7a2 2 0 0 1 1.7 2Z",
    pin: "M12 22s7-5.6 7-12a7 7 0 1 0-14 0c0 6.4 7 12 7 12ZM12 11a2 2 0 1 0 0-4 2 2 0 0 0 0 4Z",
    fire: "M12 22c4 0 7-2.7 7-6.5 0-4.5-4.5-6.5-4-11.5-3 1.5-5 4-5 7 0-1.5-1-2.5-2-3-.7 1.5-3 3.4-3 7.5C5 19.3 8 22 12 22Z",
    leaf: "M20 4C10 4 4 9 4 17v3M20 4c0 8-5 12-12 12",
    truck: "M3 7h11v9H3zM14 10h4l3 3v3h-7zM7 19a2 2 0 1 0 0-4 2 2 0 0 0 0 4Zm10 0a2 2 0 1 0 0-4 2 2 0 0 0 0 4Z",
    check: "m5 13 4 4L19 7",
    heart: "M12 21s-8-4.8-8-10a4.5 4.5 0 0 1 8-2.8A4.5 4.5 0 0 1 20 11c0 5.2-8 10-8 10Z",
    cart: "M3 4h2l2.4 11h10.2L20 7H6M9 20a1 1 0 1 0 0-2 1 1 0 0 0 0 2Zm8 0a1 1 0 1 0 0-2 1 1 0 0 0 0 2Z",
    chef: "M7 21h10M6 17h12v-2a6 6 0 0 0-12 0v2Z",
  };
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="h-full w-auto"
      aria-hidden
    >
      <path d={paths[name] ?? paths.star} />
    </svg>
  );
}

// Kept for the popup, which is the same drawing shown over the page.
export { TONE_CLASS as canvasTones };
