"use client";

// Batches made in a central kitchen.
//
// ⚠️ **The document that lets a chain have one.** A restaurant with one kitchen
// needs nothing here: the sauce is made as it goes, it is never on a shelf, and
// a dish that used it is read as having used tomatoes. A chain breaks that —
// the central kitchen makes forty kilos on Monday and sends it to three
// branches, where the sauce *is* a tub in a fridge and the tomatoes never were.
// This is where the tomatoes stop being the sauce.
//
// ⚠️ **It is the other half of the ingredient's "made in batches" flag**, and
// neither works alone: the flag says the item is kept on a shelf and consumed
// as itself, and this is what puts it there and takes its inputs off. See
// models/production.go.

import { useCallback, useEffect, useMemo, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatDate, formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import { ListScroll } from "@/components/admin/PagedList";
import type { Ingredient, Production, Warehouse } from "@/lib/types";
import { QtyInput } from "@/components/QtyInput";
import { qtyNumber } from "@/lib/qty";

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

export default function ProductionPage() {
  const t = useAdminT();
  const scope = useAdminScope();
  const [rows, setRows] = useState<Production[]>([]);
  const [made, setMade] = useState(0);
  const [from, setFrom] = useState(monthStart);
  const [to, setTo] = useState(today);
  const [ingredients, setIngredients] = useState<Ingredient[]>([]);
  const [stores, setStores] = useState<Warehouse[]>([]);
  const [at, setAt] = useState(today);
  const [ingredientId, setIngredientId] = useState("");
  const [warehouseId, setWarehouseId] = useState("");
  const [qty, setQty] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    api
      .adminProductions({ from, to })
      .then((d) => {
        setRows(d.productions);
        setMade(d.made);
      })
      .catch((e) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      );
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

  // ⚠️ **Only batched prep items are offered.** Producing something bought is a
  // delivery; producing a prep item that is not batched would subtract its
  // inputs here *and* again through the card when a dish is sold, and the
  // shortfall would surface weeks later at a count as somebody's fault.
  // Refused on the server too — the list is here so nobody is told no after
  // filling the form in.
  const makeable = useMemo(
    () => ingredients.filter((i) => i.batched),
    [ingredients],
  );

  // ⚠️ Only central kitchens: the inputs come off *these* shelves, so the store
  // has to be the room somebody actually cooked in.
  const kitchens = useMemo(
    () => stores.filter((s) => s.kind === "production"),
    [stores],
  );

  const item = byId.get(ingredientId);

  // ⚠️ The screen says why nothing can be made *before* a quantity is typed.
  // Both of these are ordinary states rather than errors: a restaurant with one
  // kitchen has neither, and this page is simply not for it.
  const refusal = (() => {
    if (kitchens.length === 0) return t.production.noKitchen;
    if (makeable.length === 0) return t.production.nothingBatched;
    return "";
  })();

  async function save() {
    setBusy(true);
    setError("");
    try {
      await api.adminCreateProduction({
        at: new Date(`${at}T12:00:00`).toISOString(),
        warehouseId,
        ingredientId,
        qty: qtyNumber(qty),
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
        <h1 className="text-xl font-semibold">{t.production.title}</h1>
        <p className="mt-1 text-sm text-ink-soft">{t.production.intro}</p>
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
            <span className="text-xs text-ink-muted">
              {t.production.whatMade}
            </span>
            <select
              className="input mt-1 w-56"
              value={ingredientId}
              onChange={(e) => setIngredientId(e.target.value)}
            >
              <option value="">—</option>
              {makeable.map((i) => (
                <option key={i.id} value={i.id}>
                  {i.name}
                </option>
              ))}
            </select>
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">
              {t.production.kitchen}
            </span>
            <select
              className="input mt-1 w-48"
              value={warehouseId}
              onChange={(e) => setWarehouseId(e.target.value)}
            >
              <option value="">—</option>
              {kitchens.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">
              {t.transfers.qty}
              {item ? ` (${t.ingredients.units[item.unit] ?? item.unit})` : ""}
            </span>
            <QtyInput className="input mt-1 w-28" value={qty} onValue={setQty} />
          </label>
          <label className="block flex-1 text-sm">
            <span className="text-xs text-ink-muted">{t.transfers.note}</span>
            <input
              className="input mt-1 w-full"
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
          </label>
          <button
            className="btn-primary px-4 py-2"
            disabled={
              busy ||
              !!refusal ||
              !ingredientId ||
              !warehouseId ||
              !qtyNumber(qty)
            }
            onClick={save}
          >
            {t.common.save}
          </button>
        </div>
        {refusal && <p className="mt-2 text-sm text-ink-muted">{refusal}</p>}
        {/* ⚠️ What a batch took is not shown before it is saved, because the
            screen does not compute it: the server reads the card and freezes
            what it took onto the document. A preview here would be a second
            implementation of the costing, and the drifted one would be the one
            an owner is looking at. */}
      </div>

      <div className="card p-0">
        <ListScroll>
          <table className="w-full text-sm">
            <thead className="sticky top-0 bg-surface text-left text-xs text-ink-muted">
              <tr>
                <th className="px-3 py-2">{t.transfers.date}</th>
                <th className="px-3 py-2">{t.production.whatMade}</th>
                <th className="px-3 py-2 text-right">{t.transfers.qty}</th>
                <th className="px-3 py-2">{t.production.took}</th>
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
                    {byId.get(x.ingredientId)?.name ?? "—"}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">{x.qty}</td>
                  <td className="px-3 py-2 text-ink-soft">
                    {/* The frozen list, not the card as it reads today: a batch
                        made in March took what it took in March. */}
                    {x.lines
                      .map((l) => `${l.name ?? "—"} ${l.qty}`)
                      .join(", ")}
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
                          await api.adminDeleteProduction(x.id);
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
              {t.production.empty}
            </p>
          )}
        </ListScroll>
        {rows.length > 0 && (
          <div className="border-t border-line px-3 py-2 text-right text-sm">
            {/* ⚠️ "Made", never "spent" — the same rule as a transfer. Nothing
                was bought and nothing was lost; tomatoes became sauce, and this
                figure is deliberately absent from the financial report's
                expenses. */}
            {t.production.madeTotal}:{" "}
            <span className="font-medium tabular-nums">
              {formatPrice(made)}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
