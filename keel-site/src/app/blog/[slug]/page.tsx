// One article.
//
// ⚠️ **The reading is counted by the reader, not by this render.** The page is
// rendered more than once per visit — the title and description are built in
// their own pass — so a counter here counted two for one reader and three when
// somebody switched language. See components/blog/CountView.

import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";

import Header from "@/components/Header";
import Article from "@/components/blog/Article";
import { BlogPostJsonLd } from "@/components/blog/BlogJsonLd";
import CountView from "@/components/blog/CountView";
import { getPost } from "@/lib/blog";
import { getLang, getPath } from "@/lib/i18n/server";
import { dicts } from "@/lib/i18n/dict";
import { alternatesFor, localePath } from "@/lib/i18n/url";
import { dateOf } from "@/lib/blogDate";

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const lang = await getLang();
  const path = await getPath();
  const post = await getPost(slug, lang);
  if (!post) return { title: "Keel" };
  return {
    title: `${post.title} · Keel`,
    description: post.excerpt,
    alternates: alternatesFor(lang, path),
    openGraph: {
      title: post.title,
      description: post.excerpt,
      type: "article",
      publishedTime: post.publishedAt,
      images: post.cover ? [post.cover] : undefined,
    },
  };
}

export default async function BlogArticle({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const lang = await getLang();
  const t = dicts[lang].blog;
  const post = await getPost(slug, lang);
  if (!post) notFound();

  return (
    <>
      <BlogPostJsonLd lang={lang} blogTitle={t.title} post={post} />
      <CountView slug={post.slug} />
      <Header />
      <main className="container-page py-14 sm:py-20">
        <Link
          href={localePath(lang, "/blog")}
          className="text-sm text-ink-muted hover:text-signal-600"
        >
          ← {t.title}
        </Link>

        <article className="mt-6 max-w-3xl">
          <h1 className="h-display text-3xl sm:text-4xl">{post.title}</h1>
          <p className="mt-3 flex items-center gap-2 text-sm text-ink-muted">
            <span>{dateOf(post.publishedAt, lang)}</span>
            <span aria-hidden>·</span>
            <span>{t.views(post.views)}</span>
          </p>

          {post.cover && (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={post.cover}
              alt=""
              className="mt-8 w-full rounded-3xl border border-line"
            />
          )}

          <div className="mt-8">
            <Article body={post.body} />
          </div>
        </article>
      </main>
    </>
  );
}
