// The classes a freely placed element is drawn with.
//
// ⚠️ **Extracted so that exactly one piece of code decides what an element looks
// like.** The renderer draws it on the server; the console's live preview has to
// re-draw it in the browser when a colour or a size changes, because the
// alternative is re-pointing the iframe at the customer's real site after every
// click — a full navigation, on somebody's shop, to show a heading one step
// larger. A second copy of these maps inside the bridge would be the copy that
// drifts, and the way that drift shows up is a preview that lies.
//
// Everything here is a **static class name**, never a computed one: Tailwind can
// only see class names that appear literally in the source, so a template string
// like `rounded-${side}-${step}` compiles to nothing at all and the page silently
// loses every rounded panel.

import type { DesignBox, DesignElement, DesignSection } from "@/lib/types";

const TONE_CLASS: Record<string, string> = {
  "": "",
  surface: "bg-surface",
  raised: "bg-raised",
  charcoal: "bg-charcoal text-white",
  brand: "bg-brand text-white",
  // ⚠️ The ink comes with the surface and is computed from its luminance
  // (theme-css.ts), never chosen: a yellow panel needs black words and a deep
  // green one needs white. A designer who picked both by hand would get it
  // right for the colour in front of them and wrong for the next customer.
  accent: "bg-accent text-accent-ink",
  ink: "bg-ink text-cream",
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
  accent: "text-accent",
};

// Font size as a step on the type scale rather than a pixel value: a headline
// stays proportional when the theme's root size changes, which is a setting the
// restaurant owns.
/** Radius as a step. `full` is what makes a circular photograph possible, which
 *  three of the starting templates are built around. */
const RADIUS_CLASS: Record<string, string> = {
  "": "",
  sm: "rounded-lg",
  md: "rounded-xl",
  lg: "rounded-3xl",
  xl: "rounded-[2.5rem]",
  "2xl": "rounded-[4rem]",
  full: "rounded-full",
};

/** The same steps, applied to one side only.
 *
 *  ⚠️ **Written out per corner rather than composed**, because Tailwind can
 *  only see class names that appear literally in the source: a template string
 *  like `rounded-${side}-${step}` compiles to nothing at all, and the page
 *  silently loses every rounded panel while the document looks correct.
 *
 *  ⚠️ **This is the shape a shop reference is actually built from** — a
 *  coloured panel rounded on the edge that faces the page, with a category list
 *  on it. With all-four-corners only, the way to approximate it was to push the
 *  box off the side of the band and hope. */
const CORNER_CLASS: Record<string, Record<string, string>> = {
  left: {
    sm: "rounded-l-lg", md: "rounded-l-xl", lg: "rounded-l-3xl",
    xl: "rounded-l-[2.5rem]", "2xl": "rounded-l-[4rem]", full: "rounded-l-full",
  },
  right: {
    sm: "rounded-r-lg", md: "rounded-r-xl", lg: "rounded-r-3xl",
    xl: "rounded-r-[2.5rem]", "2xl": "rounded-r-[4rem]", full: "rounded-r-full",
  },
  top: {
    sm: "rounded-t-lg", md: "rounded-t-xl", lg: "rounded-t-3xl",
    xl: "rounded-t-[2.5rem]", "2xl": "rounded-t-[4rem]", full: "rounded-t-full",
  },
  bottom: {
    sm: "rounded-b-lg", md: "rounded-b-xl", lg: "rounded-b-3xl",
    xl: "rounded-b-[2.5rem]", "2xl": "rounded-b-[4rem]", full: "rounded-b-full",
  },
};

/** A quarter turn, for a rail of words down the edge of the page.
 *
 *  ⚠️ **`origin-center` and nothing else.** A rotated box keeps the width and
 *  height it was drawn with — the turn is painted, not laid out — so the
 *  designer sizes the box for the text lying down and the page turns it in
 *  place. Rotating about a corner instead would move the element somewhere
 *  nobody placed it, which is the failure that makes rotation feel broken. */
