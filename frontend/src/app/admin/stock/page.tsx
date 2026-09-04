"use client";

// The store: what is on the shelf right now, and what it is worth.
//
// ⚠️ **The screen the inventory module was missing.** Everything else records a
// *movement* — a delivery, a write-off, a count — and the one report that read
// them back is a flow: how much came in and went out over a period. Nobody had
// built the question an owner actually walks into the store with: what is here
// now, and what does it cost me.
//
// ⚠️ **It is an estimate and it says so on every load.** The figure is the last
// count, plus deliveries, less what the tech cards say the dishes used, less
// write-offs. It drifts exactly as far as the kitchen drifts from its cards,
// and as far back as the last count — so the date it is measured from is beside
// it, and "never counted" is the loudest thing this page can say. A restaurant
// that believes a stock number and finds it wrong stops believing the panel.

import { useCallback, useEffect, useMemo, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatDate, formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import { ListScroll } from "@/components/admin/PagedList";
import type { StockBalances, StockMovement } from "@/lib/types";

/** The undivided store, for a restaurant that never split one. */
const MAIN = "";

export default function StockPage() {
  const t = useAdminT();
  const scope = useAdminScope();
  const [data, setData] = useState<StockBalances | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  // ⚠️ **Derived, not stored-then-corrected.** A restaurant that has split its
  // store keeps nothing in the undivided one, so opening there showed an empty
  // table under a full set of tabs — which reads as "the stock did not load"
  // rather than "you are looking at the wrong shelf". The first store that
  // actually holds something is the honest default, and one tap changes it.
  // Same shape as the room's view: a pure function of the data plus the one
  // choice the person has actually made.
  const [picked, setPicked] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  // ⚠️ Off by default. The reason this page is opened is usually something
  // running out, and the short lines already sort to the top — a filter that
  // started on would hide the store from somebody who came to look at it.
  const [lowOnly, setLowOnly] = useState(false);
  const [card, setCard] = useState<StockMovement | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    api
      .adminStockBalances()
      .then(setData)
      .catch((e: unknown) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      )
      .finally(() => setLoading(false));
  }, [t.common.loadFailed]);
  useEffect(load, [load, scope.scopeKey]);

  const stores = data?.warehouses ?? [];
  // ⚠️ The tab strip is only drawn where the owner actually split the store —
  // a single tab called "the store" is a control that teaches people this
  // screen has settings to think about. Same rule as the room's zones.
  const showTabs = stores.length > 0;

  const firstStocked = useMemo(() => {
    const held = new Set((data?.rows ?? []).map((r) => r.warehouseId));
    if (held.has(MAIN)) return MAIN;
    return (data?.warehouses ?? []).find((wh) => held.has(wh.id))?.id ?? MAIN;
  }, [data]);
  const store = picked ?? firstStocked;
  const setStore = setPicked;

  const rows = useMemo(() => {
    const q = query.trim().toLowerCase();
    return (data?.rows ?? []).filter((r) => {
      if (r.warehouseId !== store) return false;
      if (lowOnly && !r.low) return false;
      return !q || r.name.toLowerCase().includes(q);
    });
  }, [data, store, lowOnly, query]);

  const since = data?.since?.[store] ?? null;
  const value = data?.value?.[store] ?? 0;
  const lowCount = (data?.rows ?? []).filter(
    (r) => r.warehouseId === store && r.low,
  ).length;

  async function openCard(ingredientId: string) {
    // A month back, which is the period a "where did it go" question is asked
    // about. The card has its own dates once it is open.
    const to = new Date();
    const from = new Date(to.getTime() - 30 * 24 * 3600 * 1000);
    try {
      setCard(
        await api.adminStockMovement(ingredientId, {
          from: from.toISOString().slice(0, 10),
          to: to.toISOString().slice(0, 10),
        }),
      );
    } catch {
      setError(t.common.loadFailed);
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t.stock.title}</h1>
        {/* ⚠️ Said every time, not once in a tooltip. Without this sentence the
            column reads as a balance the system has been keeping all along. */}
        <p className="mt-1 text-sm text-ink-soft">
          {t.stock.intro}{" "}
          {since ? t.stock.since(formatDate(since)) : t.stock.neverCounted}
        </p>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      {showTabs && (
        <div className="flex flex-wrap gap-1.5">
          <StoreTab
            on={store === MAIN}
            onClick={() => setStore(MAIN)}
            label={t.warehouses.unfiled}
            value={data?.value?.[MAIN] ?? 0}
            counted={Boolean(data?.since?.[MAIN])}
          />
          {stores.map((wh) => (
            <StoreTab
              key={wh.id}
              on={store === wh.id}
              onClick={() => setStore(wh.id)}
              label={wh.name}
              value={data?.value?.[wh.id] ?? 0}
              counted={Boolean(data?.since?.[wh.id])}
            />
          ))}
        </div>
      )}

      <div className="card p-3">
        <div className="flex flex-wrap items-center gap-2">
          <input
            className="input h-9 w-56"
            placeholder={t.stock.search}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          {/* ⚠️ Only offered when something is short. A filter that is always
              there and always finds nothing teaches people to stop pressing it,
              and this is the one control on the page worth pressing. */}
          {lowCount > 0 && (
            <button
              type="button"
              onClick={() => setLowOnly(!lowOnly)}
              className={`rounded-full border px-3 py-1 text-xs font-semibold ${
                lowOnly
                  ? "border-amber-500/50 bg-amber-500/10 text-amber-700 dark:text-amber-300"
                  : "border-line-strong text-ink-muted"
              }`}
            >
              {t.stock.lowOnly(lowCount)}
            </button>
          )}
          <span className="ml-auto text-sm">
            <span className="text-ink-muted">{t.stock.storeValue}: </span>
            <span className="font-semibold">{formatPrice(value)}</span>
          </span>
        </div>

        <ListScroll className="mt-3">
          <table className="w-full text-sm">
            <thead className="sticky top-0 bg-surface text-xs text-ink-muted">
              <tr>
                <th className="px-3 py-2 text-left">{t.stock.name}</th>
                <th className="px-3 py-2 text-right">{t.stock.qty}</th>
                <th className="px-3 py-2 text-right">{t.stock.value}</th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.ingredientId} className="border-t border-line">
                  <td className="px-3 py-2">
                    {r.name}
                    {/* Named rather than hidden — see balanceRow.Made. */}
                    {r.made && (
                      <span className="ml-2 text-xs text-ink-muted">
                        {t.stock.madeInHouse}
                      </span>
                    )}
                    {/* ⚠️ Said in words, not left as a minus sign. "Less than
                        nothing" is not a shortage the buyer can fix by
                        ordering — it means a delivery was never entered or the
                        card takes more than the kitchen does, and until it is
                        resolved every other figure on this screen is resting
                        on it. */}
                    {r.negative && (
                      <span className="ml-2 text-xs text-danger">
                        {t.stock.negative}
                      </span>
                    )}
                  </td>
                  <td
                    className={`px-3 py-2 text-right tabular-nums ${
                      r.negative
                        ? "font-semibold text-danger"
                        : r.low
                          ? "font-semibold text-amber-700 dark:text-amber-300"
                          : ""
                    }`}
                  >
                    {r.qty} {t.ingredients.units[r.unit as "kg"]}
                    {r.low && r.minQty ? (
                      <span className="block text-xs font-normal text-ink-muted">
                        {t.stock.minIs(r.minQty)}
                      </span>
                    ) : null}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {r.made ? "—" : formatPrice(r.value)}
                  </td>
                  <td className="px-3 py-2 text-right">
                    <button
                      className="btn-ghost px-2 py-1 text-xs"
                      onClick={() => void openCard(r.ingredientId)}
                    >
                      {t.stock.card}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {!loading && rows.length === 0 && (
            <p className="py-6 text-center text-sm text-ink-muted">
              {t.stock.empty}
            </p>
          )}
        </ListScroll>
      </div>

      {card && <MovementCard card={card} onClose={() => setCard(null)} />}
    </div>
  );
}

