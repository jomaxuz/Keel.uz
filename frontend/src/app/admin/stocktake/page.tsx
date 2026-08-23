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
      .catch(() => setError(t.common.loadFailed));
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
      counted: Number(counted[r.ingredientId]) || 0,
    }));
  // Whether anything the person typed disagrees with the books — which is what
  // makes the note required, so the form must know it before the server says so.
  const off = sheet.some((r) => {
    const v = counted[r.ingredientId];
    return v !== undefined && v !== "" && Number(v) !== r.expected;
  });

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
                <th className="px-3 py-2 text-right">{t.stocktake.expected}</th>
                <th className="px-3 py-2 text-right">{t.stocktake.diff}</th>
              </tr>
            </thead>
            <tbody>
              {sheet.map((r) => {
                const raw = counted[r.ingredientId];
                const typed = raw !== undefined && raw !== "";
                const diff = typed ? Number(raw) - r.expected : 0;
                return (
                  <tr key={r.ingredientId} className="border-t border-line">
                    <td className="px-3 py-2">
                      {r.name}
                      <span className="ml-1 text-xs text-ink-muted">
                        {t.ingredients.units[r.unit] ?? r.unit}
                      </span>
                    </td>
                    <td className="px-3 py-2 text-right">
                      <input
                        type="number"
                        step="any"
                        min={0}
                        className="input w-28 py-1 text-right"
                        value={raw ?? ""}
                        onChange={(e) =>
                          setCounted({
                            ...counted,
                            [r.ingredientId]: e.target.value,
                          })
                        }
                      />
                    </td>
                    {/* ⚠️ Hidden until something is typed: beside an empty box
                        this number invites being copied, and a count that
                        agrees because it was read off the screen records
                        nothing. */}
                    <td className="px-3 py-2 text-right tabular-nums text-ink-muted">
                      {typed ? r.expected : "—"}
                    </td>
                    <td
                      className={`px-3 py-2 text-right tabular-nums ${
                        typed && diff !== 0 ? "text-danger" : "text-ink-muted"
                      }`}
                    >
                      {typed ? Math.round(diff * 1000) / 1000 : "—"}
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
          {/* Required by the server too, and for the reason the cash drawer
              requires one: a variance nobody explained is a variance nobody
              can use, and the explanation only exists on the day. */}
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">
              {off ? t.stocktake.noteRequired : t.stocktake.note}
            </span>
            <input
              className="input mt-1 w-full"
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
          </label>
          <button
            className="btn-primary px-4 py-2"
            disabled={busy || (off && !note.trim())}
            onClick={save}
          >
            {t.stocktake.save(lines.length)}
          </button>
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
