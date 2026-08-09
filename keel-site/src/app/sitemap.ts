import type { MetadataRoute } from "next";
import { ALL_LANGS, localeUrl, ORIGIN } from "@/lib/i18n/url";

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
