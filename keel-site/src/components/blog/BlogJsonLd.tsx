// What tells a search engine that a post is an article.
//
// ⚠️ **The blog is the one part of this site with a date worth declaring**, and
// that is the difference from the manual next door. The help articles carry no
// `datePublished` because we have no honest one for them — a build timestamp
// would claim all 264 were written this morning. A post has a real publication
// day, set once and kept, so it is stated: a dated article is eligible for
// things an undated one is not.
//
// ⚠️ **Nothing here is invented.** Every field is a value the page already
// shows — the title, the summary, the day, the language, the picture.
// Structured data that disagrees with the page is not ignored; it is a reason
// to distrust the block and, in the end, the site.

import type { Lang } from "@/lib/i18n/dict";
import { localeUrl, ORIGIN } from "@/lib/i18n/url";

function Ld({ data }: { data: unknown }) {
  return (
    <script
      type="application/ld+json"
      // Serialised rather than written as JSX: this has to be one JSON
      // document, and React would escape it into something no parser reads.
      dangerouslySetInnerHTML={{ __html: JSON.stringify(data) }}
    />
  );
}

/** The list. */
export function BlogIndexJsonLd({
  lang,
  title,
  lead,
  posts,
}: {
  lang: Lang;
  title: string;
  lead: string;
  posts: { slug: string; title: string; publishedAt?: string }[];
}) {
  const url = localeUrl(lang, "/blog");
  return (
    <Ld
      data={{
        "@context": "https://schema.org",
        "@type": "Blog",
        name: title,
        description: lead,
        inLanguage: lang,
        url,
        isPartOf: { "@id": `${ORIGIN}#website` },
        publisher: { "@id": `${ORIGIN}#organization` },
        blogPost: posts.map((p) => ({
          "@type": "BlogPosting",
          headline: p.title,
          url: localeUrl(lang, `/blog/${p.slug}`),
          ...(p.publishedAt ? { datePublished: p.publishedAt } : {}),
        })),
      }}
    />
  );
}

/** One post, and the trail to it. */
export function BlogPostJsonLd({
  lang,
  blogTitle,
  post,
}: {
  lang: Lang;
  blogTitle: string;
  post: {
    slug: string;
    title: string;
    excerpt: string;
    cover?: string;
    publishedAt?: string;
  };
}) {
  const url = localeUrl(lang, `/blog/${post.slug}`);
  // ⚠️ A relative cover has to become absolute here: a search engine reading
  // this block is not on our origin, and "/internal/blog/image/…" resolves to
  // nothing from where it stands.
  const image = post.cover
    ? post.cover.startsWith("http")
      ? post.cover
      : `${ORIGIN}${post.cover}`
    : undefined;

  return (
    <>
      <Ld
        data={{
          "@context": "https://schema.org",
          "@type": "BlogPosting",
          headline: post.title,
          description: post.excerpt,
          inLanguage: lang,
          url,
          mainEntityOfPage: url,
          ...(image ? { image } : {}),
          ...(post.publishedAt ? { datePublished: post.publishedAt } : {}),
          isPartOf: { "@id": `${ORIGIN}#website` },
          publisher: { "@id": `${ORIGIN}#organization` },
          author: { "@type": "Organization", name: "Keel" },
        }}
      />
      {/* The trail is what a search result shows instead of a bare URL —
          "Keel › Blog › …" rather than "keel.uz/blog/…". */}
      <Ld
        data={{
          "@context": "https://schema.org",
          "@type": "BreadcrumbList",
          itemListElement: [
            { name: "Keel", item: localeUrl(lang, "/") },
            { name: blogTitle, item: localeUrl(lang, "/blog") },
            { name: post.title, item: url },
          ].map((c, i) => ({
            "@type": "ListItem",
            position: i + 1,
            name: c.name,
            item: c.item,
          })),
        }}
      />
    </>
  );
}
