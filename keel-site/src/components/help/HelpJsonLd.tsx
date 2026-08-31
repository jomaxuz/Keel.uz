// What tells a search engine that a help article is an article.
//
// ⚠️ **This is where the long tail actually lives.** Nobody searches for
// "Keel". A great many people search "техкарта как составить", "chek
// chiqmayapti", "iiko dan qanday ko'chirish" — and 88 articles in three
// languages is 264 pages that can answer one of those, if a crawler can tell
// what they are. The root layout already declares who we are and what we sell;
// none of that says "this page is a technical article, in this language, part
// of this manual, under this section".
//
// ⚠️ **Nothing here is invented.** Every field is a value the page already
// shows — the title, the lead, the section, the language. Structured data that
// disagrees with the page is not ignored, it is a reason to distrust the block
// and, eventually, the site.
//
// ⚠️ **No `datePublished`.** It is the field every guide tells you to add, and
// we do not have an honest one: these articles have no per-article date, and a
// build timestamp would tell Google that all 264 were written this morning and
// rewritten every deploy. A missing field costs nothing; a false one is a
// claim.

import type { Article, Section } from "@/lib/help/types";
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

/** One article. */
export function ArticleJsonLd({
  lang,
  article,
  section,
  helpTitle,
}: {
  lang: Lang;
  article: Article;
  section?: Section;
  helpTitle: string;
}) {
  const url = localeUrl(lang, `/help/${article.slug}`);
  const plain = (s: string) => s.replace(/[*`]/g, "");

  // ⚠️ The trail is what a search result shows instead of a bare URL —
  // "Keel › Qo'llanma › Ombor" rather than "keel.uz/help/tech-cards". It is
  // also the only place the section a reader is in becomes machine-readable.
  const crumbs = [
    { name: "Keel", item: localeUrl(lang, "/") },
    { name: helpTitle, item: localeUrl(lang, "/help") },
    ...(section ? [{ name: section.title, item: localeUrl(lang, "/help") }] : []),
    { name: article.title, item: url },
  ];

  return (
    <>
      <Ld
        data={{
          "@context": "https://schema.org",
          "@type": "TechArticle",
          headline: article.title,
          description: plain(article.lead),
          inLanguage: lang,
          url,
          mainEntityOfPage: { "@type": "WebPage", "@id": url },
          isPartOf: { "@id": `${ORIGIN}#website` },
          publisher: { "@id": `${ORIGIN}#organization` },
          ...(section ? { articleSection: section.title } : {}),
          // The words somebody would search that are not in the prose — the
          // vocabulary gap the articles already carry for their own search box.
          ...(article.keys?.length ? { keywords: article.keys.join(", ") } : {}),
        }}
      />
      <Ld
        data={{
          "@context": "https://schema.org",
          "@type": "BreadcrumbList",
          itemListElement: crumbs.map((c, i) => ({
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

/** The index. */
export function HelpIndexJsonLd({
  lang,
  title,
  lead,
  sections,
}: {
  lang: Lang;
  title: string;
  lead: string;
  sections: Section[];
}) {
  const url = localeUrl(lang, "/help");
  return (
    <>
      <Ld
        data={{
          "@context": "https://schema.org",
          "@type": "CollectionPage",
          name: title,
          description: lead,
          inLanguage: lang,
          url,
          isPartOf: { "@id": `${ORIGIN}#website` },
          publisher: { "@id": `${ORIGIN}#organization` },
          hasPart: sections.map((s) => ({
            "@type": "CreativeWork",
            name: s.title,
            description: s.lead,
          })),
        }}
      />
      <Ld
        data={{
          "@context": "https://schema.org",
          "@type": "BreadcrumbList",
          itemListElement: [
            { "@type": "ListItem", position: 1, name: "Keel", item: localeUrl(lang, "/") },
            { "@type": "ListItem", position: 2, name: title, item: url },
          ],
        }}
      />
    </>
  );
}
