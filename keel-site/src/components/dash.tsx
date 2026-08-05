"use client";

import type { Attention, TenantStatus } from "@/lib/api";
import type { Dict } from "@/lib/i18n/dict";

/** Shared dashboard bits. In their own module rather than a page file: a Next
 *  page may only export the page itself, and these are used by two of them. */

export function Field({
  label,
  hint,
  value,
  onChange,
}: {
  label: string;
  hint?: string;
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <div>
      <label className="text-sm font-medium">{label}</label>
      <input className="input mt-1" value={value} onChange={(e) => onChange(e.target.value)} />
      {hint && <p className="mt-1 text-xs text-ink-muted">{hint}</p>}
    </div>
  );
}

export function StatusBadge({ status, label }: { status: TenantStatus; label: string }) {
  const tone =
    status === "active"
      ? "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300"
      : status === "trial"
        ? "bg-signal-500/15 text-signal-600 dark:text-signal-400"
        : status === "deleted"
          ? "bg-ink/10 text-ink-muted"
          : "bg-rose-500/15 text-rose-700 dark:text-rose-300";
  return (
    <span className={`inline-block rounded-lg px-2 py-1 text-xs font-semibold ${tone}`}>
      {label}
    </span>
  );
}

export function statusLabel(s: TenantStatus, t: Dict): string {
  return s === "active"
    ? t.dash.active
    : s === "trial"
      ? t.dash.trial
      : s === "deleted"
        ? t.dash.deleted
        : t.dash.suspended;
}

/** What a customer needs from a human, said in words rather than in a date the
 *  reader has to subtract from today.
 *
 *  "Ended today" and "ends today" get their own sentences: "0 days" reads as
 *  nothing at all, which is the opposite of what it means. */
export function attentionLabel(a: Attention, t: Dict): string {
  const d = t.dash;
  switch (a.kind) {
    case "trial_ending":
      return a.days <= 0 ? d.badgeEndingToday : d.badgeEnding.replace("{n}", String(a.days));
    case "trial_expired":
      return a.days <= 0 ? d.badgeExpiredToday : d.badgeExpired.replace("{n}", String(a.days));
    case "suspended":
      return d.badgeUnpaid;
    default:
      return "";
  }
}

/** Deliberately louder than the status badge beside it. The status says what a
 *  customer *is*; this says somebody has to pick up the phone today. */
export function AttentionBadge({ attention, t }: { attention?: Attention; t: Dict }) {
  if (!attention?.kind) return null;
  const tone =
    attention.kind === "trial_ending"
      ? "bg-amber-500/15 text-amber-700 dark:text-amber-300"
      : "bg-rose-500/15 text-rose-700 dark:text-rose-300";
  return (
    <span className={`inline-block rounded-lg px-2 py-1 text-xs font-semibold ${tone}`}>
      {attentionLabel(attention, t)}
    </span>
  );
}
