import type { MetadataRoute } from "next";
import { ORIGIN, PREFIXED_LANGS } from "@/lib/i18n/url";

/** keel.uz is one site on one domain, so this is static.
 *
 *  The console is excluded — it is behind a login, and a crawler spending its
 *  budget on a page that returns a login form is budget not spent on the
 *  pages meant to be found. The status page is excluded for a different
 *  reason: it is genuinely useful to a customer, and genuinely useless in a
 *  search result.
 *
 *  ⚠️ Both are excluded **under every language prefix too**. A rule written as a
 *  path prefix stops matching the moment the path gains a prefix of its own, so
 *  adding `/ru` and `/en` would otherwise have quietly reopened `/ru/console`
 *  to crawlers — and nothing anywhere would have reported it. */
export default function robots(): MetadataRoute.Robots {
  const priv = ["/console", "/status"];
  const disallow = [
    "/api/",
    ...priv,
    ...PREFIXED_LANGS.flatMap((l) => priv.map((p) => `/${l}${p}`)),
  ];
  return {
    rules: [{ userAgent: "*", allow: "/", disallow }],
    // Yandex treats the sitemap directive in robots.txt as its primary
    // discovery path, so it is not optional here.
    sitemap: `${ORIGIN}/sitemap.xml`,
    host: ORIGIN,
  };
}
