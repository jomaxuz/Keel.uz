"use client";

// Customer segments, as the panel shows them.
//
// The order here is the order they appear everywhere: the groups that need
// doing something about come first. "Sleeping" leads because it is the one
// that makes money — someone who used to order and stopped is far cheaper to
// bring back than a stranger is to find.
//
// The colours are not decoration either: amber for the groups that are a
// problem, green for the ones that are going well, plain for the rest.

import { useAdminT } from "@/lib/i18n/admin";
import type { CustomerSegment } from "@/lib/types";

export const SEGMENTS: CustomerSegment[] = [
  // An unanswered complaint outranks everything: it is the restaurant's own
  // doing, and it is the one that walks out of the door if left alone.
  "unhappy",
  "sleeping",
  "birthday",
  "vip",
  "regular",
  "new",
  "lost",
  "noOrders",
];

const TONE: Record<CustomerSegment, string> = {
  unhappy: "bg-rose-500/20 text-rose-700 dark:text-rose-300",
  sleeping: "bg-amber-500/15 text-amber-700 dark:text-amber-300",
  lost: "bg-rose-500/15 text-rose-700 dark:text-rose-300",
  birthday: "bg-fuchsia-500/15 text-fuchsia-700 dark:text-fuchsia-300",
  vip: "bg-brand/15 text-brand-dark",
  regular: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300",
  new: "bg-sky-500/15 text-sky-700 dark:text-sky-300",
  noOrders: "bg-ink/5 text-ink-muted",
};

export function SegmentBadge({ segment }: { segment: CustomerSegment }) {
  const t = useAdminT();
  return (
    <span
      title={t.users.segHint[segment]}
      className={`rounded-full px-2 py-0.5 text-xs font-semibold ${TONE[segment]}`}
    >
      {t.users.segment[segment]}
    </span>
  );
}
