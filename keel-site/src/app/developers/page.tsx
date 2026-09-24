// keel.uz/developers — the open API manual.
//
// ⚠️ **On keel.uz rather than on each restaurant's domain.** Every restaurant
// runs its own API at its own address, but the contract is one: a developer
// integrating their tenth Keel restaurant should read one page, and a search
// for "keel api" should find it. The owner hands them the address and the key;
// this page is everything else.
//
// ⚠️ **Server-rendered in three languages, each with its own address**, like
// /help — a reference that exists only after JavaScript runs is one no crawler
// reads. The only client component is the copy button.
//
// The content is data (`lib/developers`); this file decides only how it reads.
// The published contract is `docs/open-api.md` in the repo.

import type { Metadata } from "next";
import type { ReactNode } from "react";
import Header from "@/components/Header";
import CodeBlock from "@/components/developers/CodeBlock";
import { Footer } from "@/components/landing/Shell";
import { API_VERSION, developers, type Block } from "@/lib/developers";
import { getLang, getPath, getT } from "@/lib/i18n/server";
import { alternatesFor } from "@/lib/i18n/url";

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getLang();
  const path = await getPath();
  const d = developers[lang];
  return {
    title: d.title,
    description: d.description,
    alternates: alternatesFor(lang, path),
  };
}

/** `code` and **bold** inside a sentence — the only two marks the prose uses. */
function inline(text: string): ReactNode[] {
  return text.split(/(`[^`]+`|\*\*[^*]+\*\*)/g).map((part, i) => {
    if (part.startsWith("`") && part.endsWith("`")) {
      return (
        <code
          key={i}
          className="rounded bg-raised px-1 py-0.5 font-mono text-[0.85em] text-ink"
        >
          {part.slice(1, -1)}
        </code>
      );
    }
    if (part.startsWith("**") && part.endsWith("**")) {
      return (
        <strong key={i} className="font-semibold text-ink">
          {part.slice(2, -2)}
        </strong>
      );
    }
    return part;
  });
}

function BlockView({ b, copy, copied }: { b: Block; copy: string; copied: string }) {
  switch (b.kind) {
    case "p":
      return <p className="text-[15px] leading-relaxed text-ink-soft">{inline(b.text)}</p>;
    case "code":
      return <CodeBlock code={b.code} label={b.label} copy={copy} copied={copied} />;
    case "note":
      return (
        <p className="rounded-xl border border-signal-500/40 bg-signal-500/10 px-4 py-3 text-sm leading-relaxed text-ink">
          {inline(b.text)}
        </p>
      );
    case "list":
      return (
        <ul className="space-y-2">
          {b.items.map((it, i) => (
            <li key={i} className="flex gap-2.5 text-[15px] leading-relaxed text-ink-soft">
              <span aria-hidden className="mt-2.5 h-1 w-1 shrink-0 rounded-full bg-ink-muted" />
              <span>{inline(it)}</span>
            </li>
          ))}
        </ul>
      );
    case "table":
      // Scrolls inside itself on a phone: the page must never scroll sideways.
      return (
        <div className="overflow-x-auto rounded-xl border border-line">
          <table className="w-full min-w-[480px] text-left text-sm">
            <thead className="bg-raised text-ink">
              <tr>
                {b.head.map((h, i) => (
                  <th key={i} className="px-4 py-2 font-semibold">
                    {inline(h)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-line">
              {b.rows.map((row, i) => (
                <tr key={i} className="align-top">
                  {row.map((cell, j) => (
                    <td key={j} className="px-4 py-2 text-ink-soft">
                      {inline(cell)}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      );
  }
}

export default async function DevelopersPage() {
  const lang = await getLang();
  const t = await getT();
  const d = developers[lang];

  return (
    <>
      <Header />
      <main className="container-page py-14 sm:py-20">
        <p className="eyebrow">
          API · {d.ui.version} {API_VERSION}
        </p>
        <h1 className="h-display mt-2 text-3xl sm:text-4xl">{d.title}</h1>
        <p className="mt-4 max-w-2xl text-lg leading-relaxed text-ink-soft">{d.lead}</p>

        <div className="mt-12 grid gap-10 lg:grid-cols-[220px_minmax(0,1fr)]">
          {/* The table of contents: a sticky column on a wide screen, a plain
              list above the text on a phone — a sticky sidebar at 360px is a
              third of the screen showing links. */}
          <nav aria-label={d.ui.onThisPage} className="lg:sticky lg:top-24 lg:self-start">
            <p className="text-xs font-semibold uppercase tracking-wider text-ink-muted">
              {d.ui.onThisPage}
            </p>
            <ol className="mt-3 grid grid-cols-2 gap-x-4 gap-y-1.5 text-sm sm:grid-cols-3 lg:grid-cols-1">
              {d.sections.map((s) => (
                <li key={s.id}>
                  <a href={`#${s.id}`} className="text-ink-soft hover:text-ink">
                    {s.title}
                  </a>
                </li>
              ))}
            </ol>
          </nav>

          <article className="min-w-0 max-w-3xl space-y-14">
            {d.sections.map((s) => (
              <section key={s.id} id={s.id} className="scroll-mt-24 space-y-4">
                <h2 className="h-display text-2xl">
                  <a href={`#${s.id}`} className="hover:text-signal-600">
                    {s.title}
                  </a>
                </h2>
                {s.blocks.map((b, i) => (
                  <BlockView key={i} b={b} copy={d.ui.copy} copied={d.ui.copied} />
                ))}
              </section>
            ))}
          </article>
        </div>
      </main>
      <Footer t={t} lang={lang} />
    </>
  );
}
