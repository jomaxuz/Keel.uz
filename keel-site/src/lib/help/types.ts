// The shape of a help article.
//
// ⚠️ **Blocks, not HTML strings.** The knowledge base exists three times over,
// and a paragraph written as markup is a paragraph a translator has to keep
// balanced tags inside. A block also renders differently on a phone than a
// string of `<div>`s can: a numbered step becomes a real list, a warning gets a
// colour, a screenshot gets its legend underneath rather than beside it.
//
// ⚠️ **A figure carries no text.** The picture is one file, shared by all three
// languages, and the callouts over it are numbers; the words belong to the
// article, in the language the article is in. Baking "Bu yerga bosing" into a
// PNG means three of every screenshot, kept in step by hand, forever — and the
// day one of them is missed, a Russian reader is looking at an Uzbek arrow.

import type { Lang } from "@/lib/i18n/dict";

/** Where a callout points, as a share of the frame. Measured from the running
 *  panel by `scripts/help-screens.mjs`, never typed by hand. */
export type NoteBox = { x: number; y: number; w: number; h: number };

/** One frame in one language. */
export type FigureShot = {
  /** Capture size, so the renderer knows the aspect ratio before the image
   *  loads — without it every article reflows as its screenshots arrive. */
  w: number;
  h: number;
  notes?: Record<string, NoteBox>;
};

/** The same screen in each language.
 *
 *  ⚠️ **Three images and three sets of coordinates, not one of each.** A
 *  Russian reader shown a picture of an Uzbek panel is being shown a screen
 *  that is not theirs — the words in the frame are exactly the words they are
 *  meant to find on their own. And the boxes have to be per language too:
 *  «Заготовки · 3» is a different width from «Zagotovkalar · 3», so everything
 *  to the right of it has moved. */
export type Figure = Partial<Record<Lang, FigureShot>>;

/** One piece of an article.
 *
 *  Inline, `*text*` is bold and `` `text` `` is a button, field or menu name —
 *  the two things a set of instructions actually needs to mark, and the reason
 *  this is not a general markdown renderer. */
export type Block =
  | { p: string }
  | { h: string }
  /** Numbered: do this, then this. */
  | { steps: string[] }
  | { list: string[] }
  /** The thing that goes wrong, and it always says what goes wrong rather than
   *  "be careful". A warning that does not name a consequence is decoration. */
  | { warn: string }
  | { tip: string }
  /** A screenshot with numbered outlines. Keys must exist in `figures.json`
   *  for that frame; the order here is the order the numbers are drawn in. */
  | { fig: string; notes?: Record<string, string> }
  | { table: { head: string[]; rows: string[][] } }
  /** A pointer to another article, by slug. */
  | { see: string[] };

export type SectionId =
  | "start"
  | "site"
  | "menu"
  | "orders"
  | "delivery"
  | "till"
  | "printers"
  | "stock"
  | "team"
  | "customers"
  | "integrations"
  | "reports"
  | "settings";

export type Article = {
  slug: string;
  section: SectionId;
  title: string;
  /** One sentence. Shown on the section card and under a search hit, so it has
   *  to answer "is this my question" without the article being opened. */
  lead: string;
  /** Words somebody would search that are not in the text.
   *
   *  ⚠️ This is where the vocabulary gap goes, and it is the same gap the
   *  panel's own help has: an owner types "chek chiqmayapti", the article is
   *  called "Printerni ulash". Neither contains the other's words. */
  keys?: string[];
  body: Block[];
};

export type Section = {
  id: SectionId;
  title: string;
  lead: string;
};

/** One language's worth. Uzbek is the source of truth — `HelpDict` is derived
 *  from it, so a section or article missing from ru or en is a compile error
 *  rather than an Uzbek paragraph in the middle of a Russian page. */
export type HelpContent = {
  ui: {
    title: string;
    lead: string;
    searchPlaceholder: string;
    /** ⚠️ Templates with `{q}` / `{n}`, not functions.
     *
     *  The search box is a client component and a function cannot cross that
     *  boundary — the page rendered a 500 the first time these were arrow
     *  functions. Strings also keep the whole dictionary as plain data, which
     *  is what lets a translator work on it without touching code. */
    searchEmpty: string;
    searchCount: string;
    allArticles: string;
    inSection: string;
    back: string;
    next: string;
    prev: string;
    seeAlso: string;
    figureHint: string;
    notFound: string;
    notFoundLead: string;
    askUs: string;
    askUsLead: string;
    updated: string;
  };
  sections: Section[];
  articles: Article[];
};
