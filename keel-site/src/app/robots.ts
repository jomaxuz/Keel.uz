import type { MetadataRoute } from "next";

/** keel.uz is one site on one domain, so this is static.
 *
 *  The console is excluded — it is behind a login, and a crawler spending its
 *  budget on a page that returns a login form is budget not spent on the
 *  pages meant to be found. The status page is excluded for a different
 *  reason: it is genuinely useful to a customer, and genuinely useless in a
 *  search result. */
export default function robots(): MetadataRoute.Robots {
  return {
    rules: [{ userAgent: "*", allow: "/", disallow: ["/console", "/api/", "/status"] }],
    // Yandex treats the sitemap directive in robots.txt as its primary
    // discovery path, so it is not optional here.
    sitemap: "https://keel.uz/sitemap.xml",
    host: "https://keel.uz",
  };
}
