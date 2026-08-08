"use client";

// The mini app's first screen: which language do you read?
//
// **Why this screen exists at all.** On the open web the language is carried by
// the URL (`/ru/menu`) and remembered in a cookie, and both are set the moment a
// visitor uses the switch in the header. Inside Telegram neither is available on
// arrival: a guest opening the bot for the first time — or scanning a table QR
// that launches it — has no cookie and no address bar. Without this screen the
// app opens in Uzbek for everybody, and the language switch is something they
// have to find *while reading a menu they cannot read*.
//
// ⚠️ **Telegram's own UI language is not a substitute**, which is why it is used
// only as the fallback for the bot's messages and never to skip this screen. It
// is a guess about the phone, not an answer about the person, and it is wrong for
// exactly the guests who would notice: the Uzbek speaker running an English
// phone, and the Russian speaker whose Telegram was set up by their son.
//
// ⚠️ **Nothing here is translated, and that is the design.** A heading that says
// "Choose your language" in one language is the same bug as an app that opens in
// one language. So the three options are labelled in their own language and
// script — the only text that is guaranteed readable by the person who needs to
// read it — and the heading is a globe rather than a sentence.
//
// Shown once per account: the answer is stored server-side (see
// handlers/userlang.go), because the thing that needs it most is a **bot
// message** sent hours later, with no browser to read a cookie.

import { useEffect, useRef } from "react";
import { useSearchParams } from "next/navigation";
import { useI18n } from "@/lib/i18n/client";
import { LANGS, LANG_LABEL, LANG_SHORT, type Lang } from "@/lib/i18n";
import { useTelegram } from "@/lib/telegram";

export default function TelegramLangGate() {
  const { inTelegram, needsLang, saveLang } = useTelegram();
  const { lang, setLang } = useI18n();
  const params = useSearchParams();
  // ⚠️ `lc=1` means the bot already asked, in the chat, before the app opened.
  //
  // Without this the guest taps a language on a button and is asked the identical
  // question one second later by the app — which does not read as thoroughness,
  // it reads as the first answer having been ignored. The language itself is
  // already correct here: the bot's link opens `/ru/menu`, so `lang` below is
  // what they picked.
  const chosenInChat = params.get("lc") === "1";
  const stored = useRef(false);

  useEffect(() => {
    if (!inTelegram || !needsLang || !chosenInChat || stored.current) return;
    stored.current = true;
    void saveLang(lang);
  }, [inTelegram, needsLang, chosenInChat, lang, saveLang]);

  if (!inTelegram || !needsLang || chosenInChat) return null;

  function choose(l: Lang) {
    // ⚠️ Two writes, deliberately, and the local one goes first.
    //
    // `setLang` is the site's existing switch: it sets the cookie and navigates
    // to the language's own URL, which is what makes the *next server render*
    // come back translated. `saveLang` is what makes the choice outlive this
    // WebView and reach the bot. Doing only the first means the guest gets an
    // Uzbek message about their order; doing only the second leaves them looking
    // at the Uzbek page they just tried to leave.
    setLang(l);
    void saveLang(l);
  }

  return (
    <div
      // ⚠️ `--tg-viewport`, not `100vh`. Inside Telegram the WebView reports the
      // whole screen, so a `100vh` overlay puts its lowest button underneath
      // Telegram's own chrome — and the lowest button here is English.
      style={{ height: "var(--tg-viewport, 100vh)" }}
      className="fixed inset-0 z-[100] flex flex-col items-center justify-center gap-6 bg-cream px-6"
      role="dialog"
      aria-modal="true"
      aria-label="Language / Til / Язык"
    >
      <svg
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
        className="h-12 w-12 text-brand"
        aria-hidden
      >
        <circle cx="12" cy="12" r="9" />
        <path d="M3 12h18M12 3c2.5 2.7 2.5 15.3 0 18M12 3c-2.5 2.7-2.5 15.3 0 18" />
      </svg>

      {/* All three, on one line. Naming the languages in their own scripts is the
          only heading that does not pick a winner before the guest has. */}
      <p className="text-center text-sm font-semibold text-ink-soft">
        Tilni tanlang · Выберите язык · Choose language
      </p>

      <div className="flex w-full max-w-xs flex-col gap-3">
        {LANGS.map((l) => (
          <button
            key={l}
            type="button"
            onClick={() => choose(l)}
            // Full width and tall: this is the first thing a thumb touches in the
            // app, often one-handed on a phone in a restaurant.
            className="flex min-h-14 w-full items-center justify-between gap-3 rounded-2xl border border-line bg-surface px-5 py-4 text-base font-bold text-ink transition-colors active:border-brand active:bg-brand-tint"
          >
            <span>{LANG_LABEL[l]}</span>
            <span className="text-xs font-bold text-ink-muted">
              {LANG_SHORT[l]}
            </span>
          </button>
        ))}
      </div>
    </div>
  );
}