const ROTATE_CLASS: Record<string, string> = {
  "": "",
  "90": "rotate-90 origin-center",
  "-90": "-rotate-90 origin-center",
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

/** Which class rounds this element, given the step and the side. */
function radiusClass(radius?: string, corner?: string): string {
  if (!radius) return "";
  const step = corner ? CORNER_CLASS[corner]?.[radius] : RADIUS_CLASS[radius];
  return step ? `overflow-hidden ${step}` : "";
}

/** The class list for one element, from its style bag. */
export function elementClass(el: DesignElement, extra = ""): string {
  const style = el.style ?? {};
  return [
    TONE_CLASS[style.tone ?? ""] ?? "",
    COLOR_CLASS[style.color ?? ""] ?? "",
    SIZE_CLASS[style.size ?? 0] ?? "",
    style.font === "display" ? "font-display" : "",
    style.weight === "bold" ? "font-bold" : style.weight === "black" ? "font-black" : "",
    style.align === "center" ? "text-center" : "",
    style.rounded ? "overflow-hidden rounded-2xl" : "",
    radiusClass(style.radius, style.corner),
    ROTATE_CLASS[style.rotate ?? ""] ?? "",
    style.shadow ? "shadow-card" : "",
    extra,
  ]
    .filter(Boolean)
    .join(" ");
}

/** Where an element sits, when the band places its elements itself. */
export function elementPosition(box: DesignBox, absolute: boolean): React.CSSProperties {
  return absolute
    ? {
        position: "absolute",
        left: `${box.x}%`,
        top: `${box.y}%`,
        width: `${box.w}%`,
        height: `${box.h}%`,
        zIndex: box.z ?? 0,
      }
    : {};
}

/** ⚠️ Opacity belongs to the element's own surface, never to its words: half
 *  transparent text is unreadable, and this control exists to dim a photograph
 *  behind a headline. */
export function elementOpacity(el: DesignElement): number | undefined {
  const o = el.style?.opacity;
  return o != null && o < 100 ? o / 100 : undefined;
}

/** The extra classes each element type adds to the ones above. ⚠️ Listed here
 *  rather than inline in the renderer so the browser-side patch produces the
 *  same class list as the server did — an element that loses `flex` on the
 *  first colour change is an element that jumps. */
export const TYPE_CLASS: Record<string, string> = {
  image: "relative",
  carousel: "flex snap-x snap-mandatory gap-3 overflow-x-auto",
  icon: "flex items-center justify-center",
  badge: "flex items-center",
  quote: "flex flex-col justify-center gap-2",
  rating: "flex items-center gap-1",
  stat: "flex flex-col justify-center",
  list: "space-y-1.5 overflow-auto",
  text: "whitespace-pre-line leading-tight",
  "widget-social": "overflow-auto",
  "widget-menu": "overflow-auto",
  "widget-categories": "overflow-auto",
  "widget-hours": "overflow-auto",
  "widget-map": "overflow-auto",
  "widget-cart": "overflow-auto",
};

export { TONE_CLASS, COLOR_CLASS, SIZE_CLASS, RADIUS_CLASS, ROTATE_CLASS };

/** Column spans, written out because Tailwind cannot see a computed class name.
 *
 *  Interpolating `lg:col-span-${n}` would compile to nothing at all — the class
 *  is not in the source for the compiler to find — and the page would silently
 *  lose every width. The map is the fix and it is also the allowlist: a span
 *  outside 1–12 cannot produce a class. */
const SPAN_CLASS: Record<number, string> = {
  1: "lg:col-span-1",
  2: "lg:col-span-2",
  3: "lg:col-span-3",
  4: "lg:col-span-4",
  5: "lg:col-span-5",
  6: "lg:col-span-6",
  7: "lg:col-span-7",
  8: "lg:col-span-8",
  9: "lg:col-span-9",
  10: "lg:col-span-10",
  11: "lg:col-span-11",
  12: "lg:col-span-12",
};

/** How wide a band's contents may run.
 *
 *  ⚠️ **Applied as CSS variables rather than as a class on the band**, because
 *  the container is not here: every band component draws its own
 *  `container-page` inside itself, and a `max-w-*` on this wrapper would sit
 *  outside it and do nothing. The two variables are the ones `.container-page`
 *  reads (globals.css), so one value set here reaches whichever container the
 *  band happens to use — including the ones inside a band we have not written
 *  yet.
 *
 *  ⚠️ **"full" clears the gutter as well as the cap.** A band asking to run
 *  edge to edge and still holding a 2rem inset is not full width; it is a band
 *  that looks like the setting was ignored, which is worse than not offering
 *  it. */
const WIDTH_VARS: Record<string, React.CSSProperties> = {
  "": {},
  wide: { "--band-max": "96rem" } as React.CSSProperties,
  full: { "--band-max": "100%", "--band-pad": "0px" } as React.CSSProperties,
};

const PAD_CLASS: Record<string, string> = {
  "": "",
  sm: "py-6",
  md: "py-12",
  lg: "py-20",
};

/** The classes one band's wrapper carries. */
export function bandClass(section: DesignSection): string {
  const style = section.style ?? {};
  return [
    SPAN_CLASS[section.span] ?? SPAN_CLASS[12],
    TONE_CLASS[style.tone ?? ""] ?? "",
    PAD_CLASS[style.padding ?? ""] ?? "",
    style.align === "center" ? "text-center" : "",
    style.rounded ? "overflow-hidden rounded-3xl" : "",
  ]
    .filter(Boolean)
    .join(" ");
}

/** The two variables `.container-page` reads, for a band that runs wider than
 *  the page column. ⚠️ Variables rather than a class, because the container is
 *  not here: every band draws its own inside itself. */
export function bandWidthVars(width?: string): React.CSSProperties | undefined {
  return WIDTH_VARS[width ?? ""] ?? undefined;
}
