"use client";

import { useMemo } from "react";

import { imageUrl } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { contentName } from "@/lib/i18n/content";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
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
        <div className="flex shrink-0 flex-wrap gap-1.5 px-2.5 pb-2">
          {menu.map((g) => {
            const tint = categoryTint(g.category.id);
            const on = g.category.id === categoryID;
            return (
              <button
                key={g.category.id}
                onClick={() => onCategory(g.category.id)}
                style={
                  on
                    ? { background: tint.bar, borderColor: tint.bar }
                    : { borderColor: tint.bar }
                }
                className={`relative min-h-11 overflow-hidden rounded-[10px] border-b-[3px] px-3.5 text-sm font-semibold transition ${
                  on
                    ? "text-white"
                    : "border-x-0 border-t-0 bg-surface text-ink-soft hover:bg-ink/[0.04]"
                }`}
              >
                {contentName(g.category, lang)}
              </button>
            );
          })}
        </div>
      )}

      {/* ---- Dishes ---- */}
      <div className="min-h-0 flex-1 overflow-y-auto px-2.5 pb-3">
        {capped.length === 0 && (
          <p className="py-8 text-center text-ink-muted">
            {t.till.nothingFound}
          </p>
        )}
        <div
          // ⚠️ Denser than a website grid and deliberately so: at three
          // columns a 200-dish menu is eight screens of scrolling, and the
          // dish somebody wants is always the one below the fold.
          className={`grid gap-1.5 ${
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
  const off = !item.isAvailable;
  // ⚠️ Marked, because the two taps do different things. One tile adds a dish
  // and the next one opens a question, and a cashier who cannot tell them apart
  // taps twice on a dish that was already waiting for an answer — then finds
  // two lines on the check, or none.
  const asks = (item.options?.length ?? 0) > 0;

  return (
    <button
      onClick={() => onPick(item)}
      disabled={disabled || off}
      // ⚠️ **A finger, not a cursor.** The tile is thumb-sized even without a
      // photograph — the person pressing it is standing, talking, and not
      // looking at their hand.
      //
      // ⚠️ **The surface is neutral and the bar carries the colour.** A tinted
      // tile was the thing that made this screen unreadable: the tint is one
      // set of colours, the theme is two, and on the dark one every tile came
      // out a muddy brown with the dish name and the price sunk into it. The
      // cue survives at full strength on the bar, where nothing has to be read
      // on top of it — the same rule the category row follows.
      className="relative flex min-h-[4.5rem] flex-col overflow-hidden rounded-[10px] border border-line bg-surface text-left transition hover:bg-ink/[0.03] active:scale-[0.97] disabled:opacity-40"
    >
      {/* ⚠️ Above the photograph, not under it. The image is drawn after this
          in the DOM and covered the bar completely — the colour cue simply did
          not exist on any tile that had a picture, which is all of them. */}
      <span
        aria-hidden
        className="absolute inset-y-0 left-0 z-10 w-1.5"
        style={{ background: tint.bar }}
      />
      {src && (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={src}
          alt=""
          // ⚠️ Lazy and fixed-height: the browser decodes only what is on
          // screen, and a tile cannot grow because a photograph is portrait.
          loading="lazy"
          decoding="async"
          className="h-20 w-full object-cover"
        />
      )}
      <span className="flex flex-1 flex-col justify-between px-2 py-1.5 pl-2.5">
        <span className="line-clamp-2 text-[13px] font-semibold leading-tight">
          {contentName(item, lang)}
          {/* A dot, not a word: the tile is the densest thing on the screen and
              the label would push the dish name onto a third line. */}
          {asks && <span className="align-super text-brand"> •</span>}
        </span>
        {/* ⚠️ The price is quieter than the name. The cashier is finding a
            dish, not shopping — and a column of bold prices is a column that
            hides the words you are actually scanning. */}
        <span className="mt-0.5 text-xs font-medium tabular-nums text-ink-muted">
          {off ? t.till.soldOut : formatPrice(item.price, currency, lang)}
        </span>
      </span>
    </button>
  );
}
