// Assembly and lookup.
//
// ⚠️ **Bundled with the site, not fetched.** Same reason the panel's own help
// is bundled: an article describes the build it shipped with, and the moment
// somebody most needs it is the moment their connection is worst. It also
// means every article is server-rendered as static HTML, which is what makes
// it findable — a knowledge base that only exists after JavaScript runs is one
// no search engine ever reads.

import type { Lang } from "@/lib/i18n/dict";
import type { Article, Block, HelpContent, SectionId } from "./types";
import figuresJson from "./figures.json";
import { helpUz } from "./uz";
import { helpRu } from "./ru";
import { helpEn } from "./en";
import type { Figure } from "./types";

export const FIGURES = figuresJson as Record<string, Figure>;

export const help: Record<Lang, HelpContent> = {
  uz: helpUz,
  ru: helpRu,
  en: helpEn,
};

export function articleBySlug(lang: Lang, slug: string): Article | undefined {
  return help[lang].articles.find((a) => a.slug === slug);
}

export function articlesIn(lang: Lang, section: SectionId): Article[] {
  return help[lang].articles.filter((a) => a.section === section);
}

/** Every slug, for `generateStaticParams` and the sitemap.
 *
 *  ⚠️ Read from Uzbek and not from the language being rendered: the three
 *  languages are the same set of articles by construction (the dictionary type
 *  enforces it), and a per-language list would let a missing translation
 *  quietly produce a page that exists in one language and 404s in another —
 *  after `hreflang` has already told a crawler all three exist. */
export const ALL_SLUGS: string[] = helpUz.articles.map((a) => a.slug);

/** Plain text of a block, for the search index and the meta description. */
function blockText(b: Block): string {
  if ("p" in b) return b.p;
  if ("h" in b) return b.h;
  if ("warn" in b) return b.warn;
  if ("tip" in b) return b.tip;
  if ("steps" in b) return b.steps.join(" ");
  if ("list" in b) return b.list.join(" ");
  if ("fig" in b) return Object.values(b.notes ?? {}).join(" ");
  if ("table" in b) {
    return [b.table.head.join(" "), ...b.table.rows.map((r) => r.join(" "))].join(" ");
  }
  return "";
}

export function articleText(a: Article): string {
  return [a.title, a.lead, ...(a.keys ?? []), ...a.body.map(blockText)].join(" ");
}

/** The first paragraph, stripped of markup — the meta description and the
 *  sentence a search result shows. */
export function plain(s: string): string {
  return s.replace(/[*`]/g, "");
}

/** Ranked search over one language.
 *
 *  ⚠️ **A title hit outranks a body hit, and both outrank nothing.** The naive
 *  version — filter by "contains" — puts an article that mentions "printer"
 *  once in passing above the one called "Printerni ulash", which is the single
 *  most common query this page will ever get.
 *
 *  ⚠️ Every term must match. Somebody typing two words is narrowing, not
 *  widening: "kassa chek" should not return every article about the till.
 */
export function search(lang: Lang, query: string): Article[] {
  const terms = query
    .toLowerCase()
    .split(/\s+/)
    .map((t) => t.trim())
    .filter((t) => t.length > 1);
  if (terms.length === 0) return [];

  const scored: { a: Article; score: number }[] = [];
  for (const a of help[lang].articles) {
    const title = (a.title + " " + (a.keys ?? []).join(" ")).toLowerCase();
    const body = articleText(a).toLowerCase();
    let score = 0;
    let all = true;
    for (const term of terms) {
      if (title.includes(term)) score += 10;
      else if (body.includes(term)) score += 1;
      else all = false;
    }
    if (all) scored.push({ a, score });
  }
  scored.sort((x, y) => y.score - x.score);
  return scored.map((s) => s.a);
}

/** The article before and after this one, in reading order.
 *
 *  ⚠️ Across sections, not within one: somebody reading "Zagotovka" to the end
 *  is working through the store, and stopping them at the section boundary
 *  sends them back to an index to find the page that was next anyway. */
export function neighbours(lang: Lang, slug: string) {
  const list = help[lang].articles;
  const i = list.findIndex((a) => a.slug === slug);
  return {
    prev: i > 0 ? list[i - 1] : undefined,
    next: i >= 0 && i < list.length - 1 ? list[i + 1] : undefined,
  };
}
