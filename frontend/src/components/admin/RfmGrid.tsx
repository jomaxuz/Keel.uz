"use client";

// The customer base ranked against itself.
//
// The rule segments above this panel answer "who crossed a line". This answers
// "who matters, relative to everybody else" — and the two disagree in useful
// ways: a restaurant that raised its prices 30% has not acquired a room full of
// big spenders, though a fixed money threshold says it has.
//
// ⚠️ **The row worth building the screen around is `atRisk`**: customers who
// used to order often and have gone quiet. Nothing else in the panel can find a
// regular in the act of leaving, and by the time the 180-day "lost" rule catches
// them there is nothing left to do about it.

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import type { RfmResponse } from "@/lib/types";

type SegKey = keyof ReturnType<typeof useAdminT>["users"]["segment"];

export default function RfmGrid({ onPick }: { onPick: (segment: string) => void }) {
  const t = useAdminT();
  const [data, setData] = useState<RfmResponse | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    api
      .adminRfm()
      .then(setData)
      .catch(() => setFailed(true));
  }, []);

  if (failed) return null;
  if (!data) {
    return <p className="text-sm text-ink-muted/70">{t.common.loading}</p>;
  }

  if (!data.available) {
    // Named minimum, not an empty grid. "You need 10 customers, you have 4" is
    // a sentence somebody can act on; a grid of zeros reads as a restaurant
    // that lost everybody.
    return (
      <p className="text-sm text-ink-muted">
        {t.campaigns.rfm.tooSmall(data.base ?? 0, data.minBase ?? 0)}
      </p>
    );
  }

  const total = data.total ?? 0;
  const revenue = data.revenue ?? 0;

  return (
    <div className="space-y-3">
      <p className="text-xs text-ink-muted">
        {t.campaigns.rfm.hint}
        {data.scale && ` · ${t.campaigns.rfm.base(data.scale.base)}`}
      </p>

      <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
        {data.cells.map((c) => {
          const id = `rfm:${c.cell}`;
          const sharePeople = total ? (c.count / total) * 100 : 0;
          const shareMoney = revenue ? (c.revenue / revenue) * 100 : 0;
          return (
            <button
              key={c.cell}
              type="button"
              // An empty cell is not a choice — there is nobody to message —
              // but it stays on screen, because an empty "atRisk" is good news
              // and has to be visible as such.
              disabled={c.count === 0}
              onClick={() => onPick(id)}
              className={`rounded-2xl border p-3 text-left transition ${
                c.count === 0
                  ? "border-line opacity-50"
                  : c.cell === "atRisk"
                    ? "border-amber-500/40 bg-amber-500/5 hover:border-amber-500"
                    : "border-line hover:border-brand"
              }`}
            >
              <p className="text-sm font-semibold text-ink">
                {t.users.segment[id as SegKey] ?? c.cell}
              </p>
              <p className="mt-1 text-2xl font-bold tabular-nums text-ink">{c.count}</p>
              <p className="mt-0.5 text-xs text-ink-muted">
                {/* Two shares, never one. A cell holding a tenth of the
                    customers and half the takings is the entire reason to look
                    at this grid, and either number alone hides it. */}
                {sharePeople.toFixed(0)}% · {formatPrice(c.revenue)} ({shareMoney.toFixed(0)}%)
              </p>
              <p className="mt-1 text-[11px] leading-snug text-ink-muted/80">
                {t.users.segHint[id as SegKey] ?? ""}
              </p>
            </button>
          );
        })}
      </div>
    </div>
  );
}
