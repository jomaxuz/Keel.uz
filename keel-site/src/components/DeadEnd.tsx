"use client";

// The landing site's 404 and error pages.
//
// ⚠️ **Its own copy rather than an import from the tenant app**, the same
// decision as the charts and the logo badge: two builds, two audiences. The
// restaurant's 404 talks about the menu because a hungry guest is standing in
// front of it; this one talks about the platform, because whoever arrived here
// was deciding whether to hand us their restaurant's orders.
//
// The drawing is inline SVG in `currentColor` with one accent, so it follows the
// theme in both directions and cannot itself 404 on the page whose whole job is
// to survive a broken request.

import Link from "next/link";
import { useT } from "@/lib/i18n/client";
import { localePath } from "@/lib/i18n/url";

/** A ship's keel above the waterline: the brand's own shape, drawn as if the
 *  route ran aground. Chosen over a large "404" because the number says nothing
 *  a sentence does not, and this page is read by people who are not engineers. */
function LostArt({ className = "" }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 200 140"
      fill="none"
      stroke="currentColor"
      strokeWidth="2.2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden
    >
      {/* hull */}
      <path d="M42 74h116l-16 30a16 16 0 0 1-14 8H72a16 16 0 0 1-14-8z" />
      {/* mast and sail */}
      <path d="M100 74V22" />
      <path d="M100 30l30 30h-30" className="text-signal-500" />
      {/* water, broken in the middle — the one line that says something is off */}
      <path d="M22 120h44M84 120h26M128 120h50" className="text-ink/25" />
      <path d="M30 106h26M148 106h22" className="text-ink/20" />
    </svg>
  );
}

/** A plug pulled out of its socket: our side failed, and it is not the visitor's
 *  connection. */
function BrokenArt({ className = "" }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 200 140"
      fill="none"
      stroke="currentColor"
      strokeWidth="2.2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden
    >
      {/* socket half */}
      <path d="M20 70h34v22a14 14 0 0 0 14 14h6" />
      <path d="M20 58v24" />
      {/* plug half, pulled away */}
      <path d="M180 70h-34V48a14 14 0 0 0-14-14h-6" />
      <path d="M180 58v24" />
      {/* the gap */}
      <path d="M88 70h8M104 70h8" className="text-signal-500" />
      <path d="M100 44v-8M118 52l6-6M82 52l-6-6" className="text-signal-500/70" />
    </svg>
  );
}

export function LostPage() {
  const { t, lang } = useT();
  return (
    <Shell
      art={<LostArt className="w-full" />}
      title={t.errors.notFoundTitle}
      text={t.errors.notFoundText}
      links={[
        { href: localePath(lang, "/"), label: t.errors.home, primary: true },
        { href: localePath(lang, "/status"), label: t.errors.status },
      ]}
    />
  );
}

export function BrokenPage({
  digest,
  onRetry,
}: {
  digest?: string;
  onRetry: () => void;
}) {
  const { t, lang } = useT();
  return (
    <Shell
      art={<BrokenArt className="w-full" />}
      title={t.errors.serverTitle}
      text={t.errors.serverText}
      code={digest}
      onRetry={onRetry}
      retryLabel={t.errors.retry}
      links={[
        // ⚠️ The status page first, not the home page. Somebody who just hit an
        // error is asking "is this me or them", and that page answers it —
        // sending them to the marketing home instead is answering a question
        // they did not ask.
        { href: localePath(lang, "/status"), label: t.errors.status },
        { href: localePath(lang, "/"), label: t.errors.home },
      ]}
    />
  );
}

function Shell({
  art,
  title,
  text,
  links,
  code,
  onRetry,
  retryLabel,
}: {
  art: React.ReactNode;
  title: string;
  text: string;
  links: { href: string; label: string; primary?: boolean }[];
  code?: string;
  onRetry?: () => void;
  retryLabel?: string;
}) {
  return (
    <main className="mx-auto flex min-h-[70vh] max-w-2xl flex-col items-center justify-center px-6 py-20 text-center">
      <div className="w-full max-w-[260px] text-ink-muted">{art}</div>
      <h1 className="h-display mt-8 text-2xl sm:text-3xl">{title}</h1>
      <p className="mt-3 max-w-md text-sm leading-relaxed text-ink-muted">{text}</p>

      <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
        {onRetry && (
          <button
            type="button"
            onClick={onRetry}
            className="rounded-xl bg-ink px-6 py-3 text-sm font-semibold text-surface"
          >
            {retryLabel}
          </button>
        )}
        {links.map((l) => (
          <Link
            key={l.href}
            href={l.href}
            className={
              l.primary && !onRetry
                ? "rounded-xl bg-ink px-6 py-3 text-sm font-semibold text-surface"
                : "rounded-xl border border-line px-6 py-3 text-sm font-semibold text-ink-soft hover:text-ink"
            }
          >
            {l.label}
          </Link>
        ))}
      </div>

      {code && (
        <p className="mt-8 select-all font-mono text-xs text-ink-muted/70">{code}</p>
      )}
    </main>
  );
}
