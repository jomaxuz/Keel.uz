"use client";

// Says that somebody read this.
//
// ⚠️ **Once per page, from the browser, and that is the whole point.** The
// counter used to live inside the read on the server — and the page fetches
// itself twice on every render, once for the title and description and once for
// the words — so one visit counted two and switching language counted three.
// The number was wrong from the first day and wrong in the flattering
// direction, which is the kind nobody questions.
//
// ⚠️ **A ref, not the effect's own guard.** React runs effects twice in
// development on purpose, and a `[]` dependency array does not stop the second
// run — which is exactly the double-count this component exists to remove,
// reintroduced in the place it is easiest to miss.

import { useEffect, useRef } from "react";

export default function CountView({ slug }: { slug: string }) {
  const sent = useRef(false);
  useEffect(() => {
    if (sent.current) return;
    sent.current = true;
    // Fire and forget: nothing on this page waits for it, and a reader who
    // leaves before it lands has still read the post.
    void fetch(`/blog-view/${encodeURIComponent(slug)}`, {
      method: "POST",
      keepalive: true,
    }).catch(() => {});
  }, [slug]);
  return null;
}
