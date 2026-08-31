// What turns a blue link in a search result into a card with a name, a logo
// and a price under it.
//
// Every tenant site already emits this; keel.uz — the page whose entire job is
// to be found by somebody shopping for exactly this — emitted none. Three
// things are declared, and they answer three different questions:
//
//   • **Organization** — who is behind this. Carries the logo Google shows
//     beside the result and the contact a knowledge panel is built from.
//   • **WebSite** — the site itself, and its language variants.
//   • **SoftwareApplication** — what is being sold, with the real price. The
//     one that can put "800 so'm / buyurtma" under the link, which is the
//     single most persuasive thing about this product.
//
// ⚠️ **Nothing is invented.** A field with no honest value is omitted rather
// than filled with a placeholder: structured data that disagrees with the page
// is not ignored, it is a reason to distrust the whole block. The prices below
// are the platform's own ladder, and they have to be changed here when they are
// changed there — hence the pointer in the comment rather than a second copy of
// the reasoning.

import type { Dict } from "@/lib/i18n/dict";
import { ALL_LANGS, localeUrl, ORIGIN } from "@/lib/i18n/url";

/** The volume ladder, mirroring `PRICE_TIERS` in the control plane
 *  (`3000:800,15000:560,50000:400,0:300`). Stated as an offer catalogue rather
 *  than one price because a single "from 300" would be true and useless: the
 *  number a new customer actually pays is the first tier. */
const TIERS = [
  { name: "0–3 000", price: 800 },
  { name: "3 000–15 000", price: 560 },
  { name: "15 000–50 000", price: 400 },
  { name: "50 000+", price: 300 },
];

export default function StructuredData({ t, path }: { t: Dict; path: string }) {
  const logo = `${ORIGIN}/icon.svg`;

  // ⚠️ Typed loosely on purpose: this is a JSON-LD document, not a domain
  // model, and every node has a different shape. Inferring a union from the
  // first three entries makes the fourth a compile error for no benefit — the
  // thing that validates this is Google's own testing tool.
  const graph: Record<string, unknown>[] = [
    {
      "@type": "Organization",
      "@id": `${ORIGIN}#organization`,
      name: "Keel",
      url: ORIGIN,
      logo,
      description: t.hero.lead,
      // Only the countries actually served. "Worldwide" on a product sold in
      // two cities is the kind of claim that makes the rest of the block worth
      // less, not more.
      areaServed: "UZ",
    },
    {
      "@type": "WebSite",
      "@id": `${ORIGIN}#website`,
      url: ORIGIN,
      name: "Keel",
      description: t.hero.lead,
      publisher: { "@id": `${ORIGIN}#organization` },
      inLanguage: ALL_LANGS,
    },
    {
      "@type": "SoftwareApplication",
      "@id": `${ORIGIN}#product`,
      name: "Keel",
      applicationCategory: "BusinessApplication",
      operatingSystem: "Web",
      url: localeUrl("uz", path),
      description: t.hero.lead,
      publisher: { "@id": `${ORIGIN}#organization` },
      offers: TIERS.map((tier) => ({
        "@type": "Offer",
        name: tier.name,
        price: tier.price,
        priceCurrency: "UZS",
        // Per order, not per month — the whole pitch is that there is no
        // subscription, and an offer that omitted this would read as one.
        unitText: t.pricing.perOrder,
        availability: "https://schema.org/InStock",
      })),
    },
  ];

  // ⚠️ **Only on the landing, because that is the only page the questions are
  // on.** An FAQPage declared on a page that does not show the answers is the
  // kind of mismatch that gets structured data ignored site-wide, not just
  // here — and the answers below are the ones a visitor reads in `#faq`.
  if (path === "/") {
    graph.push({
      "@type": "FAQPage",
      "@id": `${ORIGIN}#faq`,
      mainEntity: t.faq.items.map((item) => ({
        "@type": "Question",
        name: item.q,
        acceptedAnswer: { "@type": "Answer", text: item.a },
      })),
    });
  }

  return (
    <script
      type="application/ld+json"
      // Serialised rather than written as JSX: this must be one JSON document,
      // and React would otherwise escape it into something no parser reads.
      dangerouslySetInnerHTML={{
        __html: JSON.stringify({ "@context": "https://schema.org", "@graph": graph }),
      }}
    />
  );
}
