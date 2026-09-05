// keel.uz/blog — what we have written.
//
// ⚠️ **Server-rendered, in three languages, each with its own address**, for
// the reason the manual is: a page that only exists after JavaScript runs is a
// page no crawler ever reads, and half the point of writing is being found.
//
// ⚠️ **A post absent in this language is absent from this list**, not shown in
// another one. A Russian card that opens onto an Uzbek page has handed the
// reader a language nobody offered them, and a shorter list is the honest
// version of the same page.

import type { Metadata } from "next";
import Link from "next/link";

import Header from "@/components/Header";
import { BlogIndexJsonLd } from "@/components/blog/BlogJsonLd";
import { getPosts } from "@/lib/blog";
import { getLang, getPath } from "@/lib/i18n/server";
import { dicts } from "@/lib/i18n/dict";
import { alternatesFor, localePath } from "@/lib/i18n/url";
import { dateOf } from "@/lib/blogDate";

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getLang();
  const path = await getPath();
  const t = dicts[lang].blog;
  return {
    title: `${t.title} · Keel`,
    description: t.lead,
    alternates: alternatesFor(lang, path),
  };
}

export default async function BlogIndex() {
  const lang = await getLang();
  const t = dicts[lang].blog;
  const posts = await getPosts(lang);

  return (
    <>
      <BlogIndexJsonLd lang={lang} title={t.title} lead={t.lead} posts={posts} />
      <Header />
      <main className="container-page py-14 sm:py-20">
        <p className="eyebrow">Keel</p>
        <h1 className="h-display mt-2 text-3xl sm:text-4xl">{t.title}</h1>
        <p className="mt-3 max-w-2xl text-lg text-ink-soft">{t.lead}</p>

        {posts.length === 0 ? (
          <p className="mt-12 text-ink-muted">{t.empty}</p>
        ) : (
          <div className="mt-12 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {posts.map((p) => (
              <Link
                key={p.slug}
                href={localePath(lang, `/blog/${p.slug}`)}
                className="group flex flex-col overflow-hidden rounded-3xl border border-line bg-surface transition-colors hover:border-signal-500/50"
              >
                {p.cover && (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={p.cover}
                    alt=""
                    className="aspect-[16/9] w-full object-cover"
                    loading="lazy"
                  />
                )}
                <div className="flex flex-1 flex-col p-5">
                  <h2 className="font-display text-lg font-semibold leading-snug text-ink group-hover:text-signal-600">
                    {p.title}
                  </h2>
                  {p.excerpt && (
                    <p className="mt-2 line-clamp-3 text-sm text-ink-muted">
                      {p.excerpt}
                    </p>
                  )}
                  {/* ⚠️ **The date and the reads, on every card.** A blog with
                      no dates reads as a blog nobody has touched in a year —
                      which is the impression it makes whether or not it is
                      true. */}
                  <p className="mt-4 flex items-center gap-2 pt-1 text-xs text-ink-muted">
                    <span>{dateOf(p.publishedAt, lang)}</span>
                    <span aria-hidden>·</span>
                    <span>{t.views(p.views)}</span>
                  </p>
                </div>
              </Link>
            ))}
          </div>
        )}
      </main>
    </>
  );
}
