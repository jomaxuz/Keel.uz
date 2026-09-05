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
import Modal from "@/components/admin/Modal";
import type {
  AdvanceBalance,
  Ingredient,
  Purchase,
  PurchaseLine,
  StaffRow,
  Supplier,
} from "@/lib/types";
import { sellsGoods } from "@/lib/types";
import { qtyNumber } from "@/lib/qty";
import { QtyInput } from "@/components/QtyInput";

function today() {
  const d = new Date();
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

export default function PurchasesPage() {
  const t = useAdminT();
  const scope = useAdminScope();
  // ⚠️ Whether this business is inspected on expiry dates. A default, not a
  // rule: a line that already carries a date keeps its boxes whatever kind of
  // business this is.
  const needsExpiry = sellsGoods(scope.brand);

  const [rows, setRows] = useState<Purchase[]>([]);
  const [spent, setSpent] = useState(0);
  const [ingredients, setIngredients] = useState<Ingredient[]>([]);
  const [at, setAt] = useState(today);
  const [supplier, setSupplier] = useState("");
  // ⚠️ The id **and** the typed text, because both are real answers: the three
  // regulars are picked from the list, and a market run is typed. Requiring the
  // list would stop the market run being recorded at all.
  const [supplierId, setSupplierId] = useState("");
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [lines, setLines] = useState<PurchaseLine[]>([]);
  const [picked, setPicked] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  // ⚠️ Which invoice is being corrected, or empty for a new one. An invoice is
  // forty numbers typed at a door and the twenty-first is a transposition;
  // until now the only remedy was delete and retype, which loses the entry
  // date, who took it in, and leaves the wrong price standing in the history.
  const [editing, setEditing] = useState("");
  // ⚠️ Which invoice is being settled. A tap used to mark it paid outright;
  // now it asks the one question that decides whether a box got lighter.
  const [paying, setPaying] = useState<Purchase | null>(null);

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
    api
      .adminSuppliers()
      .then((d) => setSuppliers(d.suppliers))
      .catch(() => setSuppliers([]));
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
      const body = {
        at: new Date(`${at}T12:00:00`).toISOString(),
        supplierId: supplierId || undefined,
        supplier: supplier.trim(),
        lines,
      };
      const res = editing
        ? await api.adminUpdatePurchase(editing, body)
        : await api.adminCreatePurchase(body);
      setLines([]);
      setSupplier("");
      setSupplierId("");
      setEditing("");
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

      {/* ⚠️ **Beside the deliveries it pays for, not on a screen of its own.**
          The two questions are asked in one breath — "what came in" and "who is
          still carrying our money" — and a ledger a floor away is a ledger
          nobody reconciles. */}
      <AdvancePanel />

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
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">
              {t.purchases.supplier}
            </span>
            <select
              className="input mt-1 w-52"
              value={supplierId}
              onChange={(e) => {
                setSupplierId(e.target.value);
                // Picking one clears the typed name: the server freezes the
                // supplier's own name onto the invoice, and leaving stale text
                // beside it would give the row two answers.
                if (e.target.value) setSupplier("");
              }}
            >
              <option value="">{t.purchases.supplierNone}</option>
              {suppliers
                .filter((x) => x.isActive)
                .map((x) => (
                  <option key={x.id} value={x.id}>
                    {x.name}
                  </option>
                ))}
            </select>
          </label>
          {!supplierId && (
            <label className="block flex-1 text-sm">
              <span className="text-xs text-ink-muted">
                {t.purchases.supplierTyped}
              </span>
              <input
                className="input mt-1 w-full"
                placeholder={t.purchases.supplierPlaceholder}
                value={supplier}
                onChange={(e) => setSupplier(e.target.value)}
              />
            </label>
          )}
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
                  <QtyInput
                    className="input w-24 py-1 text-right"
                    value={l.qty ?? 0}
                    placeholder={t.purchases.qty}
                    onValue={(v) => {
                      const next = lines.slice();
                      next[i] = { ...l, qty: qtyNumber(v) };
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
                  {/* ⚠️ **Shown to a shop, and to any line that already
                      carries a date.** A pharmacy is inspected on this and a
                      grocery loses money to it; a restaurant's delivery form
                      does not need two more boxes on every line. Data wins over
                      the type, here as everywhere: a date already entered keeps
                      its field. */}
                  {(needsExpiry || l.expiresAt || l.series) && (
                    <>
                      <input
                        type="date"
                        className="input w-36 py-1 text-xs"
                        title={t.purchases.expiresAt}
                        value={l.expiresAt ?? ""}
                        onChange={(e) => {
                          const next = lines.slice();
                          next[i] = { ...l, expiresAt: e.target.value };
                          setLines(next);
                        }}
                      />
                      <input
                        className="input w-24 py-1 text-xs"
                        placeholder={t.purchases.series}
                        title={t.purchases.series}
                        value={l.series ?? ""}
                        onChange={(e) => {
                          const next = lines.slice();
                          next[i] = { ...l, series: e.target.value };
                          setLines(next);
                        }}
                      />
                    </>
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

        {/* ⚠️ **What is running out, on the screen where somebody is about to
            order.** The warning lives in the ingredients list, which is not
            where the ordering happens — and a warning you have to remember in
            another room is one that gets remembered after the delivery. One
            tap puts it on this invoice. */}
        {ingredients.some(
          (i) => i.low && !lines.some((l) => l.ingredientId === i.id),
        ) && (
          <div className="rounded-xl bg-amber-500/10 px-3 py-2 text-sm">
            <span className="text-ink-soft">{t.purchases.lowTitle}</span>{" "}
            {ingredients
              .filter(
                (i) => i.low && !lines.some((l) => l.ingredientId === i.id),
              )
              .map((i) => (
                <button
                  key={i.id}
                  type="button"
                  className="mr-2 underline"
                  onClick={() =>
                    setLines([
                      ...lines,
                      { ingredientId: i.id, qty: 0, price: i.price },
                    ])
                  }
                >
                  {i.name}
                  <span className="ml-1 text-xs text-ink-muted">
                    {i.expected ?? 0}
                    {t.ingredients.units[i.unit] ?? i.unit}
                  </span>
                </button>
              ))}
          </div>
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
          {editing && (
            <button
              className="btn-ghost px-3 py-2"
              onClick={() => {
                setEditing("");
                setLines([]);
                setSupplier("");
                setSupplierId("");
              }}
            >
              {t.common.cancel}
            </button>
          )}
        </div>
        {editing && (
          // ⚠️ Said before the button is pressed, because it is the part that
          // reaches other screens: correcting a price changes what every dish
          // containing that ingredient has cost since the delivery's date.
          <p className="mt-2 text-xs text-ink-muted">
            {t.purchases.editingNotice}
          </p>
        )}
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
                <th className="px-3 py-2">{t.purchases.paidCol}</th>
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
                  {/* ⚠️ Unpaid is the state worth showing, so it is the one
                      that carries a colour and a button. A settled invoice
                      needs no action and gets no emphasis. */}
                  <td className="px-3 py-2 whitespace-nowrap">
                    {p.paid ? (
                      <span className="text-xs text-ink-muted">
                        {t.purchases.paid}
                      </span>
                    ) : (
                      <button
                        className="btn-ghost px-2 py-1 text-xs text-amber-700 dark:text-amber-300"
                        disabled={busy}
                        onClick={() => setPaying(p)}
                      >
                        {t.purchases.markPaid}
                      </button>
                    )}
                  </td>
                  <td className="px-3 py-2 text-right whitespace-nowrap">
                    <button
                      className="btn-ghost px-2 py-1 text-xs"
                      disabled={busy}
                      onClick={() => {
                        // The form above becomes this invoice. ⚠️ Saving it
                        // withdraws the prices this delivery claimed and writes
                        // them again — an edit is a claim about the price,
                        // which a delete deliberately is not.
                        setEditing(p.id);
                        setAt(p.at.slice(0, 10));
                        setSupplierId(p.supplierId ?? "");
                        setSupplier(p.supplierId ? "" : (p.supplier ?? ""));
                        setLines(p.lines);
                        setNotice("");
                        window.scrollTo({ top: 0, behavior: "smooth" });
                      }}
                    >
                      {t.common.edit}
                    </button>
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

      {paying && (
        <PayPurchaseDialog
          purchase={paying}
          onClose={() => setPaying(null)}
          onDone={() => {
            setPaying(null);
            load();
          }}
          onError={setError}
        />
      )}
    </div>
  );
}

/**
 * Settling a supplier's invoice.
 *
 * ⚠️ **One tap became one question, and it is worth the extra tap.** Marking a
 * delivery paid used to say the supplier was square and nothing at all about
 * which box the notes came out of — and paying a supplier at the door is the
 * most common way money leaves a restaurant's safe. Asked rather than inferred:
 * this is settled by transfer, out of the drawer, or out of the buyer's petty
 * cash just as often.
 */
function PayPurchaseDialog({
  purchase,
  onClose,
  onDone,
  onError,
}: {
  purchase: Purchase;
  onClose: () => void;
  onDone: () => void;
  onError: (m: string) => void;
}) {
  const t = useAdminT();
  const [fromSafe, setFromSafe] = useState(false);
  const [busy, setBusy] = useState(false);

  return (
    <Modal onClose={onClose}>
      <h2 className="font-display text-lg font-bold">{t.purchases.payTitle}</h2>
      <p className="mt-1 text-sm text-ink-muted">
        {purchase.supplier || "—"} · {formatPrice(purchase.total)}
      </p>
      <label className="mt-4 flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={fromSafe}
          onChange={(e) => setFromSafe(e.target.checked)}
        />
        {t.purchases.payFromSafe}
      </label>
      <div className="mt-5 flex justify-end gap-2">
        <button className="btn-ghost px-4 py-2 text-sm" onClick={onClose}>
          {t.common.cancel}
        </button>
        <button
          className="btn-primary px-4 py-2 text-sm disabled:opacity-50"
          disabled={busy}
          onClick={async () => {
            setBusy(true);
            try {
              await api.adminPayPurchase(purchase.id, fromSafe);
              onDone();
            } catch (e) {
              onError(e instanceof ApiError ? e.message : t.common.saveFailed);
              onClose();
            } finally {
              setBusy(false);
            }
          }}
        >
          {busy ? t.common.saving : t.purchases.markPaid}
        </button>
      </div>
    </Modal>
  );
}

/**
 * Petty cash for the buying: who was given what, and what is left in their
 * hands.
 *
 * ⚠️ **An advance is not an outgoing.** The money is spent when it buys
 * something, and that is the delivery this page already lists — which the
 * financial report already counts. Showing a hand-over as an expense too would
 * count the same money twice, once as cash leaving and once as food arriving.
 *
 * ⚠️ **Nothing is computed here.** The balance is three sums and a subtraction
 * made on the server, from the ledger and the deliveries; a browser repeating
 * that arithmetic would be a second answer to a question about money.
 */
function AdvancePanel() {
  const t = useAdminT();
  const scope = useAdminScope();
  const [balances, setBalances] = useState<AdvanceBalance[]>([]);
  const [staff, setStaff] = useState<StaffRow[]>([]);
  const [who, setWho] = useState("");
  const [amount, setAmount] = useState("");
  const [kind, setKind] = useState<"out" | "back">("out");
  /** ⚠️ **Asked rather than assumed.** A float can just as easily come from an
   *  owner's own pocket, and a safe balance that quietly counted every
   *  hand-over would be a confident figure about a box nobody opened. */
  const [fromSafe, setFromSafe] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [open, setOpen] = useState(false);

  const load = useCallback(() => {
    api
      .adminAdvances()
      .then((res) => setBalances(res.balances))
      .catch(() => setBalances([]));
  }, []);
  useEffect(load, [load, scope.scopeKey]);

  // ⚠️ Only once the form is opened: this page is opened many times a day to
  // enter a delivery, and the staff list is needed on none of those visits.
  useEffect(() => {
    if (!open || staff.length > 0) return;
    api
      .adminStaff()
      .then(setStaff)
      .catch(() => setStaff([]));
  }, [open, staff.length]);

  async function save() {
    const sum = Number(amount);
    if (!who || !(sum > 0)) return;
    setBusy(true);
    setError("");
    try {
      await api.adminCreateAdvance({ staffId: who, kind, amount: sum, fromSafe });
      setAmount("");
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="card space-y-2 p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 className="font-semibold">{t.advances.title}</h2>
          <p className="text-xs text-ink-muted">{t.advances.intro}</p>
        </div>
        <button className="btn-ghost text-sm" onClick={() => setOpen(!open)}>
          {open ? t.common.cancel : t.advances.give}
        </button>
      </div>

      {balances.length === 0 ? (
        <p className="text-sm text-ink-muted">{t.advances.nobody}</p>
      ) : (
        <ul className="divide-y divide-line text-sm">
          {balances.map((b) => (
            <li
              key={b.staffId}
              className="flex items-center justify-between gap-3 py-1.5"
            >
              <span className="truncate">{b.staffName}</span>
              <span className="shrink-0 text-xs text-ink-muted">
                {t.advances.of(formatPrice(b.issued), formatPrice(b.spent))}
              </span>
              {/* ⚠️ Below zero is shown, not hidden: a buyer who ran out and
                  paid for the last crate themselves is owed money, and a
                  balance clamped at zero would be silent about the one debt
                  somebody is actually waiting on. */}
              <span
                className={`shrink-0 font-semibold tabular-nums ${
                  b.balance < 0 ? "text-danger" : ""
                }`}
              >
                {formatPrice(b.balance)}
              </span>
            </li>
          ))}
        </ul>
      )}

      {open && (
        <div className="flex flex-wrap items-end gap-2 border-t border-line pt-2">
          <select
            className="input w-auto"
            value={who}
            onChange={(e) => setWho(e.target.value)}
          >
            <option value="">{t.advances.pickStaff}</option>
            {staff.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
          <select
            className="input w-auto"
            value={kind}
            onChange={(e) => setKind(e.target.value as "out" | "back")}
          >
            <option value="out">{t.advances.give}</option>
            <option value="back">{t.advances.take}</option>
          </select>
          <input
            className="input w-32"
            inputMode="numeric"
            placeholder={t.advances.amount}
            value={amount}
            onChange={(e) => setAmount(e.target.value.replace(/\D/g, ""))}
          />
          <label className="flex items-center gap-1.5 text-sm text-ink-muted">
            <input
              type="checkbox"
              checked={fromSafe}
              onChange={(e) => setFromSafe(e.target.checked)}
            />
            {kind === "out" ? t.advances.fromSafe : t.advances.toSafe}
          </label>
          <button
            className="btn-primary"
            disabled={busy}
            onClick={() => void save()}
          >
            {t.common.save}
          </button>
          {error && <p className="w-full text-sm text-danger">{error}</p>}
        </div>
      )}
    </div>
  );
}
