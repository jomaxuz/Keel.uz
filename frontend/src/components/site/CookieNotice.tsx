"use client";

// The cookie notice, with a decline that does something.
//
// ⚠️ **Two buttons, and the second one is honest.** Declining cannot turn off the language,
// the cart or the session — the site does not work without them, and the notice says so
// instead of pretending. What it does turn off is the anonymous visit counter, which is the
// only optional thing this site stores. A banner whose "decline" changes nothing teaches
// people that consent controls are decoration.
//
// ⚠️ **Not shown inside Telegram.** The mini app has one small viewport and Telegram's own
// chrome above it; a banner there costs the first screen of a menu to ask about storage the
// guest cannot see. The choice is still respected if it was made on the web.
//
// It never blocks the page: a bottom sheet, dismissible either way, and it does not return
// once answered.

import { useEffect, useState } from "react";
import LocaleLink from "@/components/site/LocaleLink";
import { useI18n } from "@/lib/i18n/client";
import { useTelegram } from "@/lib/telegram";
import { readCookieChoice, writeCookieChoice } from "@/lib/cookies";

export default function CookieNotice() {
  const { t } = useI18n();
  const { inTelegram } = useTelegram();
  // ⚠️ Rendered only after mount. The server has no idea what this browser has answered, so
  // drawing it during SSR would flash a banner at everybody who already answered — the
  // hydration trap this codebase has hit twice before.
  const [show, setShow] = useState(false);

  useEffect(() => {
    if (readCookieChoice() === null) setShow(true);
  }, []);

  if (!show || inTelegram) return null;

  function answer(choice: "accepted" | "declined") {
    writeCookieChoice(choice);
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
        <div className="flex shrink-0 gap-2">
          <button
            type="button"
            onClick={() => answer("declined")}
            className="btn btn-ghost px-4 py-2 text-sm"
          >
            {t.cookies.decline}
          </button>
          <button
            type="button"
            onClick={() => answer("accepted")}
            className="btn btn-primary px-5 py-2 text-sm"
          >
            {t.cookies.accept}
          </button>
        </div>
      </div>
    </div>
  );
}
