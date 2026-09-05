// The blog, read from the control plane.
//
// ⚠️ **Server-rendered, with a deadline.** Both pages below run per request, so
// a control plane having a bad day must not stop keel.uz for every visitor —
// the same rule and the same two seconds the partner strip follows, and for the
// same reason: this is a call to a neighbouring container, and anything slower
// is broken rather than slow.

import { CONTROL, INTERNAL } from "@/lib/partners";

const TIMEOUT_MS = 2000;

export type BlogCard = {
  slug: string;
  cover?: string;
  title: string;
  excerpt: string;
  publishedAt?: string;
  views: number;
};

export type BlogFull = BlogCard & {
  body: string;
  /** Which languages this post exists in. ⚠️ Sent so the page can offer the
   *  switch honestly rather than sending a reader onto an empty page. */
  langs: string[];
};

export async function getPosts(lang: string): Promise<BlogCard[]> {
  try {
    const res = await fetch(`${CONTROL}${INTERNAL}/blog?lang=${lang}`, {
      // ⚠️ A minute, not "no-store": a blog changes when somebody writes, and
      // a marketing page re-fetched on every visit is a database query per
      // crawler hit.
      next: { revalidate: 60 },
      signal: AbortSignal.timeout(TIMEOUT_MS),
    });
    if (!res.ok) return [];
    const body = (await res.json()) as { posts?: BlogCard[] };
    return body.posts ?? [];
  } catch {
    return [];
  }
}

export async function getPost(
  slug: string,
  lang: string,
): Promise<BlogFull | null> {
  try {
    const res = await fetch(
      `${CONTROL}${INTERNAL}/blog/${encodeURIComponent(slug)}?lang=${lang}`,
      {
        // ⚠️ **Not cached.** The read is what counts the view; a cached page is
        // a post that stops being counted the moment it becomes popular, which
        // is exactly backwards.
        cache: "no-store",
        signal: AbortSignal.timeout(TIMEOUT_MS),
      },
    );
    if (!res.ok) return null;
    return (await res.json()) as BlogFull;
  } catch {
    return null;
  }
}
