import type { MetadataRoute } from "next";
import { api } from "@/lib/api";
import { getSiteScope } from "@/lib/siteBrand.server";
import { siteOrigin } from "@/lib/seo";

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

  const fixed: MetadataRoute.Sitemap = [
    { url: origin, lastModified: now, changeFrequency: "daily", priority: 1 },
    { url: `${origin}/menu`, lastModified: now, changeFrequency: "daily", priority: 0.9 },
    { url: `${origin}/about`, lastModified: now, changeFrequency: "monthly", priority: 0.5 },
  ];

  try {
    const scope = await getSiteScope();
    const groups = await api.getMenu(scope);
    const dishes: MetadataRoute.Sitemap = groups.flatMap((g) =>
      g.items
        // A dish that is switched off should not be offered to a searcher who
        // would land on a page telling them they cannot have it.
        .filter((it) => it.isAvailable)
        .map((it) => ({
          url: `${origin}/menu/${it.id}`,
          lastModified: it.updatedAt ? new Date(it.updatedAt) : now,
          changeFrequency: "weekly" as const,
          priority: 0.7,
        })),
    );
    return [...fixed, ...dishes];
  } catch {
    // A backend hiccup must not produce an empty sitemap: an empty one is a
    // positive statement that this site has no pages, and a crawler believes
    // it. The three fixed entries are always true.
    return fixed;
  }
}
