"use client";

// The shared body of the 404 and the error page.
//
// ⚠️ **A client component, and that is what makes it translated.** A guest
// reaches these pages by any route — a stale QR card, a mistyped address, a link
// from a friend — and the one thing we know about them is the language they were
// already reading, which the root layout has already resolved into the provider.
// Reading it again from headers here would make the not-found route dynamic for
// every visitor to buy nothing.
//
// Both pages get a way out. A dead end with nothing but an apology on it is
// where a guest decides the restaurant is broken and closes the tab; the menu is
// two taps away and it is the thing they came for.

import LocaleLink from "@/components/site/LocaleLink";
import { useI18n } from "@/lib/i18n/client";
import { useSiteWords } from "@/lib/business";

export default function DeadEnd({
  art,
  title,
  text,
  code,
  onRetry,
}: {
  art: React.ReactNode;
  title: string;
  text: string;
  /** A digest or status the app can hand a guest to read out. Absent on a 404 —
   *  there is nothing to trace. */
  code?: string;
  /** Only the error page can offer this: a 404 does not become a 200 by asking
   *  again, and a button that never works teaches people to stop pressing. */
  onRetry?: () => void;
}) {
  // The catalogue's own address — see lib/siteWords.ts.
  const w = useSiteWords();
  const { t } = useI18n();

  return (
    <main className="container-page flex min-h-[70vh] flex-col items-center justify-center py-16 text-center">
      <div className="w-full max-w-[260px] text-ink-muted">{art}</div>

      <h1 className="mt-8 font-display text-2xl font-bold tracking-tight sm:text-3xl">
        {title}
      </h1>
      <p className="mt-3 max-w-md text-sm leading-relaxed text-ink-muted">{text}</p>

      <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
        {onRetry && (
          <button type="button" onClick={onRetry} className="btn-primary px-6 py-3">
            {t.errors.retry}
          </button>
        )}
        {/* ⚠️ LocaleLink, not next/link: on `/ru/...` a bare href would drop the
            prefix and answer a Russian guest's 404 by switching them to Uzbek. */}
        <LocaleLink
          href={w.href}
          className={onRetry ? "btn btn-ghost px-6 py-3" : "btn-primary px-6 py-3"}
        >
          {t.errors.menu}
        </LocaleLink>
        <LocaleLink href="/" className="btn btn-ghost px-6 py-3">
          {t.errors.home}
        </LocaleLink>
      </div>

      {code && (
        // Small, last, and selectable. It means nothing to the guest and
        // everything to whoever they read it out to.
        <p className="mt-8 select-all font-mono text-xs text-ink-muted/70">
          {t.errors.code}: {code}
        </p>
      )}
    </main>
  );
}
