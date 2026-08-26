"use client";

// Counting the store.
//
// ⚠️ **The difference is the product**, exactly as it is for the cash drawer.
// A screen that shows what should be there, takes what was found and stores
// only the second number has recorded nothing: the shortfall it exists to
// surface has been overwritten by the person who might have caused it.
//
// ⚠️ So the expected column is **not shown while the counting is being typed**.
// Beside an empty box it invites being copied, and a count that agrees because
// it was read off the screen is the one record this whole thing exists to
// prevent. It appears after the number is entered — the same rule, and the same
// reason, as the cash drawer's.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatDate, formatPrice } from "@/lib/format";
import { qtyNumber } from "@/lib/qty";
import { QtyInput } from "@/components/QtyInput";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import { ListScroll } from "@/components/admin/PagedList";
import { WarehousePicker } from "@/components/admin/WarehousesEditor";
import type { Stocktake, StocktakeSheetRow, Warehouse } from "@/lib/types";

export default function StocktakePage() {
  const t = useAdminT();
  const scope = useAdminScope();
  const [sheet, setSheet] = useState<StocktakeSheetRow[]>([]);
  const [since, setSince] = useState<string | null>(null);
  const [counted, setCounted] = useState<Record<string, string>>({});
  const [note, setNote] = useState("");
  const [past, setPast] = useState<Stocktake[]>([]);
  const [explain, setExplain] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState<Stocktake | null>(null);
  const [warehouses, setWarehouses] = useState<Warehouse[]>([]);
  // ⚠️ **A count is one room.** Handing somebody walking into the bar a list
  // that also has forty kitchen ingredients on it is how a count gets abandoned
  // halfway and saved anyway — and a half-counted list saves nothing for
  // everything nobody reached, which reads as a catastrophic shortfall.
  const [warehouse, setWarehouse] = useState("");

  const load = useCallback(() => {
    api
      .adminStocktakeSheet(warehouse)
      .then((d) => {
        setSheet(d.rows);
        setSince(d.since);
      })
      .catch((e) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      );
    api
      .adminStocktakes()
      .then((d) => setPast(d.stocktakes))
      .catch(() => setPast([]));
  }, [t.common.loadFailed, warehouse]);

  useEffect(load, [load, scope.scopeKey]);

  useEffect(() => {
    api
      .adminWarehouses()
      .then((d) => setWarehouses(d.warehouses))
      .catch(() => setWarehouses([]));
  }, [scope.scopeKey]);

  const lines = sheet
    .filter(
      (r) =>
        counted[r.ingredientId] !== undefined && counted[r.ingredientId] !== "",
    )
    .map((r) => ({
      ingredientId: r.ingredientId,
      counted: qtyNumber(counted[r.ingredientId] ?? ""),
    }));
  // ⚠️ Whether anything disagrees is no longer knowable here, and it cannot be
  // made knowable without handing back the figures the count exists to test. A
  // form that could tell is a form that can be asked repeatedly until it says
  // no — see the note on the sheet type.

  // ⚠️ A variance with no explanation, which is only knowable once the count
  // exists. `value` is what it was out by in money — zero means it agreed.
  const owed = past.filter((s) => s.value !== 0 && !s.notedAt);

  async function saveExplain(id: string) {
    const note = (explain[id] ?? "").trim();
    if (!note) return;
    try {
      await api.adminExplainStocktake(id, note);
    } catch {
      // ⚠️ Swallowed on purpose: the only failure worth distinguishing is
      // "somebody else explained it first" (409), and the reload below shows
      // exactly that — their note, in place of the box.
    }
    setExplain({ ...explain, [id]: "" });
    api
      .adminStocktakes()
      .then((r) => setPast(r.stocktakes ?? []))
      .catch(() => {});
  }

  async function save() {
    setBusy(true);
    setError("");
    try {
      const res = await api.adminSaveStocktake({
        lines,
        warehouseId: warehouse,
        note: note.trim(),
      });
      setSaved(res);
      setCounted({});
      setNote("");
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t.stocktake.title}</h1>
        <p className="mt-1 text-sm text-ink-soft">
          {t.stocktake.intro}{" "}
          {/* ⚠️ Where the expected figure is measured from. Without this it
              reads as a stock balance the system has been keeping all along,
              which it is not — it is four recorded facts added together. */}
          {since
            ? t.stocktake.since(formatDate(since))
            : t.stocktake.sinceNever}
        </p>
      </div>

      {/* ⚠️ Which room is being counted, and the sheet follows it. The "since"
          line above follows it too: the bar is counted on a Sunday and the
          kitchen on a Wednesday, so one date for the branch would describe half
          the list wrongly. */}
      <WarehousePicker
        warehouses={warehouses}
        value={warehouse}
        onChange={(id) => {
          // ⚠️ The typed counts are dropped with the store. They belong to the
          // shelf somebody was standing at; carried across they would be saved
          // against a room nobody walked into.
          setCounted({});
          setWarehouse(id);
        }}
      />

      {error && <p className="text-sm text-danger">{error}</p>}
      {saved && (
        <p className="rounded-xl bg-ink/[0.04] px-3 py-2 text-sm">
          {t.stocktake.savedSummary(formatPrice(saved.value))}
        </p>
      )}

      <div className="card p-0">
        <ListScroll>
          <table className="w-full text-sm">
            <thead className="sticky top-0 bg-surface text-left text-xs text-ink-muted">
              <tr>
                <th className="px-3 py-2">{t.stocktake.ingredient}</th>
                <th className="px-3 py-2 text-right">{t.stocktake.counted}</th>
                {/* ⚠️ No expected column and no difference column while the
                    counting is being typed. They were drawn only after a number
                    appeared, which read as careful and was not: type anything,
                    read the figure, correct the entry. Both come back with the
                    saved count, when nothing can be moved. */}
              </tr>
            </thead>
            <tbody>
              {sheet.map((r) => {
                const raw = counted[r.ingredientId];
                return (
                  <tr key={r.ingredientId} className="border-t border-line">
                    <td className="px-3 py-2">
                      {r.name}
                      <span className="ml-1 text-xs text-ink-muted">
                        {t.ingredients.units[r.unit] ?? r.unit}
                      </span>
                    </td>
                    <td className="px-3 py-2 text-right">
                      <QtyInput
                        className="input w-28 py-1 text-right"
                        value={raw ?? ""}
                        onValue={(v) =>
                          setCounted({ ...counted, [r.ingredientId]: v })
                        }
                      />
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
          {sheet.length === 0 && (
            <p className="p-6 text-center text-sm text-ink-muted">
              {t.stocktake.empty}
            </p>
          )}
        </ListScroll>
      </div>

      {lines.length > 0 && (
        <div className="card space-y-2 p-3">
          {/* ⚠️ **Always offered, never demanded, and always labelled the
              same.** It used to say "required" when something disagreed — and
              that label was itself the answer: a counter watching the word
              change knows they have matched the books, without ever being shown
              a figure. If a variance is left unexplained the panel asks for it
              afterwards, against a count that can no longer be edited. */}
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.stocktake.note}</span>
            <input
              className="input mt-1 w-full"
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
          </label>
          <button
            className="btn-primary px-4 py-2"
            disabled={busy}
            onClick={save}
          >
            {t.stocktake.save(lines.length)}
          </button>
        </div>
      )}

      {/* ⚠️ **The question is asked here, after the count is locked, because
          asking it earlier would answer it.** A count is insert-only, so by the
          time this appears the numbers cannot be moved to make the variance go
          away — which is exactly why it is safe to say there is one. */}
      {owed.length > 0 && (
        <div className="card space-y-3 p-3">
          <h2 className="text-sm font-semibold">{t.stocktake.owedTitle}</h2>
          <p className="text-xs text-ink-muted">{t.stocktake.owedHint}</p>
          {owed.map((s) => (
            <div key={s.id} className="rounded-xl border border-line p-3">
              <div className="flex flex-wrap items-baseline justify-between gap-2 text-sm">
                <span>
                  {formatDate(s.at)} · {s.by}
                </span>
                <span
                  className={`tabular-nums ${s.value < 0 ? "text-danger" : ""}`}
                >
                  {formatPrice(s.value)}
                </span>
              </div>
              <div className="mt-2 flex gap-2">
                <input
                  className="input flex-1"
                  placeholder={t.stocktake.noteRequired}
                  value={explain[s.id] ?? ""}
                  onChange={(e) =>
                    setExplain({ ...explain, [s.id]: e.target.value })
                  }
                />
                <button
                  type="button"
                  className="btn btn-ghost"
                  disabled={!(explain[s.id] ?? "").trim()}
                  onClick={() => saveExplain(s.id)}
                >
                  {t.common.save}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {past.length > 0 && (
        <div className="card p-0">
          <ListScroll max="max-h-64">
            <table className="w-full text-sm">
              <tbody>
                {past.map((s) => (
                  <tr key={s.id} className="border-t border-line">
                    <td className="px-3 py-2 whitespace-nowrap">
                      {formatDate(s.at)}
                    </td>
                    <td className="px-3 py-2 text-ink-muted">
                      {s.lines.length} · {s.by}
                      {s.note ? ` · ${s.note}` : ""}
                    </td>
                    <td
                      className={`px-3 py-2 text-right tabular-nums ${
                        s.value < 0 ? "text-danger" : ""
                      }`}
                    >
                      {formatPrice(s.value)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </ListScroll>
        </div>
      )}
    </div>
  );
}
