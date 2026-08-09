"use client";

// The cookie notice: one button, because there is one honest button.
//
// ⚠️ **This tells, it does not ask.** Everything the site stores is needed to run it — the
// language, the brand and branch, the cart, the session — plus an anonymous daily page count
// whose id is hashed with the day. A "decline" would run the same code as "accept", and a
// control that changes nothing teaches people that consent controls are decoration, which is
// exactly why nobody reads the ones that do work.
//
// ⚠️ **Not shown inside Telegram.** The mini app has one small viewport and Telegram's own
// chrome above it; a banner there costs the first screen of a menu to ask about storage the
// guest cannot see. The choice is still respected if it was made on the web.
//
// It never blocks the page: a bottom sheet that does not return once dismissed.

import { useEffect, useState } from "react";
import LocaleLink from "@/components/site/LocaleLink";
import { useI18n } from "@/lib/i18n/client";
import { useTelegram } from "@/lib/telegram";
import { markNoticeSeen, noticeSeen } from "@/lib/cookies";

export default function CookieNotice() {
  const { t } = useI18n();
  const { inTelegram } = useTelegram();
  // ⚠️ Rendered only after mount. The server has no idea what this browser has answered, so
  // drawing it during SSR would flash a banner at everybody who already answered — the
  // hydration trap this codebase has hit twice before.
  const [show, setShow] = useState(false);

  useEffect(() => {
    if (!noticeSeen()) setShow(true);
  }, []);

  if (!show || inTelegram) return null;

  function dismiss() {
    markNoticeSeen();
    setShow(false);
  }

  return (
    <div className="fixed inset-x-0 bottom-0 z-50 p-3 sm:p-4">
      <div className="card mx-auto flex max-w-3xl flex-col gap-3 p-4 shadow-card-hover sm:flex-row sm:items-center">
        <p className="flex-1 text-sm text-ink-soft">
          {t.cookies.text}{" "}
          <LocaleLink href="/about" className="font-semibold text-brand hover:underline">
            {t.cookies.more}
          </LocaleLink>
        </p>
        <button
          type="button"
          onClick={dismiss}
          className="btn btn-primary shrink-0 px-5 py-2 text-sm"
        >
          {t.cookies.ok}
        </button>
      </div>
    </div>
  );
}
