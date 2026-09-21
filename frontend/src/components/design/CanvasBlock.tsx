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
import {
  TONE_CLASS,
  elementClass,
  elementOpacity,
  elementPosition,
  imageFit,
  TYPE_CLASS,
} from "./canvasStyle";
import { imageUrl } from "@/lib/api";
import { localized } from "@/lib/i18n/site-content";
import type { Lang } from "@/lib/i18n/dictionaries";
import type { DesignBox, DesignCanvas, DesignElement } from "@/lib/types";








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
      {/* ⚠️ **`overflow-x-clip`, and it is not tidiness.** A box may be placed
          past the edge of the band on purpose — `sanitizeBox` allows -50% to
          150% because a shape bleeding off the side is a real technique, and
          the shop templates are built on one. Absolutely positioned, that
          overflow widened the document: the whole site got a horizontal
          scrollbar and could be dragged sideways into empty page. Clipped on
          the x axis only, so a bleed still reads as a bleed and an element that
          overhangs the band vertically is untouched. `clip` rather than
          `hidden` because `hidden` makes this a scroll container, which breaks
          `position: sticky` on anything above it — the header. */}
      <div
        data-keel-band={bandIndex}
        className={`relative hidden w-full overflow-x-clip lg:block ${TONE_CLASS[canvas?.background ?? ""] ?? ""}`}
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
          className={`relative w-full overflow-x-clip lg:hidden ${TONE_CLASS[canvas?.background ?? ""] ?? ""}`}
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
          // ⚠️ Not `data-keel-band`: this surface is flow, so it must not be
          // reported as a draggable band. It is marked all the same, because a
          // word changed in the console has to change here too — see
          // PreviewBridge's `nodesOf`.
          data-keel-flow={bandIndex}
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
  // ⚠️ Both come from `canvasStyle`, which the console's live preview also
  // reads: what an element looks like is decided in one place, so a colour
  // changed in the editor can be drawn into the page without reloading it and
  // still match what the server would have rendered.
  const position = elementPosition(box, absolute);
  const classes = elementClass(el);
  const opacity = elementOpacity(el);

  if (el.type === "widget-social" || el.type === "widget-menu" || el.type === "widget-categories" ||
      el.type === "widget-hours" || el.type === "widget-map" || el.type === "widget-cart") {
    return (
      <div style={position} {...mark} className={`${classes} ${TYPE_CLASS[el.type] ?? ""}`}>
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
      <div style={{ ...position, opacity }} {...mark} className={`${classes} ${TYPE_CLASS.image}`}>
        <img
          src={src}
          alt=""
          // The designer chose the box, and by default the picture fills it —
          // `contain` is what a cut-out asks for, and it is a choice now rather
          // than an assumption. See `imageFit`.
          className={`absolute inset-0 h-full w-full ${imageFit(el)}`}
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
      <div style={position} {...mark} className={`${classes} ${TYPE_CLASS.carousel}`}>
        {shots.map((src, i) => (
          <div key={i} className="relative h-full w-full shrink-0 snap-center overflow-hidden rounded-2xl">
            <img src={imageUrl(src, 1200) ?? ""} alt="" className={`absolute inset-0 h-full w-full ${imageFit(el)}`} />
          </div>
        ))}
      </div>
    );
  }

  if (el.type === "icon") {
    return (
      <div style={position} {...mark} className={`${classes} ${TYPE_CLASS.icon}`}>
        <Icon name={el.icon ?? "star"} />
      </div>
    );
  }

  if (el.type === "badge") {
    if (!text) return null;
    return (
      <div style={position} {...mark} className={`${classes} ${TYPE_CLASS.badge}`}>
        <span className="badge-brand" data-keel-text="plain">{text}</span>
      </div>
    );
  }

  if (el.type === "quote") {
    if (!text) return null;
    return (
      <div style={position} {...mark} className={`${classes} ${TYPE_CLASS.quote}`}>
        <p className="font-display italic leading-snug" data-keel-text="quote">
          {"\u201c" + text + "\u201d"}
        </p>
        {subtext && (
          <p className="text-sm text-ink-muted" data-keel-sub="dash">
            {"\u2014 " + subtext}
          </p>
        )}
      </div>
    );
  }

  if (el.type === "rating") {
    const filled = Math.max(0, Math.min(5, el.value ?? 5));
    return (
      <div style={position} {...mark} className={`${classes} ${TYPE_CLASS.rating}`}>
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
      <div style={position} {...mark} className={`${classes} ${TYPE_CLASS.stat}`}>
        <span
          className="font-display text-4xl font-black leading-none sm:text-5xl"
          data-keel-text="plain"
        >
          {text}
        </span>
        {subtext && (
          <span className="mt-1 text-sm text-ink-muted" data-keel-sub="plain">
            {subtext}
          </span>
        )}
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
      <ul
        style={position}
        {...mark}
        data-keel-text="lines"
        className={`${classes} ${TYPE_CLASS.list}`}
      >
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
    const href = el.link || "/menu";
    // ⚠️ **Two elements, because a button may now leave the site.**
    // `LocaleLink` prefixes `/ru` or `/en` onto an href — right for a page of
    // ours and nonsense on `https://t.me/...` — and Next's client router
    // cannot navigate to another origin at all. `noreferrer` goes with
    // `target="_blank"`: a tab opened without it can reach back through
    // `window.opener`.
    const outside = el.linkExternal || !href.startsWith("/");
    return (
      <div style={position} {...mark} className={classes}>
        {outside ? (
          <a
            href={href}
            target="_blank"
            rel="noreferrer noopener"
            className="btn btn-primary w-full"
          >
            <span data-keel-text="plain">{text}</span>
          </a>
        ) : (
          <LocaleLink href={href} className="btn btn-primary w-full">
            <span data-keel-text="plain">{text}</span>
          </LocaleLink>
        )}
      </div>
    );
  }

  // Text. `whitespace-pre-line` because a designer types line breaks where they
  // want them and a headline reflowed by the browser is a different headline.
  if (!text) return null;
  // ⚠️ **A line of text may be a link, and a shop's category rail is why.**
  // The list down the side of a lookbook hero — Office Wear, Party, Casual — is
  // plain text that goes somewhere, and the only element that could go
  // somewhere was `button`, which draws a filled accent rectangle. Four of
  // those stacked is not that design; it is four buttons. Drawing them as text
  // and leaving them dead is worse — it is a menu the guest cannot use.
  const linked = sanitizedHref(el.link);
  // ⚠️ The same span whether or not the line is a link, and that is what makes
  // it findable. The unlinked branch used to render the bare string, so the one
  // element people type into most had no node to write a changed word into —
  // and the console had to reload the page to show it.
  const body = (
    <span className="whitespace-pre-line leading-tight" data-keel-text="plain">
      {text}
    </span>
  );
  return (
    <div style={position} {...mark} className={`${classes} ${TYPE_CLASS.text}`}>
      {linked ? (
        el.linkExternal || !linked.startsWith("/") ? (
          <a href={linked} target="_blank" rel="noreferrer noopener" className="hover:opacity-70">
            {body}
          </a>
        ) : (
          <LocaleLink href={linked} className="hover:opacity-70">
            {body}
          </LocaleLink>
        )
      ) : (
        body
      )}
    </div>
  );
}

