"use client";

// Shelf labels and barcode stickers.
//
// ⚠️ **A shop cannot sell what it cannot scan.** Half of what a shop stocks
// arrives with a barcode printed by whoever made it; the other half — repacked,
// weighed out, baked in the back, bought at a market stall — arrives with
// nothing, and a till has no way to ring it up. Printing a code is not a
// convenience there: it is the difference between a product existing at the
// counter and not.
//
// ⚠️ **And a price on a shelf is the law's business.** A label is wrong the
// moment somebody changes a price, and until this screen existed nothing said
// which ones had gone stale — so the answer was a shop reprinting everything or
// reprinting nothing. The list leads with what is wrong rather than with the
// catalogue, because that is the question somebody opens this screen holding.
//
// ⚠️ **This is not Asl Belgisi.** The state's marking codes are issued to a
// producer or an importer, per unit and for money; a shop that resells scans
// them and never invents one. What is printed here is the shop's own code, in
// the range GS1 reserves for exactly that.

import { useCallback, useEffect, useMemo, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import { useAsk } from "@/components/ui/Ask";
import { ListScroll } from "@/components/admin/PagedList";
import type { StaleLabel } from "@/lib/types";

export default function AdminLabelsPage() {
  const t = useAdminT();
  const { tell } = useAsk();
  const scope = useAdminScope();

  const [rows, setRows] = useState<StaleLabel[] | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  /** How many stickers per product. ⚠️ Kept as text: an emptied box has to stay
   *  empty rather than snapping back to 1 while somebody is typing 12. */
  const [copies, setCopies] = useState<Record<string, string>>({});
  const [picked, setPicked] = useState<Set<string>>(new Set());

  const load = useCallback(async () => {
    try {
      const res = await api.adminStaleLabels();
      setRows(res.items);
      setError("");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
      setRows([]);
    }
  }, [t.common.loadFailed]);

  useEffect(() => {
    void load();
  }, [load, scope.scopeKey]);

  const list = rows ?? [];
  const chosen = useMemo(
    () => list.filter((r) => picked.has(r.id)),
    [list, picked],
  );

  function toggle(id: string) {
    setPicked((cur) => {
      const next = new Set(cur);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  async function print(items: StaleLabel[]) {
    if (items.length === 0) return;
    setBusy(true);
    try {
      const res = await api.adminPrintLabels(
        items.map((r) => ({
          id: r.id,
          copies: Math.max(1, Number(copies[r.id]) || 1),
        })),
      );
      // ⚠️ **What had to be invented is said out loud.** A barcode allocated
      // here is a fact somebody may need — it is going onto a packet, and the
      // person who pressed the button is the one who will stick it there.
      const made = res.barcoded ?? [];
      void tell({
        title: t.labels.queued(res.queued),
        body: made.length > 0 ? t.labels.barcoded(made.join(", ")) : undefined,
      });
      setPicked(new Set());
      await load();
    } catch (e) {
      void tell({ title: e instanceof ApiError ? e.message : t.common.loadFailed });
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">{t.labels.title}</h1>
          <p className="text-xs text-ink-muted">{t.labels.hint}</p>
        </div>
        {chosen.length > 0 && (
          <button
            className="btn-primary px-4 py-2 text-sm"
            disabled={busy}
            onClick={() => void print(chosen)}
          >
            {t.labels.printChosen(chosen.length)}
          </button>
        )}
      </div>

      {error !== "" && <p className="text-sm text-danger">{error}</p>}

      {rows !== null && list.length === 0 && (
        // ⚠️ Said as a state rather than left blank: an empty screen on a shop's
        // panel reads as one that did not load.
        <p className="rounded-2xl border border-line p-6 text-center text-sm text-ink-muted">
          {t.labels.allGood}
        </p>
      )}

      <ListScroll>
        <div className="space-y-2">
          {list.map((row) => {
            const on = picked.has(row.id);
            return (
              <div
                key={row.id}
                className={`flex flex-wrap items-center gap-3 rounded-xl border p-3 ${
                  on ? "border-brand bg-brand/5" : "border-line"
                }`}
              >
                <input
                  type="checkbox"
                  className="h-4 w-4"
                  checked={on}
                  onChange={() => toggle(row.id)}
                />
                <div className="min-w-40 flex-1">
                  <p className="text-sm font-medium">{row.name}</p>
                  <p className="text-xs text-ink-muted">
                    {/* ⚠️ The reason, in words that name what to do about it —
                        "no barcode" and "the price moved" are different jobs
                        for different people. */}
                    {t.labels.reason[row.reason]}
                    {row.reason === "price" && row.wasPrice ? (
                      <>
                        {" · "}
                        <span className="line-through">
                          {formatPrice(row.wasPrice)}
                        </span>
                        {" → "}
                        {formatPrice(row.price)}
                      </>
                    ) : (
                      ` · ${formatPrice(row.price)}`
                    )}
                    {row.barcode ? ` · ${row.barcode}` : ""}
                  </p>
                </div>
                <label className="flex items-center gap-2 text-xs text-ink-muted">
                  {t.labels.copies}
                  <input
                    className="input w-16 text-center"
                    inputMode="numeric"
                    value={copies[row.id] ?? ""}
                    placeholder="1"
                    onChange={(e) =>
                      setCopies((c) => ({
                        ...c,
                        [row.id]: e.target.value.replace(/\D/g, ""),
                      }))
                    }
                  />
                </label>
                <button
                  className="btn-ghost px-3 py-2 text-sm"
                  disabled={busy}
                  onClick={() => void print([row])}
                >
                  {t.labels.print}
                </button>
              </div>
            );
          })}
        </div>
      </ListScroll>
    </div>
  );
}
