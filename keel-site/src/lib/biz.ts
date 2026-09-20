// The kinds of business Keel is sold to, and their one set of words.
//
// ⚠️ **One list, because two lists drift and the drift is invisible.** The
// console names these types in two places — the form that creates a customer
// and the breakdown that reports on them — and they were two lists with two
// dictionaries. A type added to one showed up as "Restoran" in the other: not
// an error, not a blank, just a bakery quietly counted among the restaurants on
// the screen we use to decide what to sell more of.
//
// ⚠️ **Mirrors `models.BusinessTypes` on the tenant**, which is where the type
// actually means something. The order is that list's order: kitchens first,
// then shops, because somebody creating a customer already knows which of the
// two they are typing.

import type { Dict } from "@/lib/i18n/dict";

/** Every type this build knows, in the order the console offers them.
 *
 *  ⚠️ **The empty string is a restaurant and is first.** Every customer created
 *  before this field existed carries no value, and reading it as anything else
 *  would move all of them into a row of their own. */
export const BIZ_TYPES = [
  "",
  "fastfood",
  "coffee",
  "bakery",
  "pastry",
  "grocery",
  "butcher",
  "clothing",
  "cosmetics",
  "flowers",
  "pharmacy",
  "hardware",
  // ⚠️ **Last, and not among the shops.** Somebody creating a customer knows
  // first whether the place cooks and second whether it has a room to walk
  // into; an online store answers no to both, so it belongs after every
  // business that has an address rather than interleaved with them.
  "ecommerce",
] as const;

export type BizType = (typeof BIZ_TYPES)[number];

/** What this kind of business is called, in the reader's language.
 *
 *  ⚠️ **An unknown type keeps its own code rather than becoming "Restoran".** A
 *  customer created by a newer console and read by this one is a row we cannot
 *  name — and printing somebody else's name over it is how a screen reports a
 *  number for a business that is not there. */
export function bizLabel(t: Dict, type: string): string {
  const words = t.biz.types as Record<string, string | undefined>;
  return words[type] ?? type;
}
