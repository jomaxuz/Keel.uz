// Which set of words this site uses for the page a restaurant calls a menu.
//
// ⚠️ **A clothes shop has a catalogue, not a menu**, and the difference is not
// pedantry: a navigation bar whose second item reads "Menyu" over a rail of
// dresses tells the guest, in one word, that the site was built for somebody
// else. It is the same complaint the admin sidebar already answered when a
// chemist read "Masalliqlar" over a shelf of paracetamol.
//
// ⚠️ **The address changes too, and the old one keeps working.** This file used
// to say the opposite — words change, `/menu` stays `/menu` — on the reasoning
// that every saved link, QR code and printed card points at it. That reasoning
// is sound about *breaking* links and wrong about the address a shop's guest
// sees: `ecom.uz/menu` over a rail of trainers is the same sentence as "Menyu"
// in the bar, in the one place a guest is most likely to read it and most
// likely to copy. So a shop's catalogue lives at `/catalog`, and `/menu`
// **redirects** there — nothing that was ever written down stops working, and
// nothing new says "menu".
//
// ⚠️ **Two readings, not twelve.** A pharmacy is a shop, a boutique is a shop
// and an online store is a shop — one set of words covers all three, and a
// dictionary per business type is a dozen copies of every string to keep in
// step for a gain nobody can see. Same rule `settingsTabLabel` follows in the
// panel.

import type { Dict } from "@/lib/i18n/dictionaries";
import { sellsGoods } from "@/lib/types";

export interface SiteWords {
  /** What the navigation bar calls it. */
  nav: string;
  title: string;
  eyebrow: string;
  subtitle: (cats: number, items: number) => string;
  empty: string;
  /** "Back to the menu" / "Back to the catalogue". */
  back: string;
  /** Where the catalogue lives on this site: `/menu` or `/catalog`.
   *
   *  ⚠️ **One of the two is always a redirect**, so a link that points at the
   *  other still arrives — that is what makes changing the address safe. Links
   *  written inside the site should use this so the guest lands directly. */
  href: string;
  /** How many things are in a category: "5 ta taom" or "5 ta mahsulot".
   *
   *  ⚠️ A sneaker shop counting its trainers in *dishes* is the same failure as
   *  a bar that says "Menyu" — and it is the one that was still on the page
   *  after the bar was fixed, under every category tile. */
  items: (n: number) => string;
  /** What the band that asks for the order says. */
  orderTitle: string;
  orderText: string;
}

/** The words this business uses.
 *
 *  ⚠️ An unknown business type reads as a restaurant — the same fallback every
 *  predicate in lib/types.ts makes, and here it means a site written by a newer
 *  console keeps saying "Menyu" rather than saying nothing. */
export function siteWords(t: Dict, businessType?: string): SiteWords {
  if (!sellsGoods({ businessType })) {
    return {
      nav: t.nav.menu,
      title: t.menu.title,
      eyebrow: t.menu.eyebrow,
      subtitle: t.menu.subtitle,
      empty: t.menu.empty,
      back: t.common.backToMenu,
      href: "/menu",
      items: t.common.dishes,
      orderTitle: t.home.orderTitle,
      orderText: t.home.orderText,
    };
  }
  return {
    nav: t.shop.nav,
    title: t.shop.title,
    eyebrow: t.shop.eyebrow,
    subtitle: t.shop.subtitle,
    empty: t.shop.empty,
    back: t.shop.back,
    href: "/catalog",
    items: t.shop.items,
    orderTitle: t.shop.orderTitle,
    orderText: t.shop.orderText,
  };
}
