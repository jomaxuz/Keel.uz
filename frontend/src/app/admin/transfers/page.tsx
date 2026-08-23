"use client";

// Stock moved from one store to another.
//
// ⚠️ **The movement the chain was missing, and the reason it matters is that
// the two workarounds both lie.** A barman taking a case of tonic out of the
// cellar could only be recorded as a write-off there and a delivery here. The
// write-off adds a reason to the waste report for food nobody wasted — on the
// report whose whole value is that its reasons are actionable — and the
// delivery writes a purchase price into the history, which then costs every
// dish the ingredient goes into.
//
// ⚠️ **From one ingredient to another**, because a store belongs to the
// ingredient: a restaurant keeping tonic in the cellar *and* behind the bar
// already has two of them, two shelves and two counts. See models/warehouse.go.

import { useCallback, useEffect, useMemo, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatDate, formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import { ListScroll } from "@/components/admin/PagedList";
import type { Ingredient, StockTransfer, Warehouse } from "@/lib/types";

function today() {
  const d = new Date();
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

function monthStart() {
  const d = new Date();
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-01`;
}

export default function TransfersPage() {
  const t = useAdminT();
  const scope = useAdminScope();
  const [rows, setRows] = useState<StockTransfer[]>([]);
  const [moved, setMoved] = useState(0);
  const [from, setFrom] = useState(monthStart);
  const [to, setTo] = useState(today);
  const [ingredients, setIngredients] = useState<Ingredient[]>([]);
  const [stores, setStores] = useState<Warehouse[]>([]);
  const [at, setAt] = useState(today);
  const [fromId, setFromId] = useState("");
  const [toId, setToId] = useState("");
  const [qty, setQty] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    api
      .adminTransfers({ from, to })
      .then((d) => {
        setRows(d.transfers);
        setMoved(d.moved);
      })
      .catch(() => setError(t.common.loadFailed));
    api
      .adminIngredients()
      .then((d) => setIngredients(d.ingredients))
      .catch(() => setIngredients([]));
    api
      .adminWarehouses()
      .then((d) => setStores(d.warehouses))
      .catch(() => setStores([]));
  }, [t.common.loadFailed, from, to]);

  useEffect(load, [load, scope.scopeKey]);

  const byId = useMemo(() => {
    const m = new Map<string, Ingredient>();
    for (const i of ingredients) m.set(i.id, i);
    return m;
  }, [ingredients]);

  const storeName = useCallback(
    (id?: string) =>
      stores.find((s) => s.id === id)?.name ?? t.transfers.mainStore,
    [stores, t.transfers.mainStore],
  );

  // ⚠️ **Prep items are not offered at all.** A sauce is not on a shelf as
  // itself — its balance is derived from what it was made of — so moving one
  // would subtract from a figure nobody holds. Refused on the server too; the
  // list is only here so nobody has to be told no after choosing.
  const shelves = useMemo(
    () => ingredients.filter((i) => !i.made),
    [ingredients],
  );

  const src = byId.get(fromId);
  const dst = byId.get(toId);

  // ⚠️ The screen names the refusal *before* the button is pressed. Every one
  // of these is a rule the server also enforces; saying it here is what stops
  // somebody typing a quantity into a move that was never going to be allowed.
  const refusal = (() => {
    if (!src || !dst) return "";
    if (src.id === dst.id) return t.transfers.sameShelf;
    if (src.unit !== dst.unit) return t.transfers.unitMismatch;
    if ((src.warehouseId ?? "") === (dst.warehouseId ?? ""))
      return t.transfers.sameStore;
    return "";
  })();

  async function save() {
    setBusy(true);
    setError("");
    try {
      await api.adminCreateTransfer({
        at: new Date(`${at}T12:00:00`).toISOString(),
        fromId,
        toId,
        qty: Number(qty) || 0,
        note: note.trim(),
      });
      setQty("");
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
        <h1 className="text-xl font-semibold">{t.transfers.title}</h1>
        <p className="mt-1 text-sm text-ink-soft">{t.transfers.intro}</p>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      <div className="flex flex-wrap items-center gap-2 text-sm">
        <span className="text-xs text-ink-muted">{t.transfers.period}</span>
        <input
          type="date"
          className="input w-auto"
          value={from}
          onChange={(e) => setFrom(e.target.value)}
        />
        <span className="text-ink-muted">—</span>
        <input
          type="date"
          className="input w-auto"
          value={to}
          onChange={(e) => setTo(e.target.value)}
        />
      </div>

      <div className="card p-3">
        <div className="flex flex-wrap items-end gap-2">
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.transfers.date}</span>
            <input
              type="date"
              className="input mt-1 w-auto"
              value={at}
              onChange={(e) => setAt(e.target.value)}
            />
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.transfers.from}</span>
            <select
              className="input mt-1 w-56"
              value={fromId}
              onChange={(e) => setFromId(e.target.value)}
            >
              <option value="">—</option>
              {shelves.map((i) => (
                <option key={i.id} value={i.id}>
                  {i.name} · {storeName(i.warehouseId)}
                </option>
              ))}
            </select>
          </label>
          <span className="pb-2 text-ink-muted">→</span>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.transfers.to}</span>
            <select
              className="input mt-1 w-56"
              value={toId}
              onChange={(e) => setToId(e.target.value)}
            >
              <option value="">—</option>
              {shelves.map((i) => (
                <option key={i.id} value={i.id}>
                  {i.name} · {storeName(i.warehouseId)}
                </option>
              ))}
            </select>
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">
              {t.transfers.qty}
              {src ? ` (${t.ingredients.units[src.unit] ?? src.unit})` : ""}
            </span>
            <input
              type="number"
              min={0}
              step="any"
              className="input mt-1 w-28"
              value={qty}
              onChange={(e) => setQty(e.target.value)}
            />
          </label>
          <label className="block flex-1 text-sm">
            <span className="text-xs text-ink-muted">{t.transfers.note}</span>
            <input
              className="input mt-1 w-full"
              placeholder={t.transfers.notePlaceholder}
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
          </label>
          <button
            className="btn-primary px-4 py-2"
            disabled={busy || !!refusal || !fromId || !toId || !Number(qty)}
            onClick={save}
          >
            {t.common.save}
          </button>
        </div>
        {refusal && <p className="mt-2 text-sm text-danger">{refusal}</p>}
      </div>

      <div className="card p-0">
        <ListScroll>
          <table className="w-full text-sm">
            <thead className="sticky top-0 bg-surface text-left text-xs text-ink-muted">
              <tr>
                <th className="px-3 py-2">{t.transfers.date}</th>
                <th className="px-3 py-2">{t.transfers.what}</th>
                <th className="px-3 py-2 text-right">{t.transfers.qty}</th>
                <th className="px-3 py-2">{t.transfers.note}</th>
                <th className="px-3 py-2 text-right">{t.transfers.value}</th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {rows.map((x) => (
                <tr key={x.id} className="border-t border-line">
                  <td className="px-3 py-2 whitespace-nowrap">
                    {formatDate(x.at)}
                  </td>
                  <td className="px-3 py-2">
                    {byId.get(x.fromId)?.name ?? "—"}
                    <span className="mx-1.5 text-ink-muted">→</span>
                    {byId.get(x.toId)?.name ?? "—"}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">{x.qty}</td>
                  <td className="px-3 py-2 text-ink-soft">
                    {x.note}
                    {x.by && (
                      <span className="ml-1 text-xs text-ink-muted">
                        · {x.by}
                      </span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {formatPrice(x.value)}
                  </td>
                  <td className="px-3 py-2 text-right">
                    <button
                      className="btn-ghost px-2 py-1 text-xs text-danger"
                      disabled={busy}
                      onClick={async () => {
                        setBusy(true);
                        try {
                          await api.adminDeleteTransfer(x.id);
                          load();
                        } finally {
                          setBusy(false);
                        }
                      }}
                    >
                      {t.common.delete}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {rows.length === 0 && (
            <p className="p-6 text-center text-sm text-ink-muted">
              {t.transfers.empty}
            </p>
          )}
        </ListScroll>
        {rows.length > 0 && (
          <div className="border-t border-line px-3 py-2 text-right text-sm">
            {/* ⚠️ Called "moved", never "spent". Nothing was bought and nothing
                was lost — the value is here so the row can be read in money as
                well as in kilos, and it is deliberately absent from the
                financial report's expenses. */}
            {t.transfers.movedTotal}:{" "}
            <span className="font-medium tabular-nums">
              {formatPrice(moved)}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
