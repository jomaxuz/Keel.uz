"use client";

import { useMemo } from "react";

import { imageUrl } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { contentName } from "@/lib/i18n/content";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import type { Lang } from "@/lib/i18n/dictionaries";
import { categoryTint } from "@/lib/tillColors";
import type { MenuGroup, MenuItem } from "@/lib/types";

/**
 * The menu, drawn for a monoblock.
 *
 * ⚠️ **This screen is used standing up, at arm's length, by somebody who is
 * also talking.** Every decision below follows from that and from the hardware
 * it runs on: a 15" panel, usually 1024×768, a weak processor and 4 GB of RAM.
 *
 * ⚠️ **Categories never scroll sideways.** A horizontal strip is the wrong
 * control for a touchscreen with no scrollbar and no wheel: the categories past
 * the edge simply do not exist to the person using it, and the ones that do are
 * a moving target. They wrap instead, and all of them are on screen.
 *
 * ⚠️ **Colour by category, images optional.** For a 200-dish menu colour is
 * faster than photographs — the cashier learns that drinks are the blue corner
 * — and it costs no decoding. Photographs are switched on per device (see the
 * toggle in the header) because whether they help depends on the screen and the
 * processor, which are facts about that monoblock rather than about the company.
 */
export default function MenuGrid({
  menu,
  items,
  categoryID,
  onCategory,
  query,
  showImages,
  currency,
  disabled,
  onPick,
}: {
  menu: MenuGroup[];
  items: MenuItem[];
  categoryID: string | null;
  onCategory: (id: string) => void;
  query: string;
  showImages: boolean;
  currency: string;
  /** No check is open, so nothing can be added to anything. */
  disabled: boolean;
  onPick: (item: MenuItem) => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();

  // Which category each dish belongs to, so a search result keeps the colour it
  // has in its own section — the cue has to survive the search or it teaches
  // people not to trust it.
  const catOf = useMemo(() => {
    const m = new Map<string, string>();
    for (const g of menu) for (const it of g.items) m.set(it.id, g.category.id);
    return m;
  }, [menu]);

  // ⚠️ **A search is capped.** Typing two letters can match most of the menu,
  // and rendering two hundred tiles with images on a weak processor is the one
  // thing that visibly freezes this screen. The cap is generous enough that it
  // is almost never reached, and the line below says so rather than silently
  // truncating — a list that quietly hides the dish somebody is looking for is
  // worse than a slow one.
  const capped = query ? items.slice(0, MAX_RESULTS) : items;
  const hidden = items.length - capped.length;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {/* ---- Categories ----

          ⚠️ **A thin colour bar, not a filled pill.** Saturated buttons in a
          row were the single thing that made this screen look like a web page:
          eight full-strength colours across the top leave the eye nowhere to
          rest, and once everything is coloured the colour cannot mean "this
          category" any more. The bar carries the cue, the surface stays
          neutral, and the selected one is the only filled thing in the row. */}
      {!query && (
        <div className="flex shrink-0 flex-wrap gap-2 px-3 pb-3 pt-3">
          {menu.map((g) => {
            const tint = categoryTint(g.category.id);
            const on = g.category.id === categoryID;
            return (
              <button
                key={g.category.id}
                onClick={() => onCategory(g.category.id)}
                className={on ? "till-chip-btn-on" : "till-chip-btn"}
              >
                {/* ⚠️ **A dot, and the same dot when selected.** The colour is
                    the category's identity — it has to be the same mark on the
                    chip and down the side of every one of its dishes, or the
                    cue is two cues. Selection is carried by the fill instead,
                    which is the one thing on the screen that is allowed to say
                    "you are here". Colour-filled chips were the old answer and
                    they used the identity to say the state, so eight
                    full-strength colours competed across the top and none of
                    them meant anything. */}
                <span
                  className="till-seg-dot"
                  style={{ background: tint.bar }}
                  aria-hidden
                />
                {contentName(g.category, lang)}
              </button>
            );
          })}
        </div>
      )}

      {/* ---- Dishes ---- */}
      <div className="min-h-0 flex-1 overflow-y-auto px-3 pb-4">
        {capped.length === 0 && (
          <p className="py-8 text-center text-ink-muted">
            {t.till.nothingFound}
          </p>
        )}
        <div
          // ⚠️ Denser than a website grid and deliberately so: at three
          // columns a 200-dish menu is eight screens of scrolling, and the
          // dish somebody wants is always the one below the fold.
          className={`grid gap-2 ${
            showImages
              ? "grid-cols-3 xl:grid-cols-5 2xl:grid-cols-7"
              : "grid-cols-3 xl:grid-cols-6 2xl:grid-cols-8"
          }`}
        >
          {capped.map((it) => (
            <Tile
              key={it.id}
              item={it}
              tint={categoryTint(catOf.get(it.id) ?? "")}
              showImage={showImages}
              currency={currency}
              disabled={disabled}
              onPick={onPick}
            />
          ))}
        </div>
        {hidden > 0 && (
          <p className="py-4 text-center text-sm text-ink-muted">
            {t.till.moreResults(hidden)}
          </p>
        )}
      </div>
    </div>
  );
}

/** How many search results are drawn before the screen stops and says so. */
const MAX_RESULTS = 60;

