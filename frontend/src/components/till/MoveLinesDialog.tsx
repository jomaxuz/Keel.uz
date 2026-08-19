"use client";

// Moving dishes onto another check.
//
// ⚠️ **A party that split, or joined.** Two friends sit down at the counter and
// then move to a table; four people turn out to be paying separately; a table
// of six becomes two of three. Every dining room does this several times an
// evening, and a till that cannot do it makes the waiter void the food and ring
// it in again — which throws away the times, the audit trail and, once it has
// been cooked, the money.
//
// ⚠️ **The first destination is a new check**, which is what "split the bill"
// is. Before it existed, a table paying separately had to be opened as two
// checks *before anybody ordered* — the waiter guessing at the door who would
// eat what — and the seat numbers the till already collects could not be acted
// on at all. It sits in the same dialog rather than behind its own button
// because the question before it is identical: which of these dishes.
//
// ⚠️ **Lines are ticked, not dragged.** The screen is used standing up by
// somebody holding a tray; a drag that starts on a scrolling list is a drag
// that scrolls the list.

import { useState } from "react";

import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import type { Check } from "@/lib/types";

export default function MoveLinesDialog({
  check,
  others,
  currency,
  busy,
  onCancel,
  onMove,
}: {
  check: Check;
  /** The other open checks — where the food can go. */
  others: Check[];
  currency: string;
  busy: boolean;
  onCancel: () => void;
  /** `toCheckId` is empty for "onto a new check" — the split. */
  onMove: (lineIds: string[], toCheckId: string) => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [picked, setPicked] = useState<string[]>([]);
  // "new" rather than "" so that nothing is chosen by default: a destination
  // preselected on a screen used at speed is a bill divided by accident.
  const [target, setTarget] = useState<string | null>(null);

  // ⚠️ Voided lines are not offered: the record of food written off belongs to
  // the check it was written off on, and carrying it across moves the blame.
  const live = check.lines.filter((l) => !l.void);

  function toggle(lineId: string) {
    setPicked((prev) =>
      prev.includes(lineId)
        ? prev.filter((id) => id !== lineId)
        : [...prev, lineId],
    );
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-ink/50 p-4 sm:items-center">
      <div className="till till-dialog flex max-h-[85vh] w-full max-w-lg flex-col">
        <header className="shrink-0 border-b border-line px-4 py-3">
          <h2 className="text-lg font-bold">{t.till.moveLines}</h2>
          <p className="mt-0.5 text-[13px] text-ink-muted">
            {t.till.moveLinesHint}
          </p>
        </header>

        <div className="min-h-0 flex-1 overflow-y-auto p-3">
          <ul className="space-y-1">
            {live.map((l) => {
              const on = picked.includes(l.lineId);
              return (
                <li key={l.lineId}>
                  <button
                    onClick={() => toggle(l.lineId)}
                    className={`flex w-full items-center gap-2.5 rounded-[10px] border px-2.5 py-2 text-left transition ${
                      on
                        ? "border-[rgb(var(--till-accent))] bg-[rgb(var(--till-accent-tint))]"
                        : "border-line bg-surface hover:border-line-strong"
                    }`}
                  >
                    <span
                      className={`flex h-5 w-5 shrink-0 items-center justify-center rounded-[6px] border text-[12px] font-bold ${
                        on
                          ? "border-transparent bg-[rgb(var(--till-accent))] text-ink"
                          : "border-line-strong text-transparent"
                      }`}
                      aria-hidden
                    >
                      ✓
                    </span>
                    <span className="till-num w-6 text-center text-[13px] font-bold">
                      {l.qty}
                    </span>
                    <span className="min-w-0 flex-1 truncate text-sm font-semibold">
                      {l.name}
                    </span>
                    {/* Whether the kitchen has it, because that is what makes
                        this a move rather than a re-order. */}
                    {l.fired && (
                      <span className="till-chip till-chip-info shrink-0">
                        {t.till.firedLabel}
                      </span>
                    )}
                    <span className="till-num shrink-0 text-[13px]">
                      {formatPrice(l.sum, currency, lang)}
                    </span>
                  </button>
                </li>
              );
            })}
          </ul>

          <h3 className="till-label mt-4">{t.till.moveTo}</h3>
          <div className="mt-2 grid grid-cols-3 gap-1.5">
            <button
              onClick={() => setTarget("new")}
              className={`min-h-12 rounded-[10px] border px-2 text-sm font-semibold transition ${
                target === "new"
                  ? "border-[rgb(var(--till-accent))] bg-[rgb(var(--till-accent-tint))] text-[rgb(var(--till-accent-ink))]"
                  : "border-line bg-surface hover:border-line-strong"
              }`}
            >
              {t.till.splitNew}
            </button>
            {others.map((c) => (
              <button
                key={c.id}
                onClick={() => setTarget(c.id)}
                className={`min-h-12 rounded-[10px] border px-2 text-sm font-semibold transition ${
                  target === c.id
                    ? "border-[rgb(var(--till-accent))] bg-[rgb(var(--till-accent-tint))] text-[rgb(var(--till-accent-ink))]"
                    : "border-line bg-surface hover:border-line-strong"
                }`}
              >
                {c.tableNumber
                  ? `${c.tableNumber}-${t.till.table.toLowerCase()}`
                  : t.till.counter}
              </button>
            ))}
          </div>
        </div>

        <footer className="flex shrink-0 gap-2 border-t border-line p-3">
          <button className="till-btn flex-1" onClick={onCancel}>
            {t.till.back}
          </button>
          <button
            className="till-btn-accent flex-1"
            disabled={busy || picked.length === 0 || !target}
            onClick={() => onMove(picked, target === "new" ? "" : target!)}
          >
            {target === "new" ? t.till.split : t.till.move}
          </button>
        </footer>
      </div>
    </div>
  );
}
