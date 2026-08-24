"use client";

// The red corner on a counter whose subscription is running out.
//
// ⚠️ **It is in the corner and it is small, and both are deliberate.** The
// screens this lands on are the busiest in the building: a cashier is talking
// to a guest, the dish grid needs every row it can get, and a banner across the
// top would be dismissed on the first shift and never read again. What it has
// to achieve is narrow — that somebody in the restaurant learns, before the
// day it matters, that a date is coming — and a corner badge seen twenty times
// a shift does that better than a modal seen once and closed.
//
// ⚠️ **It never blocks anything.** The counter keeps selling after the date
// passes, and this component has no dismiss button and no overlay. Switching a
// till off is not a lever, it is taking a hostage: a restaurant whose register
// dies on a Friday evening cannot take money at all, and the pressure lands on
// the guest rather than on the owner who owes us. The panel and the reports are
// what close — see the platform's own policy — and the counter only ever warns.
//
// ⚠️ **No dismiss, and no "remind me later".** The countdown is a fact about a
// date, so a dismissal would be a claim that stops being true tomorrow. It
// disappears on its own when somebody pays, which is the only honest way for it
// to go away.

import { useAdminT } from "@/lib/i18n/admin";
import type { SubscriptionNotice } from "@/lib/types";

/** Where this is being drawn, which is the only thing that changes about it.
 *  The lock screen has room for a sentence; the chrome bar on a 1024px
 *  monoblock has room for a word and a number. */
type Size = "badge" | "full";

export default function SubscriptionCorner({
  notice,
  size = "badge",
}: {
  notice?: SubscriptionNotice | null;
  size?: Size;
}) {
  const t = useAdminT();
  // The common case, and it must cost nothing.
  if (!notice) return null;

  const expired = notice.level === "expired";
  // ⚠️ Amber at a week, red from three days. A week out is information — there
  // is time to arrange a payment — and painting it red spends the colour that
  // has to still mean something on the last day.
  const urgent = notice.level !== "warn";

  const tone = urgent
    ? "border-rose-300 bg-rose-50 text-rose-700"
    : "border-amber-300 bg-amber-50 text-amber-800";

  // ⚠️ **In the language of the screen, like everything else on it.** This was
  // three Uzbek strings written into the component — the one notice on the till
  // that a Russian-speaking cashier could not read, and the one that has to be
  // acted on by somebody who is not in the room.
  const text = expired
    ? t.till.subExpired
    : notice.days <= 0
      ? t.till.subToday
      : t.till.subDays(notice.days);

  if (size === "full") {
    return (
      <div
        // `role="status"` rather than `alert`: an alert interrupts a screen
        // reader mid-sentence, and this is not an interruption — the cashier is
        // mid-transaction and nothing here needs doing this second.
        role="status"
        className={`rounded-xl border px-3 py-2 text-[13px] font-semibold ${tone}`}
      >
        <span className="flex items-center gap-2">
          <Dot urgent={urgent} />
          {text}
        </span>
        <span className="mt-0.5 block text-[11px] font-normal opacity-80">
          {notice.until} · {t.till.subTellOwner}
        </span>
      </div>
    );
  }

  return (
    <span
      role="status"
      // ⚠️ `shrink-0` and no wrap: this sits in a header that already truncates
      // a long staff name to keep the clock and the lock button on screen, and
      // a badge that grows would push one of them off a 1024px monoblock.
      className={`flex shrink-0 items-center gap-1.5 whitespace-nowrap rounded-full border px-2.5 py-1 text-[12px] font-semibold ${tone}`}
      title={`${text} · ${notice.until}`}
    >
      <Dot urgent={urgent} />
      {expired || notice.days <= 0 ? text : `${notice.days} kun`}
    </span>
  );
}

function Dot({ urgent }: { urgent: boolean }) {
  return (
    <span
      aria-hidden
      className={`h-2 w-2 shrink-0 rounded-full ${
        urgent ? "bg-rose-500" : "bg-amber-500"
      }`}
    />
  );
}
