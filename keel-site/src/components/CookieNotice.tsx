"use client";

// The cookie notice on keel.uz.
//
// ⚠️ **Copied rather than imported**, like the logo mark and the charts: keel.uz and the
// restaurant app are separate builds sharing no code, and a package for one component would be
// a dependency to keep in step for years.
//
// ⚠️ **The number of buttons follows what is actually installed.** This started as one button,
// and the reasoning was written down: this page had no counter, no analytics and no third-party
// script, so a "decline" would have run exactly the same code as "accept" — and a button that
// changes nothing teaches people these buttons are decoration, which is precisely why nobody
// reads the one on the site where it does work.
//
// An advertising pixel changed the fact, so it changes the notice. When `ads` is on there are
// two buttons and declining genuinely means no script, no request to Meta, no identifier; when
// it is off the old single-button notice returns, still true. The rule is the same one as
// before — **the notice describes what the page does** — and the shape it produces just moved.

import { useEffect, useState } from "react";
import Link from "next/link";
import { useT } from "@/lib/i18n/client";
import { localePath } from "@/lib/i18n/url";
import { readConsent, writeConsent } from "@/lib/pixel";
import { CONSENT_EVENT } from "@/components/MetaPixel";

const SEEN_KEY = "cookie_notice_v1";

export default function CookieNotice({ ads = false }: { ads?: boolean }) {
  const { t, lang } = useT();
  // Only after mount: the server cannot know what this browser answered, and rendering it
  // during SSR would flash the banner at everybody who already has.
  const [show, setShow] = useState(false);

  useEffect(() => {
    if (ads) {
      setShow(readConsent() === "unanswered");
      return;
    }
    try {
      // "seen" is this page's answer; the other two are accepted here so that somebody who
      // answered on a restaurant site built from the same key is not asked twice.
      const v = window.localStorage.getItem(SEEN_KEY);
      if (v !== "seen" && v !== "accepted" && v !== "declined") setShow(true);
    } catch {
      // Storage refused — shown again next visit, which is the honest failure.
      setShow(true);
    }
  }, [ads]);

  if (!show) return null;

  function answer(value: "granted" | "denied") {
    writeConsent(value);
    // Same visit, not the next one: on a single-page landing most visitors never navigate,
    // so waiting for a page load would lose almost everybody who said yes.
    window.dispatchEvent(new Event(CONSENT_EVENT));
    setShow(false);
  }

  function dismiss() {
    try {
      window.localStorage.setItem(SEEN_KEY, "seen");
    } catch {
      /* nothing to remember it with */
    }
    setShow(false);
  }

  return (
    <div className="fixed inset-x-0 bottom-0 z-50 p-3 sm:p-4">
      <div className="mx-auto flex max-w-3xl flex-col gap-3 rounded-2xl border border-line bg-surface p-4 shadow-xl shadow-hull-950/15 sm:flex-row sm:items-center">
        <p className="flex-1 text-sm text-ink-soft">
          {ads ? t.cookies.textAds : t.cookies.text}{" "}
          <Link
            href={localePath(lang, "/privacy-policy")}
            className="font-semibold text-signal-600 hover:underline"
          >
            {t.cookies.more}
          </Link>
        </p>
        {ads ? (
          <div className="flex shrink-0 gap-2">
            {/* Decline first and equally weighted. A refusal hidden behind a ghost button in
                the corner is the pattern that made everybody stop reading these. */}
            <button
              type="button"
              onClick={() => answer("denied")}
              className="rounded-xl border border-line-strong px-5 py-2 text-sm font-semibold text-ink"
            >
              {t.cookies.decline}
            </button>
            <button
              type="button"
              onClick={() => answer("granted")}
              className="rounded-xl bg-ink px-5 py-2 text-sm font-semibold text-surface"
            >
              {t.cookies.accept}
            </button>
          </div>
        ) : (
          <button
            type="button"
            onClick={dismiss}
            className="shrink-0 rounded-xl bg-ink px-5 py-2 text-sm font-semibold text-surface"
          >
            {t.cookies.ok}
          </button>
        )}
      </div>
    </div>
  );
}
