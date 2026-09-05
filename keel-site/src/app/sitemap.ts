import type { MetadataRoute } from "next";
import { ALL_LANGS, alternatesFor, localeUrl } from "@/lib/i18n/url";
import type { Lang } from "@/lib/i18n/dict";
import { ALL_SLUGS } from "@/lib/help";
import { getPosts } from "@/lib/blog";

/** Every slug that exists in any language, once. */
function allSlugs(byLang: Record<Lang, Set<string>>): string[] {
  const all = new Set<string>();
  for (const l of ALL_LANGS) for (const s of byLang[l]) all.add(s);
  return [...all];
}

/** Every page, in every language, each as its own entry.
 *
 *  ⚠️ **Three entries per page, not one with two alternates attached.** The
 *  first version listed each page once — the Uzbek address — with `ru` and `en`
 *  hanging off it as `<xhtml:link>`, on the belief that one entry says "one page,
 *  three addresses" while three would say "three pages". That reads sensibly and
 *  is not what Google documents: in a sitemap **each language version needs its
 *  own `<url>` element**, and each of those lists the complete alternate set
 *  including itself. A one-way declaration is ignored — the same rule the pages'
 *  own `<head>` already follows through `alternatesFor`.
 *
 *  The symptom was quiet and looked like success: Search Console accepted the
 *  file and reported 93 pages, which is a real number for a site with 279
 *  addresses. Two thirds of the site was submitted as an attribute of the other
 *  third, left to be found by crawling — the exact wait a sitemap exists to
 *  skip, and the Russian pages are the ones that most needed not to wait.
 *
 *  Duplicate content is not the risk it sounds like here: `hreflang` is what
 *  makes three addresses one page, and every entry carries the full set both
 *  ways. Listing them is how Google is told they exist at all.
 *
 *  Anchor URLs are deliberately absent: a crawler treats "/#pricing" as the
 *  same page as "/", so listing them adds nothing. When the landing grows real
 *  sub-pages — a blog, a case study — they belong here, each in three languages.
 *
 *  `/status` and `/console` stay out. The console is behind a login; the status
 *  page is genuinely useful to a customer and genuinely useless in a search
 *  result, and crawl budget spent there is budget not spent on the page meant
 *  to be found. */

type Entry = MetadataRoute.Sitemap[number];

/** One path becomes one entry per language, all carrying the same alternates. */
function inEveryLanguage(
  path: string,
  rest: Omit<Entry, "url" | "alternates">,
): MetadataRoute.Sitemap {
  return ALL_LANGS.map((lang) => {
    const { canonical, languages } = alternatesFor(lang, path);
    return { url: canonical, alternates: { languages }, ...rest };
  });
}

