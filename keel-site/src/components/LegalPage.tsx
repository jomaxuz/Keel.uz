// One renderer for both legal documents.
//
// ⚠️ **Both pages, one component.** An offer and a policy that look different are two pages
// somebody has to read twice as carefully — and the second one always ends up worse, because
// it is the one nobody revisits. The documents are data (`lib/legal.ts`); this decides only
// how they are read.
//
// ⚠️ **The draft warning is at the top, not the bottom.** These were written from the product
// rather than from a lawyer's desk, and the person who most needs to know that is the one
// about to rely on them. Hiding it under ten sections would be the same as leaving it out.

import Header from "@/components/Header";
import type { LegalDoc } from "@/lib/legal";

export default function LegalPage({ doc }: { doc: LegalDoc }) {
  return (
    <>
      <Header />
      <main className="container-page py-16 sm:py-20">
        <h1 className="h-display text-3xl sm:text-4xl">{doc.title}</h1>
        <p className="mt-2 text-xs text-ink-muted">{doc.updated}</p>

        <p className="mt-6 rounded-2xl border border-line bg-raised px-4 py-3 text-sm text-ink-soft">
          {doc.disclaimer}
        </p>

        {/* A reading width, not the full page: a legal document at 1200px is a document
            nobody finishes a paragraph of. */}
        <div className="mt-8 max-w-3xl space-y-8">
          {doc.intro.map((p, i) => (
            <p key={i} className="text-base leading-relaxed text-ink-soft">
              {p}
            </p>
          ))}

          {doc.sections.map((s) => (
            <section key={s.heading}>
              <h2 className="text-lg font-bold text-ink">{s.heading}</h2>
              <div className="mt-3 space-y-2.5">
                {s.body.map((line, i) =>
                  // A line beginning with an em dash is a list item, and is drawn as one
                  // rather than as a paragraph that happens to start with punctuation.
                  line.startsWith("— ") ? (
                    <p key={i} className="flex gap-2 text-sm leading-relaxed text-ink-soft">
                      <span aria-hidden className="text-ink-muted">
                        —
                      </span>
                      <span>{line.slice(2)}</span>
                    </p>
                  ) : (
                    <p key={i} className="text-sm leading-relaxed text-ink-soft">
                      {line}
                    </p>
                  ),
                )}
              </div>
            </section>
          ))}
        </div>
      </main>
    </>
  );
}
