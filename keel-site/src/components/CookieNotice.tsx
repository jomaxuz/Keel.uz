"use client";

// The cookie notice on keel.uz.
//
// ⚠️ **Copied rather than imported**, like the logo mark and the charts: keel.uz and the
// restaurant app are separate builds sharing no code, and a package for one component would be
// a dependency to keep in step for years.
//
// ⚠️ **One button, not two, and that is the whole point.** The restaurant site offers a real
// choice because it has one optional thing — the anonymous visit counter — and declining
// switches it off. This page has no counter, no analytics and no third-party script; the only
// thing it keeps is the chosen language, which cannot be declined without breaking the page.
// So a "decline" here would run exactly the same code as "accept". Writing "there is no
// difference" in the text does not fix that: a button that changes nothing is still a button
// that changes nothing, and it teaches people that these buttons are decoration — which is
// precisely why nobody reads the one on the site where it does work. So this is a notice, not
// a question, and it is worded as one.

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