// ⚠️ **Now async, and that is the whole change in shape.** Everything above is
// known at build time; the blog is not — a post written this afternoon has to
// be in the file this evening, and a sitemap generated once at build would list
// the posts that existed when the image was made. Next re-renders this route,
// so the list is asked for each time it is requested.
//
// ⚠️ **The control plane being unreachable must not empty the file.** `getPosts`
// answers with an empty list rather than throwing, so a bad minute costs the
// blog entries and leaves the other 279 addresses exactly where they were — the
// alternative is a sitemap that briefly says the site has no pages, which is a
// far more expensive thing to tell a crawler.
export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  // ⚠️ **Asked in Uzbek, listed in three.** A post exists at three addresses
  // whenever it is written in three languages; asking once and expanding is
  // what keeps the file from claiming a Russian page that was never written.
  // The list endpoint already leaves out what has no text in the language, so
  // the three calls are not the same list.
  const [uzPosts, ruPosts, enPosts] = await Promise.all([
    getPosts("uz"),
    getPosts("ru"),
    getPosts("en"),
  ]);
  const byLang: Record<Lang, Set<string>> = {
    uz: new Set(uzPosts.map((p) => p.slug)),
    ru: new Set(ruPosts.map((p) => p.slug)),
    en: new Set(enPosts.map((p) => p.slug)),
  };
  const lastMod = new Map<string, Date>();
  for (const p of uzPosts) {
    if (p.publishedAt) lastMod.set(p.slug, new Date(p.publishedAt));
  }

  return [
    ...inEveryLanguage("/", {
      lastModified: new Date(),
      changeFrequency: "weekly",
      priority: 1,
    }),
    // ⚠️ **Weighted just under the landing**, because it now carries the
    // product detail the home page used to. "keel kassa" is the phrase this
    // market actually types, and the page that answers it should be the page
    // that ranks for it.
    ...inEveryLanguage("/kassa", {
      lastModified: new Date(),
      changeFrequency: "monthly",
      priority: 0.9,
    }),
    // ⚠️ /download is listed but not weighted above the landing: it is a page
    // people are sent to, not one they search for. It is here at all because a
    // restaurant already using Keel searches for "keel kassa" when setting up a
    // second branch, and a page that cannot be found is a support call.
    //
    // Its own entry rather than joining the legal pages below, because it is
    // the one page here that genuinely changes: every release replaces the
    // download behind it, and telling a crawler "yearly" is how a new version
    // stays unlisted for months.
    ...inEveryLanguage("/download", {
      lastModified: new Date(),
      changeFrequency: "monthly",
      priority: 0.5,
    }),
    // ⚠️ **Every article, individually.** This is the one part of the site with
    // real long-tail search value: nobody looks for "Keel", a great many people
    // look for "техкарта как составить" or "chek chiqmayapti". Listing only
    // /help would hide eighty-eight pages behind a page that links to them,
    // which is exactly the crawl the sitemap exists to avoid.
    ...["/help", ...ALL_SLUGS.map((s) => `/help/${s}`)].flatMap((path) =>
      inEveryLanguage(path, {
        lastModified: new Date(),
        // Monthly, honestly: an article changes when the screen it describes
        // changes, which is neither weekly nor yearly.
        changeFrequency: "monthly",
        // The index above the articles: it is the page worth ranking as an entry
        // point, the articles are worth ranking one query at a time.
        priority: path === "/help" ? 0.6 : 0.4,
      }),
    ),
    // ⚠️ **The blog index, then every post that exists in each language.**
    // This is the second place on the site with real long-tail value, and the
    // one that grows: an article about weighing scales at a grocery counter is
    // a page somebody finds months before they have heard of us.
    ...inEveryLanguage("/blog", {
      lastModified: uzPosts[0]?.publishedAt
        ? new Date(uzPosts[0].publishedAt)
        : new Date(),
      // Weekly, honestly: a blog changes when somebody writes in it.
      changeFrequency: "weekly",
      priority: 0.6,
    }),
    // ⚠️ **Only the languages a post was actually written in**, and each entry
    // carries the alternates of exactly those. Declaring a Russian address for
    // a post with no Russian text is telling a crawler about a page that
    // answers in another language — which is worse than not listing it, because
    // the reader who follows it leaves.
    ...allSlugs(byLang).flatMap((slug) => {
      const langs = ALL_LANGS.filter((l) => byLang[l].has(slug));
      const languages: Record<string, string> = {};
      for (const l of langs) languages[l] = localeUrl(l, `/blog/${slug}`);
      // ⚠️ **`x-default` points at whichever language the post actually has.**
      // Uzbek is the site's base and every post has it today — but a post that
      // somehow does not would otherwise declare a default that 404s, and a
      // crawler treats that as the page it should have shown everybody.
      languages["x-default"] = localeUrl(
        langs.includes("uz") ? "uz" : langs[0],
        `/blog/${slug}`,
      );
      return langs.map((lang) => ({
        url: localeUrl(lang, `/blog/${slug}`),
        alternates: { languages },
        lastModified: lastMod.get(slug) ?? new Date(),
        changeFrequency: "monthly" as const,
        priority: 0.5,
      }));
    }),
    // ⚠️ The legal pages are listed, unlike /status. They are the pages a payment provider,
    // a bank or a cautious customer looks for by name before signing anything — and a
    // document that cannot be found is a document that does not count. Rarely changed, so a
    // low priority and a yearly frequency say so honestly.
    ...["/public-offer", "/privacy-policy"].flatMap((path) =>
      inEveryLanguage(path, {
        lastModified: new Date(),
        changeFrequency: "yearly",
        priority: 0.3,
      }),
    ),
  ];
}
