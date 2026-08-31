// One article.
//
// ⚠️ **Static, and generated for every slug at build time.** These pages are
// the ones a search engine is meant to hold, and they have no data behind them
// — rendering them per request would be a database-free page paying a
// request-time cost forever.

import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import Header from "@/components/Header";
import Blocks from "@/components/help/Blocks";
import SectionIcon from "@/components/help/SectionIcon";
import { ArticleJsonLd } from "@/components/help/HelpJsonLd";
import { ALL_SLUGS, articleBySlug, help, neighbours, plain } from "@/lib/help";
import { getLang } from "@/lib/i18n/server";
import { alternatesFor, localePath } from "@/lib/i18n/url";
import { TELEGRAM } from "@/lib/links";

export function generateStaticParams() {
  return ALL_SLUGS.map((slug) => ({ slug }));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const lang = await getLang();
  const a = articleBySlug(lang, slug);
  if (!a) return { title: "Keel" };
  return {
    title: `${a.title} · Keel`,
    description: plain(a.lead),
    // ⚠️ Built from the slug rather than from the request path: this page is
    // pre-rendered, so there is no request to read a header off.
    alternates: alternatesFor(lang, `/help/${slug}`),
  };
}

export default async function HelpArticle({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const lang = await getLang();
  const a = articleBySlug(lang, slug);
  if (!a) notFound();

  const { ui, sections } = help[lang];
  const section = sections.find((s) => s.id === a.section);
  const { prev, next } = neighbours(lang, slug);

  return (
    <>
      <ArticleJsonLd
        lang={lang}
        article={a}
        section={section}
        helpTitle={ui.title}
      />
      <Header />
      <main className="container-page py-12 sm:py-16">
        <div className="mx-auto max-w-[760px]">
          <nav className="flex flex-wrap items-center gap-2 text-sm text-ink-muted">
            <Link
              href={localePath(lang, "/help")}
              className="underline-offset-4 hover:text-ink hover:underline"
            >
              {ui.title}
            </Link>
            {section && (
              <>
                <span aria-hidden>·</span>
                {/* ⚠️ The icon repeats the one on the index card, and that
                    repetition is the point: it is the only thing on this page
                    that says "you are still in the same section" to somebody
                    who arrived from a search result rather than the index. */}
                <span className="inline-flex items-center gap-1.5">
                  <SectionIcon id={section.id} className="h-4 w-4" />
                  {section.title}
                </span>
              </>
            )}
          </nav>

          <h1 className="h-display mt-3 text-2xl sm:text-3xl">{a.title}</h1>
          <p className="mt-3 text-lg text-ink-soft">{a.lead}</p>

          <div className="mt-8">
            <Blocks
              blocks={a.body}
              lang={lang}
              figureHint={ui.figureHint}
              seeAlso={ui.seeAlso}
            />
          </div>

          <div className="mt-12 rounded-2xl border border-line bg-raised px-5 py-4">
            <p className="font-medium text-ink">{ui.askUs}</p>
            <p className="mt-1 text-sm text-ink-soft">{ui.askUsLead}</p>
            <a href={TELEGRAM} className="btn-ghost mt-3 text-sm">
              Telegram
            </a>
          </div>

          {/* ⚠️ Across section boundaries. Somebody reading the store from the
              top is working through it; stopping them at the last article of a
              section sends them back to an index to find the page that was
              next anyway. */}
          <nav className="mt-8 grid gap-3 sm:grid-cols-2">
            {prev ? (
              <Link
                href={localePath(lang, `/help/${prev.slug}`)}
                className="rounded-2xl border border-line px-4 py-3 transition hover:bg-raised"
              >
                <span className="block text-xs uppercase tracking-wide text-ink-muted">
                  {ui.prev}
                </span>
                <span className="mt-0.5 block text-[15px] font-medium text-ink">
                  {prev.title}
                </span>
              </Link>
            ) : (
              <span />
            )}
            {next && (
              <Link
                href={localePath(lang, `/help/${next.slug}`)}
                className="rounded-2xl border border-line px-4 py-3 text-right transition hover:bg-raised sm:text-right"
              >
                <span className="block text-xs uppercase tracking-wide text-ink-muted">
                  {ui.next}
                </span>
                <span className="mt-0.5 block text-[15px] font-medium text-ink">
                  {next.title}
                </span>
              </Link>
            )}
          </nav>
        </div>
      </main>
    </>
  );
}
