"use client";

import { useCallback, useEffect, useMemo, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import type { ShoppingDraftRow, ShoppingOrder } from "@/lib/types";

// The shopping list, written where the news arrives.
//
// ⚠️ **Its own section rather than a corner of the stop list.** The two screens
// answer opposite questions — "this is off the menu now" and "buy this
// tomorrow" — and a cashier reaching for one at eight in the evening must not
// land on the other. They share a subject and nothing else.
//
// ⚠️ **The form starts from the shortage the store already worked out**, not
// from a blank page. A restaurant with two shopping lists — one the arithmetic
// produces and one a person types — gives the buyer no way to tell which is
// real, and the one they follow will be whichever they saw last. So the
// computed list fills the form and a person edits it: the same function the
// panel's shopping screen reads.
//
// ⚠️ **The unit is never editable.** A market sells mint in bunches and flour
// in sacks; "5" typed into a field measured in kilos is five kilos on the shelf
// instead of a quarter of one, the figure is then twenty times too high, the
// stop list never fires, and the gap surfaces a month later at a count as an
// unexplained shortfall.

type Draft = {
  key: string;
  ingredientId?: string;
  name: string;
  unit: string;
  qty: string;
  onHand?: number;
};

export default function ZakupScreen({
  onError,
}: {
  onError: (message: string) => void;
}) {
  const t = useAdminT();

  const [lines, setLines] = useState<Draft[]>([]);
  const [orders, setOrders] = useState<ShoppingOrder[]>([]);
  const [suggested, setSuggested] = useState<ShoppingDraftRow[]>([]);
  const [forDate, setForDate] = useState(() => {
    // ⚠️ Tomorrow, because that is what a shopping list is for. Today's has
    // already been shopped by the time anybody is standing at a till.
    const d = new Date();
    d.setDate(d.getDate() + 1);
    return d.toISOString().slice(0, 10);
  });
  const [query, setQuery] = useState("");
  const [preview, setPreview] = useState(false);
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState("");

  const load = useCallback(() => {
    api
      .staffBuyOrderDraft()
      .then((res) => setSuggested(res.rows))
      .catch(() => setSuggested([]));
    api
      .staffBuyOrders()
      .then((res) => setOrders(res.orders))
      .catch(() => setOrders([]));
  }, []);
  useEffect(load, [load]);

  const chosen = useMemo(
    () => new Set(lines.map((l) => l.ingredientId).filter(Boolean)),
    [lines],
  );

  const shown = useMemo(() => {
    const q = query.trim().toLowerCase();
    return suggested.filter(
      (row) =>
        !chosen.has(row.ingredientId) &&
        (q === "" || row.name.toLowerCase().includes(q)),
    );
  }, [suggested, chosen, query]);

  /** Whether what was typed names nothing the store knows about.
   *
   *  ⚠️ **Offered rather than refused.** A list somebody cannot finish writing
   *  is a list they write on paper instead, and then nothing here sees it —
   *  the lesson this product paid for with the supplier field and the void
   *  reason. The line carries the typed name and the delivery invents the
   *  ingredient later, marked for somebody to finish. */
  const unknown = useMemo(() => {
    const q = query.trim();
    if (q.length < 2) return "";
    return suggested.some((r) => r.name.toLowerCase() === q.toLowerCase())
      ? ""
      : q;
  }, [suggested, query]);

  function add(row: Partial<Draft> & { name: string }) {
    setDone("");
    setLines((cur) => [
      ...cur,
      {
        key: row.ingredientId ?? `new-${Date.now()}`,
        ingredientId: row.ingredientId,
        name: row.name,
        unit: row.unit ?? "",
        qty: row.qty ?? "",
        onHand: row.onHand,
      },
    ]);
    setQuery("");
  }

  const ready = lines.filter((l) => Number(l.qty) > 0);

  async function send() {
    if (ready.length === 0) {
      onError(t.zakup.nothingToSend);
      return;
    }
    setBusy(true);
    try {
      await api.staffCreateBuyOrder({
        forDate,
        lines: ready.map((l) => ({
          ingredientId: l.ingredientId,
          name: l.name,
          qty: Number(l.qty),
        })),
      });
      setLines([]);
      setPreview(false);
      setDone(t.zakup.sent(ready.length));
      load();
    } catch (e) {
      onError(e instanceof ApiError ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  const chip = (on: boolean) =>
    `shrink-0 rounded-[12px] px-3.5 py-2 text-[14px] font-semibold transition ${
      on
        ? "bg-[rgb(var(--till-accent-tint))] text-[rgb(var(--till-accent-ink))]"
        : "bg-ink/[0.05] text-ink-soft hover:bg-ink/[0.09]"
    }`;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <header className="border-b border-line px-3 py-3">
        <div className="flex flex-wrap items-center gap-3">
          <label className="text-[14px] text-ink-muted">
            {t.zakup.forDate}
            <input
              type="date"
              className="till-input ml-2 h-11 w-auto"
              value={forDate}
              onChange={(e) => setForDate(e.target.value)}
            />
          </label>
          <span className="ml-auto text-sm font-semibold text-ink-muted">
            {t.zakup.chosen(ready.length)}
          </span>
        </div>
      </header>

      <div className="min-h-0 flex-1 overflow-y-auto px-3 py-3">
        {done !== "" && (
          <p className="mb-3 text-sm font-semibold text-[rgb(var(--till-accent-ink))]">
            {done}
          </p>
        )}

        {/* ---- What is going on the list ---- */}
        {lines.length > 0 && (
          <>
            <h2 className="mb-2 text-[15px] font-bold">{t.zakup.listTitle}</h2>
            <ul className="mb-4 space-y-2">
              {lines.map((l) => (
                <li
                  key={l.key}
                  className="flex items-center gap-2 rounded-[14px] border border-line p-2.5"
                >
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-[15px] font-semibold">
                      {l.name}
                    </span>
                    {l.onHand !== undefined && (
                      <span className="text-[13px] text-ink-muted">
                        {t.zakup.onHand(l.onHand, l.unit)}
                      </span>
                    )}
                  </span>
                  <input
                    className="till-input h-11 w-24 text-center"
                    inputMode="decimal"
                    value={l.qty}
                    onChange={(e) =>
                      setLines((cur) =>
                        cur.map((x) =>
                          x.key === l.key
                            ? { ...x, qty: e.target.value.replace(",", ".") }
                            : x,
                        ),
                      )
                    }
                  />
                  {/* ⚠️ Shown, never chosen — see the note at the top. */}
                  <span className="w-10 text-[13px] text-ink-muted">
                    {l.unit}
                  </span>
                  <button
                    className="px-2 text-ink-muted"
                    onClick={() =>
                      setLines((cur) => cur.filter((x) => x.key !== l.key))
                    }
                  >
                    ✕
                  </button>
                </li>
              ))}
            </ul>
          </>
        )}

        {/* ---- What the store says is short ---- */}
        <h2 className="mb-2 text-[15px] font-bold">{t.zakup.shortTitle}</h2>
        <input
          className="till-input mb-2 h-11 w-full"
          placeholder={t.zakup.search}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        {shown.length === 0 && unknown === "" ? (
          <p className="py-4 text-sm text-ink-muted">{t.zakup.nothingShort}</p>
        ) : (
          <ul className="space-y-2">
            {shown.map((row) => (
              <li key={row.ingredientId}>
                <button
                  className="flex w-full items-center gap-2 rounded-[14px] border border-line p-3 text-left"
                  onClick={() =>
                    add({
                      ingredientId: row.ingredientId,
                      name: row.name,
                      unit: row.unit,
                      // ⚠️ Pre-filled with what is short, not locked to it: a
                      // manager who knows a holiday is coming buys more.
                      qty: String(row.qty),
                      onHand: row.onHand,
                    })
                  }
                >
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-[15px] font-semibold">
                      {row.name}
                    </span>
                    <span className="text-[13px] text-ink-muted">
                      {t.zakup.onHand(row.onHand, row.unit)} ·{" "}
                      {t.zakup.need(row.qty, row.unit)}
                    </span>
                  </span>
                  <span className="text-[rgb(var(--till-accent-ink))]">+</span>
                </button>
              </li>
            ))}
            {unknown !== "" && (
              <li>
                <button
                  className="w-full rounded-[14px] border border-dashed border-line p-3 text-left text-[15px]"
                  onClick={() => add({ name: unknown })}
                >
                  {t.zakup.addNew(unknown)}
                </button>
              </li>
            )}
          </ul>
        )}

        {/* ---- Lists already sent ---- */}
        {orders.length > 0 && (
          <>
            <h2 className="mb-2 mt-5 text-[15px] font-bold">
              {t.zakup.sentTitle}
            </h2>
            <ul className="space-y-2">
              {orders.slice(0, 8).map((o) => (
                <li
                  key={o.id}
                  className="rounded-[14px] border border-line p-3 text-[14px]"
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-semibold">{o.forDate}</span>
                    <span
                      className={
                        o.status === "done"
                          ? "text-[13px] text-ink-muted"
                          : "text-[13px] font-semibold text-[rgb(var(--till-accent-ink))]"
                      }
                    >
                      {o.status === "done"
                        ? t.zakup.statusDone
                        : t.zakup.statusSent}
                    </span>
                  </div>
                  {/* ⚠️ "Asked for ten, brought six" is the sentence this whole
                      document exists to make possible, so the counts are shown
                      together rather than only the result. */}
                  <p className="text-[13px] text-ink-muted">
                    {t.zakup.progress(
                      o.lines.filter((l) => l.gotAt && !l.missing).length,
                      o.lines.length,
                    )}
                    {o.createdBy ? ` · ${o.createdBy}` : ""}
                  </p>
                </li>
              ))}
            </ul>
          </>
        )}
      </div>

      <div className="border-t border-line px-3 py-2.5">
        <button
          className={chip(true) + " w-full py-3 text-[16px]"}
          disabled={ready.length === 0 || busy}
          onClick={() => setPreview(true)}
        >
          {t.zakup.review}
        </button>
      </div>

      {/* ---- The last look before it is sent ----
          ⚠️ **A list is somebody else's morning.** The buyer will not be able
          to ask what "5" meant, so the numbers and the units are read back once
          on one page, in the words they will arrive in. */}
      {preview && (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-ink/40 p-4 sm:items-center">
          <div className="till-dialog max-h-[85dvh] w-full max-w-sm overflow-y-auto p-4">
            <h2 className="font-display text-lg font-bold">
              {t.zakup.previewTitle}
            </h2>
            <p className="mt-1 text-sm text-ink-soft">
              {t.zakup.previewBody(forDate)}
            </p>
            <ul className="mt-3 divide-y divide-line text-sm">
              {ready.map((l) => (
                <li key={l.key} className="flex justify-between gap-2 py-1.5">
                  <span className="truncate">{l.name}</span>
                  <span className="shrink-0 font-semibold tabular-nums">
                    {l.qty} {l.unit}
                  </span>
                </li>
              ))}
            </ul>
            <div className="mt-5 flex gap-2">
              <button
                className="till-btn flex-1"
                autoFocus
                onClick={() => setPreview(false)}
              >
                {t.zakup.back}
              </button>
              <button
                className="till-btn-primary flex-1"
                disabled={busy}
                onClick={() => void send()}
              >
                {t.zakup.send}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