/** The address, if there is one a browser can follow.
 *
 *  ⚠️ A second check on the client although the server already cleaned it. The
 *  document is sanitised on every read, so this is belt and braces rather than
 *  the boundary — but this value becomes an `href` on a public page, and the
 *  one place a check like that is worth repeating is the place it is used. */
function sanitizedHref(raw?: string): string {
  const h = (raw ?? "").trim();
  if (!h) return "";
  if (h.startsWith("//")) return "";
  if (h.startsWith("/") || h.startsWith("https://") || h.startsWith("http://")) return h;
  return "";
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
    // ⚠️ The shop set. The twelve above were drawn for a restaurant — a chef's
    // hat, a chilli, a leaf — and a clothes shop composing a lookbook hero
    // needs none of them and four that were missing. Without these a designer
    // draws a rectangle and types a character into it, which is how a page ends
    // up with a lookbook button nobody recognises as one.
    play: "M8 5.5v13l11-6.5-11-6.5Z",
    search: "M11 19a8 8 0 1 0 0-16 8 8 0 0 0 0 16ZM21 21l-4.3-4.3",
    user: "M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM4 21a8 8 0 0 1 16 0",
    bag: "M6 7h12l1 13H5L6 7ZM9 7V5a3 3 0 0 1 6 0v2",
    "arrow-up": "M12 19V5M6 11l6-6 6 6",
    "arrow-down": "M12 5v14M6 13l6 6 6-6",
    "arrow-right": "M5 12h14M13 6l6 6-6 6",
    "arrow-left": "M19 12H5M11 18l-6-6 6-6",
    plus: "M12 5v14M5 12h14",
    minus: "M5 12h14",
  };
  // ⚠️ Two of them are solid shapes rather than outlines. A play triangle drawn
  // as a 1.6px stroke reads as a chevron at the size this element is used at,
  // and a star with no fill is an empty star — which on a rating means
  // something else entirely.
  const solid = name === "play" || name === "star";
  return (
    <svg
      viewBox="0 0 24 24"
      fill={solid ? "currentColor" : "none"}
      stroke="currentColor"
      strokeWidth={solid ? "0" : "1.6"}
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
