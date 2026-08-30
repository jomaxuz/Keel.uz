// Searching the help articles.
//
// ⚠️ **Word-start matching, not whole words and not "contains".** This is the
// same lesson the operator console's search cost: Uzbek is agglutinative, so
// somebody typing "chek" has to find "chekda", "chekni" and "cheklar", and
// somebody typing "printer" has to find "printerdan". A whole-word match finds
// none of them and the panel shows "nothing found" for an article three lines
// down the list.
//
// It must not be a plain substring match either. "kassa" inside "kassada" is a
// hit; "kassa" inside some unrelated word that happens to contain the letters
// is noise, and noise at the top of a help search is what teaches somebody to
// stop using it.

import type { HelpArticle } from "./articles";

/** Where a hit was found, in order of how much it means. */
const WEIGHT = { title: 6, keys: 4, body: 1 };

/** What counts as a word boundary. ⚠️ The apostrophe is a letter in Uzbek —
 *  "o'zgartirish" is one word — so it is deliberately not in this set. A
 *  boundary rule that split on it would make "o'zgartirish" two words and stop
 *  "o'zgar" from matching it. */
function splitWords(text: string): string[] {
  return text
    .toLowerCase()
    .split(/[^\p{L}\p{N}'’]+/u)
    .filter(Boolean);
}

/** The shortest article word that may swallow a longer query term. */
const STEM_MIN = 3;

/** Does any word in `text` relate to `term`?
 *
 *  ⚠️ **Both directions, because Uzbek suffixes both ways round.** The article
 *  says "chekda" and the person types "chek" — the article's word is longer.
 *  The article says "til" and the person types "tilida" — the query's word is
 *  longer. Only checking the first direction made a whole natural question fail
 *  on one of its five words, which with an all-words rule meant no answer at
 *  all. */
function relates(text: string, term: string): boolean {
  return splitWords(text).some(
    (w) =>
      w.startsWith(term) || (w.length >= STEM_MIN && term.startsWith(w)),
  );
}

export type HelpHit = { article: HelpArticle; score: number };

/**
 * Ranks articles against what somebody typed.
 *
 * ⚠️ **Most of the query has to hit, not all of it and not one word.**
 *
 * One word is too little: "chek" alone would put every receipt article above
 * the one that is actually about language. All of them is too much, and it was
 * the first rule here — an owner typing "chek rus tilida chiqmayapti" gets
 * nothing, because "chiqmayapti" is the negative of the article's "chiqyapti"
 * and Uzbek puts that "ma" in the *middle* of the word. No prefix rule can
 * relate them, and a help search that answers a keyword but not a sentence is a
 * help search people stop typing sentences into.
 *
 * Half the words, then, scored by where they hit. Words under three letters are
 * dropped before counting: they carry no meaning and would match most of the
 * base.
 */
export function searchHelp(articles: HelpArticle[], query: string): HelpHit[] {
  const terms = splitWords(query).filter((w) => w.length >= 3);
  if (terms.length === 0) return [];

  const hits: HelpHit[] = [];
  for (const article of articles) {
    let score = 0;
    let matched = 0;
    for (const term of terms) {
      let best = 0;
      if (relates(article.title, term)) best = WEIGHT.title;
      else if (article.keys?.some((k) => relates(k, term))) best = WEIGHT.keys;
      else if (relates(article.body, term)) best = WEIGHT.body;
      if (best === 0) continue;
      matched++;
      score += best;
    }
    // ⚠️ Rounded up, so a two-word question needs both and a three-word
    // question needs two. Rounding down would let one word out of two answer,
    // which is the "chek" problem again.
    if (matched > 0 && matched * 2 >= terms.length) {
      hits.push({ article, score });
    }
  }
  // ⚠️ Ties broken by id rather than left to array order, so the same question
  // asked twice gives the same answers in the same order. A help list that
  // reshuffles looks like the answer changed.
  hits.sort((a, b) =>
    b.score - a.score || a.article.id.localeCompare(b.article.id),
  );
  return hits;
}
