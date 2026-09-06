// Which rows of the panel's navigation this business sees, in which order, and
// under what name.
//
// ⚠️ **Its own file because it is a rule, not a layout.** These three questions
// used to be answered inside the sidebar component, where the only way to check
// an answer was to log in as that kind of business and look — and the answer
// that was wrong was wrong for every chemist and clothes shop on the platform
// at once. A rule with a test beside it is a rule somebody can change.
//
// ⚠️ **Presentation, never permission.** Hiding a row is a courtesy; the server
// decides who may open a page (handlers/panelgate.go), and a shop that types the
// address of a screen this file does not draw still finds the screen.

import { composes, hasKitchen, hasTables, sellsGoods } from "@/lib/types";
import type { AdminDict } from "@/lib/i18n/admin";

export type BrandLike = { businessType?: string } | null | undefined;

/** What a section needs the business to be, when it needs anything.
 *
 *  ⚠️ **A brand new to this reads as a restaurant**, so a panel that has not
 *  loaded its brand yet — or one whose brand predates the field, which is every
 *  brand today — shows exactly what it showed before. */
export type Needs = "tables" | "kitchen" | "composes" | "goods";

export function needsMet(need: Needs | undefined, brand: BrandLike): boolean {
  switch (need) {
    case "tables":
      return hasTables(brand);
    case "kitchen":
      return hasKitchen(brand);
    case "composes":
      return composes(brand);
    case "goods":
      return sellsGoods(brand);
    default:
      return true;
  }
}

/** What a row is called in this business.
 *
 *  ⚠️ **A pharmacy's stock list is not called "ingredients".** The screen is the
 *  same one — a shop's goods are stock rows exactly as a kitchen's are — but the
 *  word is a kitchen's, and it was sitting in the sidebar of every chemist and
 *  clothes shop on the platform. A person reading "Masalliqlar" over a shelf of
 *  paracetamol does not conclude that the word is loose; they conclude the panel
 *  was built for somebody else.
 *
 *  ⚠️ **A label, never a different page.** Two screens would be two things to
 *  keep in step, and the difference between a shop's stock row and a kitchen's
 *  is a word.
 *
 *  ⚠️ Anything not named here keeps the one name it has: "Kirim", "Chiqim" and
 *  "Inventarizatsiya" are warehouse words, not kitchen ones, and translating
 *  them twice would be two dictionaries drifting for no gain.
 */
export function navLabel(key: string, t: AdminDict, brand: BrandLike): string {
  const words = t.nav as unknown as Record<string, string>;
  if (key === "ingredients" && sellsGoods(brand)) return words.goodsList;
  return words[key];
}

/** The store section, in the order a **shop** walks through it.
 *
 *  ⚠️ **A shop's store is not a kitchen's, and the order says which.** The list
 *  in NAV_GROUPS is written in a kitchen's order — what is on the shelf, what to
 *  buy, what it costs, what goes into a portion — and a grocery reading it top
 *  to bottom finds its three most-used screens (the price labels it reprints
 *  every week, the marked bottles it scans off a delivery, the dates it is
 *  inspected on) at the very bottom, under four documents it touches monthly.
 *  Nothing was *wrong* there; it was simply somebody else's morning.
 *
 *  ⚠️ **An order, not a second list of pages.** Whether a row exists is decided
 *  once, by `needs`, where the reason lives beside it — a second membership list
 *  here would be a place for the two to disagree, and the disagreement would be
 *  a screen that vanished from a sidebar with no explanation anywhere. Anything
 *  this list does not name keeps its place after the ones it does. */
export const SHOP_STOCK_ORDER = [
  // What is on the shelf, and what to reorder.
  "stock",
  "shopping",
  // The catalogue behind the shelf, and where it comes from.
  "ingredients",
  // ⚠️ **A florist is a shop that composes**, and its card is the most literal
  // one in the product: a bouquet is fifteen stems, a wrap and a ribbon. Left
  // out of this list its two screens fell to the bottom, under the monthly
  // paperwork — which is the same complaint this order exists to answer, made
  // by the one shop that needs them. A grocery and a chemist never see them:
  // `needs: "composes"` decides that, one place up.
  "techCards",
  "purchases",
  "suppliers",
  // The three a shop opens weekly and a kitchen almost never does.
  "labels",
  "marking",
  "expiring",
  // What the back room made today — a florist's fifty bouquets for the eighth
  // of March, which is the same document as a kitchen's pot of sauce.
  "production",
  // The corrections, which every business does and none does daily.
  "writeoffs",
  "transfers",
  "stocktake",
];

/** Puts one group in the order this business reads it in.
 *
 *  ⚠️ **Stable**, so a row this list has never heard of — one added later, or
 *  one belonging to another group — keeps the position it was written in rather
 *  than jumping to the front. */
export function orderedFor<T extends { key: string }>(
  group: { key: string; items: readonly T[] },
  brand: BrandLike,
): readonly T[] {
  if (group.key !== "stock" || !sellsGoods(brand)) return group.items;
  const place = (k: string) => {
    const i = SHOP_STOCK_ORDER.indexOf(k);
    return i < 0 ? SHOP_STOCK_ORDER.length : i;
  };
  return [...group.items].sort((a, b) => place(a.key) - place(b.key));
}

