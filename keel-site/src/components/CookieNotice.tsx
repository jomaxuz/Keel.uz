"use client";

// The cookie notice on keel.uz.
//
// ⚠️ **Copied rather than imported**, like the logo mark and the charts: keel.uz and the
// restaurant app are separate builds sharing no code, and a package for one component would be
// a dependency to keep in step for years.
//
// ⚠️ **A notice, not a question — and the code says so.** Nothing on this page waits for the
// button: the language cookie and the Meta pixel both run from the first paint (see
// `lib/pixel.ts`). One button, because a second one labelled "decline" would change nothing,
// and a button that changes nothing is what teaches people these banners are decoration —
// which is precisely why nobody reads the one on a site where it does work.
//
// So the honesty has to sit in two other places instead, and it must stay there: the notice
// says cookies are in use rather than claiming they are "only the necessary ones", and the
// privacy policy behind the link names the pixel, what it is for, and how to refuse it in the
// browser. Whoever edits this text next: those are the sentences carrying the disclosure now.

import { useEffect, useState } from "react";
import Link from "next/link";
import { useT } from "@/lib/i18n/client";
import { localePath } from "@/lib/i18n/url";

const KEY = "cookie_notice_v1";

export default function CookieNotice() {
  const { t, lang } = useT();
  // Only after mount: the server cannot know what this browser answered, and rendering it
  // during SSR would flash the banner at everybody who already has.
  const [show, setShow] = useState(false);

  useEffect(() => {
    try {
      // "seen" is this page's answer; the other two are accepted here so that somebody who
      // answered on a restaurant site built from the same key is not asked twice.
      const v = window.localStorage.getItem(KEY);
      if (v !== "seen" && v !== "accepted" && v !== "declined") setShow(true);
    } catch {
      // Storage refused — shown again next visit, which is the honest failure.
      setShow(true);
    }
  }, []);

  if (!show) return null;

  function dismiss() {
    try {
      window.localStorage.setItem(KEY, "seen");
    } catch {
      /* nothing to remember it with */
    }
    setShow(false);
  }

  return (
    <div className="fixed inset-x-0 bottom-0 z-50 p-3 sm:p-4">
      <div className="mx-auto flex max-w-3xl flex-col gap-3 rounded-2xl border border-line bg-surface p-4 shadow-xl shadow-hull-950/15 sm:flex-row sm:items-center">
        <p className="flex-1 text-sm text-ink-soft">
          {t.cookies.text}{" "}
          <Link
            href={localePath(lang, "/privacy-policy")}
            className="font-semibold text-signal-600 hover:underline"
          >
            {t.cookies.more}
          </Link>
        </p>
        <button
          type="button"
          onClick={dismiss}
          className="shrink-0 rounded-xl bg-ink px-5 py-2 text-sm font-semibold text-surface"
        >
          {t.cookies.ok}
        </button>
      </div>
    </div>
  );
}
