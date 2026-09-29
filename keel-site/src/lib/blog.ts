// The blog, read from the control plane.
//
// ⚠️ **Server-rendered, with a deadline.** Both pages below run per request, so
// a control plane having a bad day must not stop keel.uz for every visitor —
// the same rule and the same two seconds the partner strip follows, and for the
// same reason: this is a call to a neighbouring container, and anything slower
// is broken rather than slow.

import { cache } from "react";

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

/** One post.
 *
 *  ⚠️ **Wrapped in `cache` because the page asks twice.** Next builds the title
 *  and description from `generateMetadata` and then draws the words, and both
 *  need the post — two calls to the control plane for one reader. `cache`
 *  collapses them within the request. It used to be worse than an extra
 *  request: the read counted a view, so a single visit counted two. */
export const getPost = cache(async function getPost(
  slug: string,
  lang: string,
): Promise<BlogFull | null> {
  try {
    const res = await fetch(
      `${CONTROL}${INTERNAL}/blog/${encodeURIComponent(slug)}?lang=${lang}`,
      {
        // ⚠️ **Cacheable again**, because reading no longer counts anything:
        // the reader's browser says it read (see components/blog/CountView).
        // A minute, like the list — a post changes when somebody edits it.
        next: { revalidate: 60 },
        signal: AbortSignal.timeout(TIMEOUT_MS),
      },
    );
    if (!res.ok) return null;
    return (await res.json()) as BlogFull;
  } catch {
    return null;
  }
});

/** Every published post in one language, with its body — for `/llms-full.txt`.
 *
 *  ⚠️ **One request, not one per post.** The full file carries every article's
 *  words, and building it from `getPost` would be a round trip per post on a
 *  route crawlers fetch. The control plane answers the whole list at once
 *  (`/internal/blog/full`), and like the rest of this file an unreachable
 *  control plane is an empty list — the help centre and the product pages are
 *  still in the file without it.
 *
 *  ⚠️ Same minute of cache as the list, so a post saved in the console is in the
 *  file within about a minute — the console's watcher waits past that before it
 *  reads the file back and pings (control/internal/handlers/llms.go). */
export async function getPostsFull(lang: string): Promise<(BlogCard & { body: string })[]> {
  try {
    const res = await fetch(`${CONTROL}${INTERNAL}/blog/full?lang=${lang}`, {
      next: { revalidate: 60 },
      signal: AbortSignal.timeout(TIMEOUT_MS * 2),
    });
    if (!res.ok) return [];
    const body = (await res.json()) as { posts?: (BlogCard & { body: string })[] };
    return body.posts ?? [];
  } catch {
    return [];
  }
}
