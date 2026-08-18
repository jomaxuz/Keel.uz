"use client";

// Splitting a table's bill by guest.
//
// ⚠️ **The selected tab is where the next dish goes.** That is the whole
// mechanism, and it is why splitting is something a waiter does *while* taking
// the order rather than a sorting exercise at the end: they are already asking
// "and for you?" around the table, and the tab follows the question.
//
// ⚠️ **"Stol" is a real choice, not an empty state.** Most meals are one bill,
// and a screen that made every check start unsplit-but-not-yet-assigned would
// put a decision in front of every order that almost nobody needs to make.

import { useAdminT } from "@/lib/i18n/admin";
import type { CheckLine } from "@/lib/types";

export default function GuestTabs({
  lines,
  guests,
  value,
  onPick,
  onAdd,
}: {
  lines: CheckLine[];
  /** How many guests the check was opened for — the tabs start from that. */
  guests: number;
  /** 0 = the whole table. */
  value: number;
  onPick: (guest: number) => void;
  onAdd: () => void;
}) {
  const t = useAdminT();

  // ⚠️ Drawn from the check as well as from the head count: a table opened for
  // two that ends up splitting three ways must not lose the third tab, and a
  // line already assigned to a guest has to have somewhere to be shown.
  const highest = lines.reduce((max, l) => Math.max(max, l.guest ?? 0), 0);
  const count = Math.max(guests > 1 ? guests : 0, highest);
  if (count === 0) return null;

  const sums = new Map<number, number>();
  for (const l of lines) {
    if (l.void) continue;
    const g = l.guest ?? 0;
    sums.set(g, (sums.get(g) ?? 0) + l.sum);
  }

  return (
    <div className="no-scrollbar flex shrink-0 items-center gap-1.5 overflow-x-auto border-b border-line py-2 pl-2.5 pr-3">
      <button
        className={`${value === 0 ? "till-seg-on" : "till-seg"} shrink-0`}
        onClick={() => onPick(0)}
      >
        {t.till.wholeTable}
      </button>
      {Array.from({ length: count }, (_, i) => i + 1).map((g) => (
        <button
          key={g}
          className={`${value === g ? "till-seg-on" : "till-seg"} shrink-0`}
          onClick={() => onPick(g)}
        >
          {t.till.guestTab} {g}
          {/* What this guest owes, on the tab: the number the waiter is asked
              for out loud, one guest at a time, at the end of the meal. */}
          {sums.get(g) ? (
            <span className="till-num text-[11px] opacity-60">
              {Math.round((sums.get(g) ?? 0) / 1000)}k
            </span>
          ) : null}
        </button>
      ))}
      <button
        className="till-btn h-10 w-10 shrink-0 px-0 text-base"
        onClick={onAdd}
        aria-label={t.till.addGuest}
        title={t.till.addGuest}
      >
        +
      </button>
    </div>
  );
}
