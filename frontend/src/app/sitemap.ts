import type { MetadataRoute } from "next";
import { api } from "@/lib/api";
import { getSiteScope } from "@/lib/siteBrand.server";
import { siteOrigin } from "@/lib/seo";
import { DEFAULT_LANG, LANGS, localePath } from "@/lib/i18n";

// Per host, and computed at request time: one build serves every restaurant,
// so a sitemap generated once would list one shop's dishes to all of them.
export const dynamic = "force-dynamic";
// Re-read every few minutes rather than on every crawl hit. A menu changes a
// few times a week; a crawler can ask far more often than that.
export const revalidate = 600;

/** Every page worth putting in a search result.
 *
 *  Deliberately only four kinds: the home page, the menu, each dish, and the
 *  about page. A cart or a checkout is different for every visitor and ranks
 *  for nothing; listing them spends the crawl budget on pages that can never
 *  earn a click.
 *
 *  **The dish pages are the point.** They are what somebody searching for
 *  "lag'mon yetkazib berish" can actually land on, and they are the only pages
 *  here with content a search engine has not already seen on ten thousand
 *  other restaurant sites.
 */
export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const origin = await siteOrigin();
  const now = new Date();

  const abs = (lang: (typeof LANGS)[number], path: string) => {
    const p = localePath(lang, path);
    return p === "/" ? origin : `${origin}${p}`;
  };

  /** One page, listed once at its Uzbek address with the other two languages
   *  attached.
   *
   *  ⚠️ Listed **once**, not three times. Three separate entries describe three
   *  pages, which is the duplicate-content problem `hreflang` exists to
   *  prevent; `alternates.languages` says "same page, three addresses" — and it
   *  is the form Google documents for sitemaps. Every variant, including the
   *  Uzbek one, appears in its own alternate list: a one-way declaration is
   *  ignored. */
  const entry = (
    path: string,
    rest: Omit<MetadataRoute.Sitemap[number], "url" | "alternates">,
  ): MetadataRoute.Sitemap[number] => ({
    url: abs(DEFAULT_LANG, path),
    ...rest,
    alternates: {
      languages: Object.fromEntries(LANGS.map((l) => [l, abs(l, path)])),
    },
  });

  const fixed: MetadataRoute.Sitemap = [
    entry("/", { lastModified: now, changeFrequency: "daily", priority: 1 }),
    entry("/menu", { lastModified: now, changeFrequency: "daily", priority: 0.9 }),
    entry("/about", { lastModified: now, changeFrequency: "monthly", priority: 0.5 }),
  ];

  try {
    const scope = await getSiteScope();
    const groups = await api.getMenu(scope);
    const dishes: MetadataRoute.Sitemap = groups.flatMap((g) =>
      g.items
        // A dish that is switched off should not be offered to a searcher who
        // would land on a page telling them they cannot have it.
        .filter((it) => it.isAvailable)
        .map((it) =>
          entry(`/menu/${it.id}`, {
            lastModified: it.updatedAt ? new Date(it.updatedAt) : now,
            changeFrequency: "weekly" as const,
            priority: 0.7,
          }),
        ),
    );
    return [...fixed, ...dishes];
  } catch {
    // A backend hiccup must not produce an empty sitemap: an empty one is a
    // positive statement that this site has no pages, and a crawler believes
    // it. The three fixed entries are always true.
    return fixed;
  }
}
