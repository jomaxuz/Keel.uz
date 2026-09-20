// Which set of words this site uses for the page a restaurant calls a menu.
//
// ⚠️ **A clothes shop has a catalogue, not a menu**, and the difference is not
// pedantry: a navigation bar whose second item reads "Menyu" over a rail of
// dresses tells the guest, in one word, that the site was built for somebody
// else. It is the same complaint the admin sidebar already answered when a
// chemist read "Masalliqlar" over a shelf of paracetamol.
//
// ⚠️ **Words, never a second route.** `/menu` stays `/menu`. Every saved link,
// QR code, sitemap entry, Telegram button and printed card in existence points
// at it, and a shop gains nothing from a different address that it does not
// lose twice over in links that stop working. What changes is what the page is
// called, which is the whole of what a guest sees.
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
    };
  }
  return {
    nav: t.shop.nav,
    title: t.shop.title,
    eyebrow: t.shop.eyebrow,
    subtitle: t.shop.subtitle,
    empty: t.shop.empty,
    back: t.shop.back,
  };
}
