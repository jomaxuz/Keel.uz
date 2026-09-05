// The help articles the panel searches.
//
// ⚠️ **Shipped with the panel, not fetched from the platform.** The obvious
// design is to hold these in the control plane so an answer can be corrected
// without releasing thirty containers. It is the wrong one twice over: an
// article describes what *this build* does, and a platform-hosted answer would
// eventually describe a version the restaurant is not running — and the moment
// somebody most needs the help is the moment their container cannot reach
// anything. Bundled, the search works with the network down and always
// describes the software in front of the person reading it.
//
// ⚠️ **An answer that is wrong is worse than no article.** Everything here is
// written from the behaviour in this repository, not from what the product
// ought to do: the PIN lockout is five minutes because that is the constant,
// the code page is chosen from the text because that is what the encoder does.
// When one of those changes, the article beside it changes in the same commit.

import type { Lang } from "@/lib/i18n/dictionaries";

export type HelpCategory =
  "till" | "orders" | "menu" | "stock" | "team" | "money" | "settings";

export type HelpArticle = {
  id: string;
  cat: HelpCategory;
  title: string;
  body: string;
  /** Words somebody would search that are not in the text.
   *
   *  ⚠️ This is where the vocabulary gap goes. An owner types "kassa
   *  ochilmayapti"; the article is titled "PIN qabul qilinmayapti". Neither
   *  contains the other's words, and without this the search finds nothing for
   *  the single most common question there is. */
  keys?: string[];
};

// ---- Where the articles now come from ----
//
// ⚠️ **Fetched from this restaurant's own server, not bundled here.** They used
// to be three arrays in this file, and the argument above still holds — an
// article describes *this build*, so it must be deployed by the same commit as
// the behaviour it explains. That is unchanged: the server embeds them
// (`backend/internal/help/articles.json`) and is deployed from the same commit
// this panel is.
//
// ⚠️ **What changed is that there is one copy.** The owner's native application
// cannot import a TypeScript module, and the obvious fix — a Kotlin copy for the
// phone — is the copy that stops describing this build the first time an
// article is corrected and only two of the three are edited. Both clients read
// the server's.
//
// ⚠️ **Cached for the session.** The base does not change while somebody has the
// panel open, and re-fetching on every keystroke of a help search would put a
// request behind a control that has to feel instant.

import { api } from "@/lib/api";

const cache = new Map<string, HelpArticle[]>();

const CATS: HelpCategory[] = [
  "till", "orders", "menu", "stock", "team", "money", "settings",
];

function res(payload: {
  articles: { id: string; cat: string; title: string; body: string; keys?: string[] }[];
}): HelpArticle[] {
  return payload.articles.map((a) => ({
    ...a,
    cat: (CATS as string[]).includes(a.cat) ? (a.cat as HelpCategory) : "settings",
  }));
}

/** The articles in a language, or the base when that language has none.
 *
 *  ⚠️ Answers `[]` rather than throwing when the server cannot be reached: the
 *  help panel then shows its "ask an operator" half, which is the useful half
 *  when the network is the problem. A thrown error here would take the whole
 *  widget down with it. */
export async function loadHelp(lang: Lang): Promise<HelpArticle[]> {
  const hit = cache.get(lang);
  if (hit) return hit;
  try {
    // ⚠️ Narrowed here rather than trusted: `cat` is a union in this file and a
    // plain string on the wire, and a category the panel does not know would
    // otherwise reach the grouping code as a key nothing renders.
    const list = res(await api.supportArticles(lang));
    cache.set(lang, list);
    return list;
  } catch {
    return [];
  }
}

