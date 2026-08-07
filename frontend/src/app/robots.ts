import type { MetadataRoute } from "next";
import { siteOrigin } from "@/lib/seo";
import { PREFIXED_LANGS } from "@/lib/i18n";

// Served per host, because one build serves every restaurant: a static file
// would name one site's sitemap to all of them.
export const dynamic = "force-dynamic";

/** What a crawler is allowed to read.
 *
 *  The disallow list is not about secrecy — every one of these pages already
 *  refuses to render anything private without a token. It is about **not
 *  spending the crawl budget on pages that cannot rank**: a cart, a checkout
 *  and a per-order tracking URL are different for every visitor and worth
 *  nothing in a search result, and an order number in an index is a URL
 *  somebody can guess at.
 *
 *  `/api/` and `/uploads/` are excluded from crawling but the images
 *  themselves stay reachable — Open Graph and rich results fetch them
 *  directly, which robots.txt does not affect. */
export default async function robots(): Promise<MetadataRoute.Robots> {
  const origin = await siteOrigin();

  // The private pages, and — because every page now also answers under `/ru`
  // and `/en` — the same list under each prefix. Without this the language
  // rewrite would quietly reopen `/ru/checkout` and `/ru/order/1234` to
  // crawlers: a rule written as a path prefix stops matching the moment the
  // path gains a prefix of its own, and nothing anywhere reports it.
  const priv = ["/admin", "/kuryer", "/staff", "/kiosk", "/cart", "/checkout", "/order/", "/profile"];
  const disallow = [
    "/api/",
    ...priv,
    ...PREFIXED_LANGS.flatMap((l) => priv.map((p) => `/${l}${p}`)),
  ];

  return {
    rules: [{ userAgent: "*", allow: "/", disallow }],
    // The line Yandex in particular looks for: its crawler treats the sitemap
    // directive in robots.txt as the primary discovery path.
    sitemap: `${origin}/sitemap.xml`,
    host: origin,
  };
}
