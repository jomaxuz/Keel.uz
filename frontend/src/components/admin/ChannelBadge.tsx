"use client";

// Where an order came from: the site, Telegram, or an operator's keyboard.
//
// ⚠️ Worth a badge rather than a column in a report, because the question it
// answers is asked while looking at the order: a guest who ordered through the
// bot is reachable through the bot, and one who ordered on the site is not. It
// also answers the question a restaurant paying for a bot actually has — "does
// anybody use it?" — and "half our orders come from Telegram" and "nobody has
// ever used it" lead to opposite decisions.
//
// Absent on orders placed before the field existed, and that renders nothing at
// all rather than "unknown": a label that says nothing is worse than no label,
// because it looks like a fact about the order.

import { useAdminT } from "@/lib/i18n/admin";

export default function ChannelBadge({ channel }: { channel?: string }) {
  const t = useAdminT();
  if (!channel) return null;

  if (channel === "telegram") {
    return (
      <span className="badge inline-flex items-center gap-1 bg-sky-100 text-sky-700 dark:bg-sky-500/15 dark:text-sky-300">
        <svg viewBox="0 0 24 24" fill="currentColor" className="h-3 w-3" aria-hidden>
          <path d="M21.9 4.3 19 19.2c-.2 1-.8 1.2-1.6.8l-4.4-3.3-2.1 2c-.2.3-.5.4-.8.4l.3-4.4 8.1-7.3c.3-.3 0-.5-.5-.2L8 12.1l-4.3-1.3c-.9-.3-.9-.9.2-1.3l16.5-6.4c.8-.3 1.5.2 1.5 1.2Z" />
        </svg>
        {t.calls.channelTelegram}
      </span>
    );
  }
  if (channel === "operator") {
    return (
      <span className="badge bg-violet-100 text-violet-700 dark:bg-violet-500/15 dark:text-violet-300">
        {t.calls.channelOperator}
      </span>
    );
  }
  // The site: deliberately the quiet one. It is the common case, and a loud
  // badge on every second row stops any of them being read.
  return <span className="badge bg-ink/5 text-ink-muted">{t.calls.channelWeb}</span>;
}
