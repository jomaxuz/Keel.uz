// Menu search: forgiving about how the guest types, strict about what it shows.
//
// ⚠️ **Not Elasticsearch, and deliberately not.** One restaurant on one VPS
// answering a few hundred searches a day does not need a search cluster; it needs
// a search that survives how people in Uzbekistan actually type. The menu is
// already in the page — a hundred-odd dishes — so matching happens in the
// browser, instantly, with no request per keystroke on a phone connection where
// each round trip is a third of a second.
//
// What "smart" has to mean here is spelled out by the alphabet, not by the
// algorithm:
//
//   - **Two scripts.** The same guest types "лагман" today and "lagmon"
//     tomorrow, on two keyboards. Both are the same dish.
//   - **The apostrophe is optional and comes in six shapes.** "Lag'mon",
//     "lagʻmon", "lag‘mon", "lagmon" — nobody hunts for the right glyph on a
//     phone keyboard, and a search that insists on one finds nothing.
//   - **Sounds that trade places.** o'/u, x/h, q/k, v/w: standard in Uzbek
//     spelling and in Russian transliteration of it. "Qaymoq" and "kaymak" are
//     one word typed by two people.
//   - **Typos.** A dropped letter must not empty the screen — the guest sees a
//     restaurant with no lagman, not their own slip.
//
// Everything above is folding: both the query and the dish name are pushed onto
// one alphabet before anything is compared. What is left is scoring, which
// decides the order — and the order is the whole product, because a guest reads
// the first three results.

import type { MenuGroup, MenuItem } from "@/lib/types";
import type { Lang } from "@/lib/i18n/dictionaries";
import { contentDescription, contentName } from "@/lib/i18n/content";

// ---- Folding ----

// Cyrillic → Latin, in the shape Uzbek is actually written in. `х` and `ҳ` both
// land on `h` (see the x/h note below); `ц` on `s`, because "цезарь" is typed
// "sezar" as often as "tsezar".
const CYRILLIC: Record<string, string> = {
  а: "a", б: "b", в: "v", г: "g", ғ: "g", д: "d", е: "e", ё: "yo", ж: "j",
  з: "z", и: "i", й: "y", к: "k", қ: "q", л: "l", м: "m", н: "n", о: "o",
  ў: "o", п: "p", р: "r", с: "s", т: "t", у: "u", ф: "f", х: "h", ҳ: "h",
  ц: "s", ч: "ch", ш: "sh", щ: "sh", ъ: "", ы: "i", ь: "", э: "e", ю: "yu",
  я: "ya",
};

/**
 * fold pushes a string onto the one alphabet everything is compared in.
 *
 * ⚠️ Both sides go through this, always. Folding only the query would make the
 * feature look half-working: "лагман" would find nothing while "lagmon" found
 * everything, and the bug would be invisible to whoever tested it in one script.
 */