function StoreTab({
  on,
  onClick,
  label,
  value,
  counted,
}: {
  on: boolean;
  onClick: () => void;
  label: string;
  value: number;
  /** Whether this store has ever been counted. ⚠️ Without a count the figure
   *  behind this money is "everything that ever arrived, less everything the
   *  cards account for, since the beginning of time" — arithmetic that is
   *  correct and describes nothing. Printed as a number on a tab it is read as
   *  a valuation, which is the one thing it is not. */
  counted: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded-full border px-3 py-1.5 text-sm ${
        on
          ? "border-brand bg-brand/10 font-semibold text-brand"
          : "border-line text-ink-muted hover:border-line-strong"
      }`}
    >
      {label}
      {/* What the store holds, on the tab. "The bar is four million" is a
          sentence somebody can act on; a branch total is one they can only
          nod at. */}
      <span className="ml-1.5 text-xs opacity-70">
        {counted ? formatPrice(value) : "—"}
      </span>
    </button>
  );
}

/** Where one ingredient went.
 *
 *  ⚠️ **The report a shortfall sends somebody to.** A balance says forty kilos
 *  are missing; only this says whether they were sold, thrown away, or never
 *  counted in the first place. Built from the same four facts as the balance —
 *  taken apart instead of added up — so the two cannot disagree. */
function MovementCard({
  card,
  onClose,
}: {
  card: StockMovement;
  onClose: () => void;
}) {
  const t = useAdminT();
  const unit = t.ingredients.units[card.ingredient.unit as "kg"];
  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-ink/40 p-4 sm:items-center"
      onClick={onClose}
    >
      <div
        className="card max-h-[85dvh] w-full max-w-lg overflow-y-auto p-4"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-2">
          <div>
            <h2 className="font-semibold">{card.ingredient.name}</h2>
            <p className="text-xs text-ink-muted">
              {formatDate(card.from)} — {formatDate(card.to)}
            </p>
          </div>
          <button className="btn-ghost px-2 py-1 text-sm" onClick={onClose}>
            ✕
          </button>
        </div>

        <dl className="mt-3 space-y-1.5 text-sm">
          <Line label={t.stock.opening} value={`${card.opening} ${unit}`} />
          <Line label={t.stock.cameIn} value={`+ ${card.in} ${unit}`} />
          {/* ⚠️ Two out columns, not one. "Sold" and "thrown away" are different
              questions about the same missing kilo, and a single line would
              make them one — which is exactly the answer somebody opened this
              to get apart. */}
          <Line label={t.stock.soldOut} value={`− ${card.used} ${unit}`} />
          <Line label={t.stock.writtenOff} value={`− ${card.written} ${unit}`} />
          {/* ⚠️ **Shown only when they happened, and that is why they were
              missing.** A single-kitchen restaurant never moves stock and never
              batches anything, so four permanent zero rows would be four lines
              of noise on every card. But a restaurant that does either had a
              card whose lines did not add up to its own closing figure — the
              opening and closing come from the balance arithmetic, which counts
              both, and this list did not. The reader was left to explain a gap
              the screen invented. */}
          {card.movedIn > 0 && (
            <Line label={t.stock.movedIn} value={`+ ${card.movedIn} ${unit}`} />
          )}
          {card.movedOut > 0 && (
            <Line
              label={t.stock.movedOut}
              value={`− ${card.movedOut} ${unit}`}
            />
          )}
          {card.produced > 0 && (
            <Line
              label={t.stock.produced}
              value={`+ ${card.produced} ${unit}`}
            />
          )}
          {card.producedUsed > 0 && (
            <Line
              label={t.stock.producedUsed}
              value={`− ${card.producedUsed} ${unit}`}
            />
          )}
          <div className="flex justify-between border-t border-line pt-1.5 font-semibold">
            <span>{t.stock.closing}</span>
            <span>
              {card.closing} {unit}
            </span>
          </div>
        </dl>

        {card.docs.length > 0 && (
          <ul className="mt-3 space-y-1 border-t border-line pt-2 text-xs">
            {card.docs.map((d, i) => (
              <li key={i} className="flex justify-between gap-2">
                <span className="truncate text-ink-muted">
                  {formatDate(d.at)} · {t.stock.kinds[d.kind] ?? d.kind}
                  {d.note ? ` · ${d.note}` : ""}
                </span>
                <span className={d.qty < 0 ? "text-danger" : ""}>
                  {d.qty > 0 ? "+" : ""}
                  {d.qty} {unit}
                </span>
              </li>
            ))}
          </ul>
        )}
        {/* ⚠️ Said out loud: what the dishes used has no document behind it. It
            is computed from the tech cards, so it cannot be listed line by line
            the way a delivery can — and an owner comparing the two columns
            deserves to know which one is measured and which is derived. */}
        <p className="mt-3 text-xs text-ink-muted">{t.stock.soldNote}</p>
      </div>
    </div>
  );
}

function Line({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-2">
      <span className="text-ink-muted">{label}</span>
      <span className="tabular-nums">{value}</span>
    </div>
  );
}
