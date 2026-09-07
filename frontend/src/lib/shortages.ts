// Which answers a shortfall may be given in this business, and in whose words.
//
// ⚠️ **Its own file because it is a rule, not a layout** — the same reason
// `adminNav.ts` is one. The wrong answer here is wrong for every chemist on the
// platform at once, and the only way to check it inside a page component is to
// log in as one and look.
//
// ⚠️ **Words and relevance only. The arithmetic never changes.** A shortfall is
// "what the books expected, less what was found" in a kitchen, a pharmacy and a
// flower shop alike, and `models/businesstype.go` says at length why the stock
// module is deliberately not tailored: arithmetic that changed with the business
// type would be arithmetic nobody could check, in the one part of the product
// where a wrong number is silent. What is tailored is which answers are offered
// and what the caveat calls things.

import { composes } from "@/lib/types";
import type { BrandLike } from "@/lib/adminNav";
import type { ShortageVerdict } from "@/lib/types";

/** Every verdict, in the order they are offered — most ordinary first, and the
 *  one that admits defeat last.
 *
 *  ⚠️ **"Lost" is last on purpose.** It is the only entry a reader can hear as
 *  an accusation, and a list that opens with it invites the whole queue to be
 *  filed under it. The four above it are what a shortfall usually turns out to
 *  be. */
const ALL: ShortageVerdict[] = [
  "miscount",
  "waste",
  "swap",
  "paperwork",
  "card",
  "lost",
];

/** What this business may answer with.
 *
 *  ⚠️ **A card takes more than the kitchen does — and a chemist has no kitchen
 *  and no card.** A grocery, a pharmacy and a clothes shop sell the packet they
 *  bought: the one-line card behind it is written by the server and cannot be
 *  padded, so offering that answer there is offering a reason that cannot be
 *  true. A florist keeps it: a bouquet is fifteen stems and a ribbon, which is
 *  the most literal technical card in the product.
 *
 *  ⚠️ **Hidden, never refused.** The server accepts all six (see
 *  models/shortagecase.go): a stored verdict has to mean the same thing on
 *  every install, and a chain that runs a kitchen and a shop must be able to
 *  count a month of them together. This is presentation, exactly as a hidden
 *  navigation row is. */
export function verdictsFor(brand: BrandLike): ShortageVerdict[] {
  if (composes(brand)) return ALL;
  return ALL.filter((v) => v !== "card");
}
