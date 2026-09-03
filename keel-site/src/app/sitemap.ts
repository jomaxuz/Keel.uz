import type { MetadataRoute } from "next";
import { ALL_LANGS, alternatesFor } from "@/lib/i18n/url";
import { ALL_SLUGS } from "@/lib/help";

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

export default function sitemap(): MetadataRoute.Sitemap {
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