export function fold(input: string): string {
  let s = input.toLowerCase();
  // Combining marks first, so "é" is an "e" before anything else looks at it.
  s = s.normalize("NFD").replace(/[\u0300-\u036f]/g, "");
  s = s.replace(/[а-яёғқҳў]/g, (ch) => CYRILLIC[ch] ?? ch);
  // Every apostrophe shape, including the two Unicode ones Uzbek orthography
  // actually specifies (ʻ ʼ) and the four keyboards produce instead.
  s = s.replace(/['’‘`´ʻʼ]/g, "");
  // Sounds that trade places. Applied to both sides, so they are not "wrong
  // spellings" being corrected — they are two spellings meeting in the middle.
  s = s
    .replace(/x/g, "h")
    .replace(/q/g, "k")
    .replace(/w/g, "v")
    .replace(/ts/g, "s")
    .replace(/yo/g, "o")
    .replace(/yu/g, "u")
    .replace(/ya/g, "a");
  // o/u after the above: "do'lma"/"dulma", "sho'rva"/"shurva".
  s = s.replace(/u/g, "o");
  // Anything that is not a letter or a digit is a separator. Hyphens, dots and
  // the "×" in combo contents included.
  s = s.replace(/[^a-z0-9]+/g, " ").trim();
  return s;
}

export function tokens(input: string): string[] {
  const f = fold(input);
  return f ? f.split(" ") : [];
}

// ---- Distance ----

/**
 * Bounded Levenshtein: how many edits, giving up past `max`.
 *
 * Bounded on purpose — an unbounded distance over a 200-dish menu on every
 * keystroke is work thrown away, since anything past two edits is not a typo,
 * it is a different word.
 */
export function editDistance(a: string, b: string, max: number): number {
  if (a === b) return 0;
  if (Math.abs(a.length - b.length) > max) return max + 1;
  let prev = Array.from({ length: b.length + 1 }, (_, i) => i);
  let cur = new Array<number>(b.length + 1);
  for (let i = 1; i <= a.length; i++) {
    cur[0] = i;
    let best = cur[0];
    for (let j = 1; j <= b.length; j++) {
      const cost = a[i - 1] === b[j - 1] ? 0 : 1;
      cur[j] = Math.min(prev[j] + 1, cur[j - 1] + 1, prev[j - 1] + cost);
      if (cur[j] < best) best = cur[j];
    }
    // Whole row already past the budget: no later row can come back under it.
    if (best > max) return max + 1;
    [prev, cur] = [cur, prev];
  }
  return prev[b.length];
}

/** How wrong a word of this length is allowed to be. Short words get no slack:
 *  at four letters, two edits reaches a different dish. */
function budget(len: number): number {
  if (len <= 3) return 0;
  if (len <= 6) return 1;
  return 2;
}

// ---- Scoring ----

/** How well one query token matches one field's tokens: 0 when it does not. */
function tokenScore(query: string, fieldTokens: string[]): number {
  let best = 0;
  for (const word of fieldTokens) {
    if (word === query) return 1;
    if (word.startsWith(query)) {
      // A prefix of a long word is a weaker signal than a prefix of a short one
      // ("sam" in "samsa" beats "sam" in "samarqandcha palov").
      best = Math.max(best, 0.85 - Math.min(0.2, (word.length - query.length) * 0.02));
      continue;
    }
    if (query.length >= 3 && word.includes(query)) {
      best = Math.max(best, 0.6);
      continue;
    }
    const max = budget(query.length);
    if (max > 0) {
      const d = editDistance(query, word, max);
      if (d <= max) best = Math.max(best, 0.45 - (d - 1) * 0.1);
    }
  }
  return best;
}

/** The searchable text of one dish, folded once and kept. */
export interface Indexed {
  item: MenuItem;
  categoryId: string;
  /** Field tokens, heaviest first. Weight lives in `FIELD_WEIGHTS`. */
  fields: string[][];
}

// Name above everything: a guest typing "somsa" wants the dish called somsa,
// not the three dishes whose description mentions it. The description is worth
// searching at all only because it is where "achchiq" and "tovuqli" live.
const FIELD_WEIGHTS = [1, 0.5, 0.45, 0.4, 0.22];

/**
 * buildIndex folds every dish once, in the language on screen.
 *
 * ⚠️ Language matters here: the guest reading a Russian menu types Russian, and
 * the Russian name is the one on their screen. The base (uz) name is indexed too
 * — the dish is called that on the receipt, and half of Tashkent types it that
 * way whatever language the page is in.
 */
export function buildIndex(groups: MenuGroup[], lang: Lang): Indexed[] {
  const out: Indexed[] = [];
  for (const g of groups) {
    const category = contentName(g.category, lang);
    for (const item of g.items) {
      const shown = contentName(item, lang);
      const combo = (item.comboContents ?? []).map((c) => c.name).join(" ");
      out.push({
        item,
        categoryId: g.category.id,
        fields: [
          // Both the displayed name and the base one, so neither script loses.
          tokens(`${shown} ${item.name}`),
          tokens(category),
          tokens((item.tags ?? []).join(" ")),
          tokens(combo),
          tokens(contentDescription(item, lang)),
        ],
      });
    }
  }
  return out;
}

/**
 * score is how well one dish answers one query, or 0 for "do not show it".
 *
 * ⚠️ **Every query word has to match something.** Scoring the sum and keeping
 * the top N would let "achchiq lag'mon" return every lagman and every spicy
 * dish — a longer query would widen the results, which is the opposite of what
 * typing more words means.
 */
export function score(entry: Indexed, queryTokens: string[]): number {
  let total = 0;
  for (const q of queryTokens) {
    let bestForToken = 0;
    for (let f = 0; f < entry.fields.length; f++) {
      const s = tokenScore(q, entry.fields[f]);
      if (s > 0) bestForToken = Math.max(bestForToken, s * FIELD_WEIGHTS[f]);
    }
    if (bestForToken === 0) return 0;
    total += bestForToken;
  }
  // Nudges, not verdicts: they only separate dishes the text already tied.
  if (entry.item.isPopular) total += 0.06;
  // What cannot be ordered sinks rather than disappears — a guest searching for
  // it deserves to be told it ran out, not left wondering whether it exists.
  if (!entry.item.isAvailable || entry.item.soldOut) total -= 0.5;
  return total;
}

/** The active filters. Every field is "no opinion" when empty or false. */
export interface MenuFilters {
  categoryIds: string[];
  minPrice: number | null;
  maxPrice: number | null;
  tags: string[];
  /** Hide what cannot be ordered right now. */
  availableOnly: boolean;
  popularOnly: boolean;
  discountOnly: boolean;
  comboOnly: boolean;
  sort: MenuSort;
}

export type MenuSort = "relevance" | "cheap" | "expensive" | "popular";

export const NO_FILTERS: MenuFilters = {
  categoryIds: [],
  minPrice: null,
  maxPrice: null,
  tags: [],
  availableOnly: false,
  popularOnly: false,
  discountOnly: false,
  comboOnly: false,
  sort: "relevance",
};

/** How many filters the guest has actually set — the number on the icon. */
export function activeFilterCount(f: MenuFilters): number {
  return (
    (f.categoryIds.length > 0 ? 1 : 0) +
    (f.minPrice != null || f.maxPrice != null ? 1 : 0) +
    f.tags.length +
    (f.availableOnly ? 1 : 0) +
    (f.popularOnly ? 1 : 0) +
    (f.discountOnly ? 1 : 0) +
    (f.comboOnly ? 1 : 0) +
    (f.sort !== "relevance" ? 1 : 0)
  );
}

function passesFilters(entry: Indexed, f: MenuFilters): boolean {
  const it = entry.item;
  if (f.categoryIds.length && !f.categoryIds.includes(entry.categoryId)) return false;
  if (f.minPrice != null && it.price < f.minPrice) return false;
  if (f.maxPrice != null && it.price > f.maxPrice) return false;
  if (f.availableOnly && (!it.isAvailable || it.soldOut)) return false;
  if (f.popularOnly && !it.isPopular) return false;
  if (f.discountOnly && !(it.oldPrice != null && it.oldPrice > it.price)) return false;
  if (f.comboOnly && !(it.comboItems?.length || it.comboContents?.length)) return false;
  if (f.tags.length) {
    const has = (it.tags ?? []).map((t) => fold(t));
    if (!f.tags.every((t) => has.includes(fold(t)))) return false;
  }
  return true;
}

/**
 * runSearch applies the query and the filters and returns dishes in the order
 * they should be read.
 *
 * With no query the order is the menu's own (the owner arranged it, and that
 * arrangement is a decision about what to sell) — relevance has nothing to say
 * when nothing was asked.
 */
export function runSearch(
  index: Indexed[],
  query: string,
  filters: MenuFilters,
): MenuItem[] {
  const qt = tokens(query);
  const rows: { entry: Indexed; score: number; order: number }[] = [];
  index.forEach((entry, order) => {
    if (!passesFilters(entry, filters)) return;
    if (qt.length === 0) {
      rows.push({ entry, score: 0, order });
      return;
    }
    const s = score(entry, qt);
    if (s > 0) rows.push({ entry, score: s, order });
  });

  rows.sort((a, b) => {
    switch (filters.sort) {
      case "cheap":
        return a.entry.item.price - b.entry.item.price || a.order - b.order;
      case "expensive":
        return b.entry.item.price - a.entry.item.price || a.order - b.order;
      case "popular":
        return (
          Number(b.entry.item.isPopular) - Number(a.entry.item.isPopular) ||
          b.score - a.score ||
          a.order - b.order
        );
      default:
        return b.score - a.score || a.order - b.order;
    }
  });
  return rows.map((r) => r.entry.item);
}
