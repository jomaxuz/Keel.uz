"use client";

// Deliveries: what came in, from whom, and what it cost.
//
// ⚠️ **This is where an ingredient's price stops being retyped.** It used to be
// a number somebody read off an invoice and entered by hand, which is exactly
// the step that stops happening after the fortieth delivery — and a stale price
// makes every dish's cost, margin and report quietly wrong. Entering the
// delivery is work the restaurant already does; the price falls out of it.
//
// ⚠️ **Dated by the invoice, not by today.** A delivery is a measurement
// carrying its own date, so its prices apply from that day — unlike editing a
// price by hand, which can only mean "from now on" because the form cannot tell
// a correction from a rise.
//
// ⚠️ **Not stock.** Nothing here subtracts what the kitchen used, and no line
// on this screen claims a remaining quantity: a restaurant that believes a
// stock figure and finds it wrong stops believing the panel entirely.

import { useCallback, useEffect, useMemo, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatDate, formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import { ListScroll } from "@/components/admin/PagedList";
import type { Ingredient, Purchase, PurchaseLine } from "@/lib/types";

function today() {
  const d = new Date();
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

export default function PurchasesPage() {
  const t = useAdminT();
  const scope = useAdminScope();
  const [rows, setRows] = useState<Purchase[]>([]);
  const [spent, setSpent] = useState(0);
  const [ingredients, setIngredients] = useState<Ingredient[]>([]);
  const [at, setAt] = useState(today);
  const [supplier, setSupplier] = useState("");
  const [lines, setLines] = useState<PurchaseLine[]>([]);
  const [picked, setPicked] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const load = useCallback(() => {
    api
      .adminPurchases()
      .then((d) => {
        setRows(d.purchases);
        setSpent(d.spent);
      })
      .catch(() => setError(t.common.loadFailed));
    api
      .adminIngredients()
      .then((d) => setIngredients(d.ingredients))
      .catch(() => setIngredients([]));
  }, [t.common.loadFailed]);

  useEffect(load, [load, scope.scopeKey]);

  const byId = useMemo(() => {
    const m = new Map<string, Ingredient>();
    for (const i of ingredients) m.set(i.id, i);
    return m;
  }, [ingredients]);

  const total = lines.reduce((s, l) => s + Math.round(l.price * l.qty), 0);

  async function save() {
    if (lines.length === 0) return;
    setBusy(true);
    setError("");
    setNotice("");
    try {
      const res = await api.adminCreatePurchase({
        at: new Date(`${at}T12:00:00`).toISOString(),
        supplier: supplier.trim(),
        lines,
      });
      setLines([]);
      setSupplier("");
      // ⚠️ Says how many prices moved. That is the part of this that changes
      // other screens — the cost of every dish containing them — and somebody
      // entering an invoice should be told it happened rather than discover it
      // in a report.
      setNotice(
        res.pricesChanged > 0
          ? t.purchases.pricesChanged(res.pricesChanged)
          : t.purchases.noPriceChange,
      );
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
        <h1 className="text-xl font-semibold">{t.purchases.title}</h1>
        <p className="mt-1 text-sm text-ink-soft">{t.purchases.intro}</p>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}
      {notice && <p className="text-sm text-ink-soft">{notice}</p>}

      <div className="card space-y-3 p-3">
        <div className="flex flex-wrap items-end gap-2">
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.purchases.date}</span>
            <input
              type="date"
              className="input mt-1 w-auto"
              value={at}
              onChange={(e) => setAt(e.target.value)}
            />
          </label>
          <label className="block flex-1 text-sm">
            <span className="text-xs text-ink-muted">
              {t.purchases.supplier}
            </span>
            <input
              className="input mt-1 w-full"
              placeholder={t.purchases.supplierPlaceholder}
              value={supplier}
              onChange={(e) => setSupplier(e.target.value)}
            />
          </label>
        </div>

        {lines.length > 0 && (
          <ul className="space-y-1">
            {lines.map((l, i) => {
              const ing = byId.get(l.ingredientId);
              const was = ing?.price ?? 0;
              return (
                <li key={l.ingredientId} className="flex items-center gap-2">
                  <span className="min-w-0 flex-1 truncate text-sm">
                    {ing?.name ?? "—"}
                  </span>
                  <input
                    type="number"
                    min={0}
                    step="any"
                    className="input w-24 py-1 text-right"
                    value={l.qty || ""}
                    placeholder={t.purchases.qty}
                    onChange={(e) => {
                      const next = lines.slice();
                      next[i] = { ...l, qty: Number(e.target.value) || 0 };
                      setLines(next);
                    }}
                  />
                  <span className="w-8 text-xs text-ink-muted">
                    {ing ? t.ingredients.units[ing.unit] : ""}
                  </span>
                  <input
                    type="number"
                    min={0}
                    className="input w-32 py-1 text-right"
                    value={l.price || ""}
                    placeholder={t.purchases.unitPrice}
                    onChange={(e) => {
                      const next = lines.slice();
                      next[i] = { ...l, price: Number(e.target.value) || 0 };
                      setLines(next);
                    }}
                  />
                  {/* ⚠️ What it used to cost, beside what is being typed. A
                      delivery that quietly doubles a price is the thing this
                      screen exists to make visible, and the person holding the
                      invoice is the only one who can say whether it is right. */}
                  {was > 0 && l.price > 0 && l.price !== was && (
                    <span
                      className={`w-28 text-xs ${l.price > was ? "text-danger" : "text-ink-muted"}`}
                    >
                      {t.purchases.was(formatPrice(was))}
                    </span>
                  )}
                  <span className="w-28 text-right text-xs text-ink-muted tabular-nums">
                    {formatPrice(Math.round(l.price * l.qty))}
                  </span>
                  <button
                    type="button"
                    className="btn-ghost px-2 py-1 text-xs"
                    onClick={() => setLines(lines.filter((_, j) => j !== i))}
                  >
                    ✕
                  </button>
                </li>
              );
            })}
          </ul>
        )}

        <div className="flex flex-wrap items-center gap-2">
          <select
            className="input w-auto py-1"
            value={picked}
            onChange={(e) => setPicked(e.target.value)}
          >
            <option value="">{t.purchases.addLine}</option>
            {ingredients
              .filter((i) => !i.made)
              .filter((i) => !lines.some((l) => l.ingredientId === i.id))
              .map((i) => (
                <option key={i.id} value={i.id}>
                  {i.name}
                </option>
              ))}
          </select>
          <button
            type="button"
            className="btn px-3 py-1 text-sm"
            disabled={!picked}
            onClick={() => {
              const ing = byId.get(picked);
              setLines([
                ...lines,
                // Prefilled with the price we know: most deliveries do not
                // change it, and retyping the same number forty times is how
                // this screen would stop being used.
                { ingredientId: picked, qty: 0, price: ing?.price ?? 0 },
              ]);
              setPicked("");
            }}
          >
            {t.common.add}
          </button>
          <span className="ml-auto text-sm">
            {t.purchases.total}:{" "}
            <span className="font-medium tabular-nums">
              {formatPrice(total)}
            </span>
          </span>
          <button
            className="btn-primary px-4 py-2"
            disabled={busy || lines.length === 0 || total <= 0}
            onClick={save}
          >
            {t.common.save}
          </button>
        </div>
      </div>

      <div className="card p-0">
        <ListScroll>
          <table className="w-full text-sm">
            <thead className="sticky top-0 bg-surface text-left text-xs text-ink-muted">
              <tr>
                <th className="px-3 py-2">{t.purchases.date}</th>
                <th className="px-3 py-2">{t.purchases.supplier}</th>
                <th className="px-3 py-2">{t.purchases.linesCol}</th>
                <th className="px-3 py-2 text-right">{t.purchases.total}</th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {rows.map((p) => (
                <tr key={p.id} className="border-t border-line">
                  <td className="px-3 py-2 whitespace-nowrap">
                    {formatDate(p.at)}
                  </td>
                  <td className="px-3 py-2">{p.supplier || "—"}</td>
                  <td className="px-3 py-2 text-ink-muted">
                    {p.lines
                      .map((l) => byId.get(l.ingredientId)?.name ?? "—")
                      .join(", ")}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {formatPrice(p.total)}
                  </td>
                  <td className="px-3 py-2 text-right">
                    <button
                      className="btn-ghost px-2 py-1 text-xs text-danger"
                      disabled={busy}
                      onClick={async () => {
                        setBusy(true);
                        try {
                          await api.adminDeletePurchase(p.id);
                          // ⚠️ The prices it wrote stay: later deliveries and
                          // every dish costed in between sit on top of them,
                          // and a delete that re-costed a month would be worse
                          // than a wrong invoice row.
                          setNotice(t.purchases.deletedKeepsPrices);
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
              {t.purchases.empty}
            </p>
          )}
        </ListScroll>
        {rows.length > 0 && (
          <div className="border-t border-line px-3 py-2 text-right text-sm">
            {t.purchases.spent}:{" "}
            <span className="font-medium tabular-nums">
              {formatPrice(spent)}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
