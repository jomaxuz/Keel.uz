import type { MetadataRoute } from "next";

/** One page, and the anchors on it.
 *
 *  Anchor URLs are deliberately not listed: a crawler treats "/#pricing" as
 *  the same page as "/", so listing them adds nothing and dilutes nothing.
 *  When the landing grows real sub-pages — a blog, a case study — they belong
 *  here and each one is worth its own entry. */
export default function sitemap(): MetadataRoute.Sitemap {
  return [
    {
      url: "https://keel.uz",
      lastModified: new Date(),
      changeFrequency: "weekly",
      priority: 1,
    },
  ];
}
