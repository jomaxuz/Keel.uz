"use client";

// The cookie notice on keel.uz.
//
// ⚠️ **Copied rather than imported**, like the logo mark and the charts: keel.uz and the
// restaurant app are separate builds sharing no code, and a package for one component would be
// a dependency to keep in step for years.
//
// ⚠️ **What "necessary only" means here is different, and the text says so.** The restaurant
// site has an optional visit counter that declining switches off. This landing page has no
// counter, no analytics and no third-party script at all — the only thing it keeps is the
// chosen language. So the second button records the answer and nothing changes, and the notice
// states that plainly instead of implying a choice that is not there. A banner that claims to
// switch something off and does not is worse than no banner.

import { useEffect, useState } from "react";
import Link from "next/link";
import { useT } from "@/lib/i18n/client";
import { localePath } from "@/lib/i18n/url";

const KEY = "cookie_choice_v1";

export default function CookieNotice() {
  const { t, lang } = useT();
  // Only after mount: the server cannot know what this browser answered, and rendering it
  // during SSR would flash the banner at everybody who already has.
  const [show, setShow] = useState(false);

  useEffect(() => {
    try {
      const v = window.localStorage.getItem(KEY);
      if (v !== "accepted" && v !== "declined") setShow(true);
    } catch {
      // Storage refused — shown again next visit, which is the honest failure.
      setShow(true);
    }
  }, []);

  if (!show) return null;

  function answer(choice: "accepted" | "declined") {
    try {
      window.localStorage.setItem(KEY, choice);
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
        <div className="flex shrink-0 gap-2">
          <button
            type="button"
            onClick={() => answer("declined")}
            className="rounded-xl border border-line px-4 py-2 text-sm font-semibold text-ink-soft"
          >
            {t.cookies.decline}
          </button>
          <button
            type="button"
            onClick={() => answer("accepted")}
            className="rounded-xl bg-ink px-5 py-2 text-sm font-semibold text-surface"
          >
            {t.cookies.accept}
          </button>
        </div>
      </div>
    </div>
  );
}
