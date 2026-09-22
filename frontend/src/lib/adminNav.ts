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

export type BrandLike =
  | { businessType?: string; hasMarked?: boolean; hasGoods?: boolean }
  | null
  | undefined;

/** What a section needs the business to be, when it needs anything.
 *
 *  ⚠️ **A brand new to this reads as a restaurant**, so a panel that has not
 *  loaded its brand yet — or one whose brand predates the field, which is every
 *  brand today — shows exactly what it showed before. */
export type Needs =
  | "tables"
  | "kitchen"
  | "composes"
  | "goods"
  | "marked"
  | "labelled";

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
    // ⚠️ **The type, or the goods themselves.** Asking only what kind of
    // business this is got two real cases wrong, and both are restaurants: a
    // bar receives marked bottles, and a bakery sells packaged bread off a
    // shelf. Both were told the screen did not exist — the page was never
    // gated, but a screen you have never seen is one you do not type the
    // address of.
    //
    // ⚠️ **Still not shown to a kitchen that has neither**, which is the half
    // worth keeping: a row every cook reads past forever is what the
    // type-only rule was protecting against. Nobody has to find a setting —
    // flagging a drink as marked in the menu is what turns the row on.
    case "marked":
      return sellsGoods(brand) || brand?.hasMarked === true;
    case "labelled":
      return sellsGoods(brand) || brand?.hasGoods === true;
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
  if (!sellsGoods(brand)) return words[key];
  if (key === "ingredients") return words.goodsList;
  // ⚠️ **And the catalogue itself.** The row a shop opens most often was still
  // called "Menyu" — the same word the guest's own navigation bar stopped
  // saying when `/catalog` was added, left behind on the screen where the owner
  // edits what the guest reads.
  if (key === "menu") return t.goods.catalog;
  return words[key];
}

/** What a sidebar *group* heading is called.
 *
 *  ⚠️ The heading and the row under it carry the same word, and fixing one and
 *  not the other leaves "Menyu › Katalog" in a shop's sidebar — which reads as
 *  two different sections rather than one renamed. */
export function navGroupLabel(
  key: string,
  t: AdminDict,
  brand: BrandLike,
): string {
  const groups = t.nav.groups as unknown as Record<string, string>;
  if (key === "menu" && sellsGoods(brand)) return t.goods.catalog;
  return groups[key];
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
  "dispatch",
  "stocktake",
  // And what the count found. ⚠️ After it for the same reason it is after it
  // everywhere: a shortfall is a count's second half, and a shop reads the two
  // in that order.
  "shortages",
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


// ---- The settings page's own tabs ----
//
// ⚠️ **The same complaint, one screen along.** The tabs are named and ordered
// for a restaurant: a chemist opens Settings and is asked about its "Restoran
// profili" under a tab called "Zal va buyurtma" — which for a shop holds the
// scale, the marked goods and the till's screen, and no room at all.

export type SettingsTab =
  | "restaurant"
  | "site"
  | "hall"
  | "delivery"
  | "money"
  | "integrations";

/** The tabs in the order a restaurant reads them: what the restaurant is, what
 *  the guest sees, how the room works, how food travels, how money arrives, and
 *  what is plugged in behind all of it. */
const KITCHEN_TABS: SettingsTab[] = [
  "restaurant",
  "site",
  "hall",
  "delivery",
  "money",
  "integrations",
];

/** And the order a shop reads them.
 *
 *  ⚠️ **The counter comes second, and delivery goes near the end.** What a shop
 *  sets up on the day it opens is the scale and the marked goods; delivery is
 *  off by default for every shop but a florist (BusinessType.Defaults), so a tab
 *  about it sat third in front of every chemist on the platform. */
const SHOP_TABS: SettingsTab[] = [
  "restaurant",
  "hall",
  "site",
  "money",
  "delivery",
  "integrations",
];

export function settingsTabs(brand: BrandLike): SettingsTab[] {
  return sellsGoods(brand) ? SHOP_TABS : KITCHEN_TABS;
}

/** What a tab is called in this business.
 *
 *  ⚠️ **Two readings, not five.** A pharmacy is a shop, a boutique is a shop and
 *  a flower stall is a shop — one word covers all three, and a dictionary per
 *  business type is five copies of every string to keep in step for a gain
 *  nobody can see. The one distinction that matters is whether there is a
 *  kitchen and a dining room behind the counter. */
export function settingsTabLabel(
  tab: SettingsTab,
  t: AdminDict,
  brand: BrandLike,
): string {
  const g = t.settings.groups as unknown as Record<string, string>;
  if (!sellsGoods(brand)) return g[tab];
  // A shop's own words for the two tabs that were named after a restaurant.
  if (tab === "restaurant") return g.shop;
  if (tab === "hall") return g.counter;
  return g[tab];
}

// ---- What a printer can be asked to print, and which receipts exist ----
//
// ⚠️ **A shop has no pass and a kitchen has no shelf.** The full list offered a
// grocery a kitchen ticket to design and a printer to send it to, and offered a
// restaurant a price label for a shelf it does not have. A tick that can only
// ever produce paper nobody reads is a tick somebody tries once.
//
// ⚠️ **Data wins over the template, here as everywhere on these screens.** A
// kind already ticked on a printer, or a receipt already switched on, stays
// offered whatever kind of business this is — a bakery counter inside a shop is
// real, and hiding the box would leave a setting switched on that nobody can
// find to switch off.

/** Every kind a printer can take, in the order of the day. */
export const PRINT_KINDS = [
  "kitchen",
  "precheck",
  "till",
  "customer",
  "label",
] as const;

export type PrintKind = (typeof PRINT_KINDS)[number];

export function printKindsFor(
  brand: BrandLike,
  printers: { kinds?: string[] }[],
): PrintKind[] {
  const used = new Set(printers.flatMap((p) => p.kinds ?? []));
  return PRINT_KINDS.filter((k) => {
    if (used.has(k)) return true;
    if (k === "kitchen") return hasKitchen(brand);
    if (k === "label") return sellsGoods(brand);
    return true;
  });
}

/** The receipts this business designs. ⚠️ The guest's copy and the till's are
 *  everybody's; the kitchen ticket is a kitchen's. */
export type ReceiptKind = "customer" | "till" | "kitchen";

export function receiptKindsFor(
  brand: BrandLike,
  kitchenEnabled: boolean,
): ReceiptKind[] {
  const kinds: ReceiptKind[] = ["customer", "till"];
  if (hasKitchen(brand) || kitchenEnabled) kinds.push("kitchen");
  return kinds;
}
