import type { MetadataRoute } from "next";
import { ALL_LANGS, localeUrl, ORIGIN } from "@/lib/i18n/url";
import { ALL_SLUGS } from "@/lib/help";

/** One page, in three languages.
 *
 *  ⚠️ Listed **once**, with the other two attached as alternates — not three
 *  times. Three entries describe three pages, which is the duplicate-content
 *  problem `hreflang` exists to prevent; `alternates.languages` says "same page,
 *  three addresses", and it is the form Google documents for sitemaps.
 *
 *  Anchor URLs are deliberately absent: a crawler treats "/#pricing" as the
 *  same page as "/", so listing them adds nothing. When the landing grows real
 *  sub-pages — a blog, a case study — they belong here, each with its own three
 *  addresses.
 *
 *  `/status` and `/console` stay out. The console is behind a login; the status
 *  page is genuinely useful to a customer and genuinely useless in a search
 *  result, and crawl budget spent there is budget not spent on the page meant
 *  to be found. */
export default function sitemap(): MetadataRoute.Sitemap {
  return [
    {
      url: ORIGIN,
      lastModified: new Date(),
      changeFrequency: "weekly",
      priority: 1,
      alternates: {
        languages: Object.fromEntries(ALL_LANGS.map((l) => [l, localeUrl(l, "/")])),
      },
    },
    // ⚠️ /download is listed but not weighted above the landing: it is a page
    // people are sent to, not one they search for. It is here at all because a
    // restaurant already using Keel searches for "keel kassa" when setting up a
    // second branch, and a page that cannot be found is a support call.
    //
    // Its own entry rather than joining the legal pages below, because it is
    // the one page here that genuinely changes: every release replaces the
    // download behind it, and telling a crawler "yearly" is how a new version
    // stays unlisted for months.
    {
      url: localeUrl(ALL_LANGS[0], "/download"),
      lastModified: new Date(),
      changeFrequency: "monthly",
      priority: 0.5,
      alternates: {
        languages: Object.fromEntries(
          ALL_LANGS.map((l) => [l, localeUrl(l, "/download")]),
        ),
      },
    },
    // ⚠️ **Every article, individually.** This is the one part of the site with
    // real long-tail search value: nobody looks for "Keel", a great many people
    // look for "техкарта как составить" or "chek chiqmayapti". Listing only
    // /help would hide eighty-eight pages behind a page that links to them,
    // which is exactly the crawl the sitemap exists to avoid.
    ...["/help", ...ALL_SLUGS.map((s) => `/help/${s}`)].map((path) => ({
      url: localeUrl(ALL_LANGS[0], path),
      lastModified: new Date(),
      // Monthly, honestly: an article changes when the screen it describes
      // changes, which is neither weekly nor yearly.
      changeFrequency: "monthly" as const,
      // The index above the articles: it is the page worth ranking as an entry
      // point, the articles are worth ranking one query at a time.
      priority: path === "/help" ? 0.6 : 0.4,
      alternates: {
        languages: Object.fromEntries(
          ALL_LANGS.map((l) => [l, localeUrl(l, path)]),
        ),
      },
    })),
    // ⚠️ The legal pages are listed, unlike /status. They are the pages a payment provider,
    // a bank or a cautious customer looks for by name before signing anything — and a
    // document that cannot be found is a document that does not count. Rarely changed, so a
    // low priority and a yearly frequency say so honestly.
    ...["/public-offer", "/privacy-policy"].map((path) => ({
      url: localeUrl(ALL_LANGS[0], path),
      lastModified: new Date(),
      changeFrequency: "yearly" as const,
      priority: 0.3,
      alternates: {
        languages: Object.fromEntries(ALL_LANGS.map((l) => [l, localeUrl(l, path)])),
      },
    })),
  ];
}
