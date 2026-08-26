"use client";

// Joining this check onto another one.
//
// ⚠️ **The mirror of splitting, and just as ordinary an evening.** Two friends
// at the counter move to a table; a couple is joined by four more; two tables
// push together for a birthday. A till that cannot do it makes the waiter read
// one check out loud while retyping it into another — which loses the times and
// the courses, and re-fires food the kitchen has already cooked.
//
// ⚠️ **The direction is stated in words, not implied by which check is open.**
// "Merge" with two names beside it is a control half the people who press it
// read backwards, and the wrong answer moves a paid-for dinner onto the wrong
// bill. This screen says whose food is moving and where it lands.

import { useState } from "react";
import { LuMerge } from "react-icons/lu";

import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import type { Check } from "@/lib/types";

export default function MergeDialog({
  check,
  others,
  currency,
  busy,
  onCancel,
  onMerge,
}: {
  check: Check;
  /** The other open checks — where this one can land. */
  others: Check[];
  currency: string;
  busy: boolean;
  onCancel: () => void;
  onMerge: (intoId: string) => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [target, setTarget] = useState("");

  const name = (c: Check) =>
    c.tableNumber
      ? `${c.tableNumber}-${t.till.table.toLowerCase()}`
      : t.till.counter;

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-ink/50 p-4 sm:items-center">
      <div className="till till-dialog flex max-h-[85vh] w-full max-w-md flex-col">
        <header className="shrink-0 border-b border-line px-4 py-3">
          <h2 className="flex items-center gap-2 text-lg font-bold">
            <LuMerge className="text-ink-muted" aria-hidden />
            {t.till.merge}
          </h2>
          {/* Which check is moving, with its total, so the sentence on screen
              is the one the waiter would say at the table. */}
          <p className="mt-0.5 text-[13px] text-ink-muted">
            {t.till.mergeHint
              .replace("{from}", name(check))
              .replace("{sum}", formatPrice(check.total, currency, lang))}
          </p>
        </header>

        <div className="min-h-0 flex-1 overflow-y-auto p-3">
          {others.length === 0 ? (
            <p className="text-sm text-ink-muted">{t.till.moveNoTarget}</p>
          ) : (
            <div className="grid grid-cols-3 gap-1.5">
              {others.map((c) => (
                <button
                  key={c.id}
                  onClick={() => setTarget(c.id)}
                  className={`min-h-14 rounded-[10px] border px-2 text-sm font-semibold transition ${
                    target === c.id
                      ? "border-[rgb(var(--till-accent))] bg-[rgb(var(--till-accent-tint))] text-[rgb(var(--till-accent-ink))]"
                      : "border-line bg-surface hover:border-line-strong"
                  }`}
                >
                  {name(c)}
                  <span className="till-num mt-0.5 block text-[12px] font-normal text-ink-muted">
                    {formatPrice(c.total, currency, lang)}
                  </span>
                </button>
              ))}
            </div>
          )}
        </div>

        <footer className="flex shrink-0 gap-2 border-t border-line p-3">
          <button className="till-btn flex-1" onClick={onCancel}>
            {t.till.back}
          </button>
          <button
            className="till-btn-accent flex-1"
            disabled={busy || !target}
            onClick={() => onMerge(target)}
          >
            {t.till.merge}
          </button>
        </footer>
      </div>
    </div>
  );
}