function Tile({
  item,
  tint,
  showImage,
  currency,
  disabled,
  onPick,
}: {
  item: MenuItem;
  tint: { background: string; bar: string };
  showImage: boolean;
  currency: string;
  disabled: boolean;
  onPick: (item: MenuItem) => void;
}) {
  const { lang } = useI18n();
  const t = useAdminT();
  // ⚠️ 300 is the smallest size the server will resize to, and the tile is
  // ~180px wide — so this is already the cheapest image available. Asking for
  // the original would put a 4 MB phone photograph on a 4 GB monoblock, forty
  // times over.
  const src = showImage ? imageUrl(item.imageUrl, 300) : null;
  // ⚠️ **Off the menu and off today are the same answer to a finger.** A dish
  // on the branch's stop list stayed pressable and refused only after the tap,
  // with "the dish has run out" — which is the server telling the cashier
  // something the screen already knew, in front of the guest. The tile is the
  // place to say it.
  const off = !item.isAvailable || !!item.soldOut;
  // ⚠️ Marked, because the two taps do different things. One tile adds a dish
  // and the next one opens a question, and a cashier who cannot tell them apart
  // taps twice on a dish that was already waiting for an answer — then finds
  // two lines on the check, or none.
  const asks = (item.options?.length ?? 0) > 0;
  const name = contentName(item, lang);

  return (
    <button
      onClick={() => onPick(item)}
      disabled={disabled || off}
      // ⚠️ **Named by the dish, not by everything printed on it.** Without this
      // the tile's accessible name is the dish, the price and the currency run
      // together — which is what a screen reader says out loud and what any
      // by-name lookup has to match.
      aria-label={off ? `${name} — ${t.till.soldOut}` : name}
      // ⚠️ **A finger, not a cursor.** The tile is thumb-sized even without a
      // photograph — the person pressing it is standing, talking, and not
      // looking at their hand.
      className="till-tile h-[10.5rem] justify-between p-3.5"
    >
      {src ? (
        <>
          {/* ⚠️ The colour strip has to sit above the photograph: the image is
              drawn after it in the DOM and covered it completely, so the cue
              simply did not exist on any tile that had a picture — which is all
              of them. */}
          <span
            className="till-tile-bar"
            style={{ background: tint.bar }}
            aria-hidden
          />
          {/* ⚠️ **The bleed is on this box, and the picture fills it.**
              Two earlier attempts put it on the `<img>` itself and both left a
              pale strip down the right of every card — the left is covered by
              the colour bar, which is why only one side looked wrong.

              An `<img>` is a *replaced* element: with `width: auto` it takes its
              own intrinsic width, so `align-self: stretch` does nothing to it,
              and a `calc(100% + …)` has to agree with the tile's padding and
              border to the pixel. A plain box has neither problem — it stretches
              because it is not replaced, and the photograph inside is told to
              fill it. Changing `p-3.5` now moves both together. */}
          <span className="-mx-3.5 -mt-3.5 block h-[4.5rem] w-auto self-stretch overflow-hidden">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={src}
              alt=""
              // Lazy and fixed-height: the browser decodes only what is on
              // screen, and a tile cannot grow because a photograph is portrait.
              loading="lazy"
              decoding="async"
              className="h-full w-full object-cover"
            />
          </span>
        </>
      ) : (
        // ⚠️ **Colour as a place, not as decoration.** Without photographs the
        // grid is 200 identical rectangles; the tinted square gives each
        // category a corner of the screen the hand learns, which is why a
        // colour-only till can be quicker than one with pictures.
        <span
          className="flex h-9 w-9 items-center justify-center rounded-[9px] text-[15px] font-bold"
          style={{ background: tint.background, color: tint.bar }}
          aria-hidden
        >
          {name.slice(0, 1).toUpperCase()}
        </span>
      )}

      {/* ⚠️ A corner dot, not a superscript one beside the name. Inline it read
          as punctuation after the dish, and on a two-line name it landed
          wherever the wrap put it. */}
      {asks && (
        <span
          aria-hidden
          className="absolute right-2.5 top-2.5 z-10 h-2.5 w-2.5 rounded-full ring-2 ring-white"
          style={{ background: "rgb(var(--till-accent))" }}
        />
      )}

      <span className="line-clamp-2 text-[15px] font-semibold leading-snug">
        {name}
      </span>

      {off ? (
        <span className="till-chip till-chip-late self-start">
          {t.till.soldOut}
        </span>
      ) : (
        // ⚠️ The unit sits **beside** the number, not at the other end of the
        // tile. Pushed apart they read as two facts — a price and a stray word
        // — and on a 1024px monoblock the gap between them is wider than the
        // number itself. Quieter, yes; separated, no.
        <span className="flex items-baseline gap-1">
          <span className="till-num text-[17px] font-semibold">
            {formatPrice(item.price, currency, lang).replace(/\s*\S+$/, "")}
          </span>
          <span className="text-[12px] text-[rgb(var(--till-dim))]">
            {currencyWord(currency, lang)}
          </span>
        </span>
      )}
    </button>
  );
}

/** The word after the number, split off so the figure can carry its own weight.
 *
 *  ⚠️ Taken from `formatPrice` rather than hard-coded: a restaurant billing in
 *  something other than so'm exists, and "12 000 so'm" printed under a dollar
 *  price is worse than no unit at all. */
function currencyWord(currency: string, lang: Lang): string {
  const parts = formatPrice(0, currency, lang).split(/\s+/);
  return parts[parts.length - 1] ?? "";
}
