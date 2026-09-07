"use client";

// The central store's van, and the slip that travels with it.
//
// ⚠️ **The movement a chain has every morning and this panel could not
// record.** Everything else moves stock inside one branch; a central kitchen
// that buys the meat, marinates it on Monday and sends it to four branches had
// no document at all. The two ways of faking it both lie — a write-off puts
// food nobody wasted on the waste report, and a delivery at the far end writes
// a purchase price into the price history and costs every dish it goes into.
//
// ⚠️ **One screen for both ends of the van.** A branch is a sender in the
// morning and a receiver in the afternoon; two screens would mean the person
// chasing a missing crate has to know which half they are in before they can
// look for it.
//
// ⚠️ **The paper is the point.** The slip is printed here, carried by a driver
// and signed three times — and its language is chosen at the moment of
// printing, because whose language it is in is a question about the person
// receiving it. See lib/nakladnoy.ts.

import { useCallback, useEffect, useMemo, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatDate, formatPrice } from "@/lib/format";
import { qtyNumber } from "@/lib/qty";
import { QtyInput } from "@/components/QtyInput";
import { adminEn, adminRu, adminUz, useAdminT } from "@/lib/i18n/admin";
import type { AdminDict } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import { printNakladnoy } from "@/lib/nakladnoy";
import type {
  Branch,
  DispatchRow,
  DispatchStockRow,
} from "@/lib/types";

/** The three the slip can be printed in. ⚠️ Not the panel's language: the
 *  storekeeper works in one and the branch foreman signing at the far end may
 *  read another, and that changes from van to van. */
const PRINT_LANGS: { code: string; label: string; dict: AdminDict }[] = [
  { code: "uz", label: "O'zbekcha", dict: adminUz },
  { code: "ru", label: "Русский", dict: adminRu },
  { code: "en", label: "English", dict: adminEn },
];

type Draft = { ingredientId: string; qty: string };

export default function DispatchPage() {
  const t = useAdminT();
  const scope = useAdminScope();
  const [rows, setRows] = useState<DispatchRow[]>([]);
  const [stock, setStock] = useState<DispatchStockRow[]>([]);
  const [branches, setBranches] = useState<Branch[]>([]);
  const [to, setTo] = useState("");
  const [driver, setDriver] = useState("");
  const [note, setNote] = useState("");
  const [lines, setLines] = useState<Draft[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [printLang, setPrintLang] = useState("uz");
  // Why the item list is empty, when it is.
  const [stockError, setStockError] = useState("");
  // Which incoming van is being signed for, and what the branch counted.
  const [signing, setSigning] = useState("");
  const [got, setGot] = useState<Record<string, string>>({});

  const load = useCallback(() => {
    api
      .adminDispatches()
      .then((d) => setRows(d.rows))
      .catch((e) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      );
    // ⚠️ **The failure is shown, never swallowed.** A dispatch leaves one
    // store, so the server refuses to name a shelf while the panel is looking
    // at every branch at once — and a caught-and-ignored error left the item
    // list empty with no reason on the screen, which reads as a broken page
    // rather than as a lens that has not been chosen.
    api
      .adminDispatchStock()
      .then((d) => {
        setStock(d.rows);
        setStockError("");
      })
      .catch((e) => {
        setStock([]);
        setStockError(e instanceof ApiError ? e.message : t.common.loadFailed);
      });
    api
      .adminBranches()
      .then((d) => setBranches(d))
      .catch(() => setBranches([]));
  }, [t.common.loadFailed]);

  useEffect(load, [load, scope.scopeKey]);

  const byId = useMemo(() => {
    const m: Record<string, DispatchStockRow> = {};
    for (const r of stock) m[r.id] = r;
    return m;
  }, [stock]);

  // ⚠️ The branch that is doing the sending is the one in view, so it is not
  // offered as a destination — a van to itself is not a movement.
  const targets = branches.filter(
    (b) => !scope.branch || b.id !== scope.branch.id,
  );

  async function save() {
    const body = lines
      .map((l) => ({ ingredientId: l.ingredientId, qty: qtyNumber(l.qty) }))
      .filter((l) => l.ingredientId && l.qty > 0);
    if (!to || body.length === 0) return;
    setBusy(true);
    setError("");
    try {
      await api.adminCreateDispatch({
        toBranchId: to,
        driver: driver.trim(),
        note: note.trim(),
        lines: body,
      });
      setLines([]);
      setDriver("");
      setNote("");
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  async function accept(row: DispatchRow) {
    setBusy(true);
    setError("");
    try {
      await api.adminAcceptDispatch(
        row.dispatch.id,
        // ⚠️ Only what differs travels: silence means "the slip was right",
        // which is the ordinary case and must not need typing forty numbers.
        row.dispatch.lines
          .map((l) => ({
            ingredientId: l.ingredientId,
            got: qtyNumber(got[`${row.dispatch.id}:${l.ingredientId}`] ?? ""),
          }))
          .filter((l) => l.got > 0),
      );
      setSigning("");
      setGot({});
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  /** Print every slip of one day at once, four to a sheet — which is how the
   *  paper form this copies is used: one sheet, one morning, four branches. */
  function printDay(day: string) {
    const dict =
      PRINT_LANGS.find((l) => l.code === printLang)?.dict ?? adminUz;
    printNakladnoy(
      rows
        .filter(
          (r) =>
            r.dispatch.at.slice(0, 10) === day &&
            (!scope.branch || r.dispatch.fromBranchId === scope.branch.id),
        )
        .map((r) => ({ dispatch: r.dispatch, to: r.to, from: r.from })),
      dict,
    );
  }

  const days = useMemo(() => {
    const seen: string[] = [];
    for (const r of rows) {
      const day = r.dispatch.at.slice(0, 10);
      if (!seen.includes(day)) seen.push(day);
    }
    return seen;
  }, [rows]);

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-semibold">{t.dispatch.title}</h1>
        <p className="mt-1 max-w-3xl text-sm text-ink-soft">
          {t.dispatch.intro}
        </p>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      {/* ---- Loading a van ---- */}
      <div className="card space-y-3 p-4">
        <div className="font-medium">{t.dispatch.newTitle}</div>
        {/* ⚠️ **A van is loaded from one store**, so the branch has to be
            chosen before anything can be put on it. Said here rather than left
            as an empty dropdown: the same rule every other store screen
            follows (handlers/placements.go), and the same sentence. */}
        {scope.loading ? null : !scope.branch ? (
          <p className="rounded-2xl border border-amber-500/40 bg-amber-500/10 px-4 py-3 text-sm text-amber-700">
            {t.dispatch.pickBranch}
          </p>
        ) : stockError ? (
          <p className="text-sm text-danger">{stockError}</p>
        ) : stock.length === 0 ? (
          <p className="text-sm text-ink-muted">{t.dispatch.noStock}</p>
        ) : null}
        <div
          className={`grid gap-3 sm:grid-cols-3 ${
            scope.branch || scope.loading ? "" : "hidden"
          }`}
        >
          <label className="text-sm">
            <span className="text-ink-muted">{t.dispatch.to}</span>
            <select
              value={to}
              onChange={(e) => setTo(e.target.value)}
              className="mt-1 w-full rounded-xl border border-line bg-surface px-3 py-2"
            >
              <option value="">—</option>
              {targets.map((b) => (
                <option key={b.id} value={b.id}>
                  {b.name}
                </option>
              ))}
            </select>
          </label>
          <label className="text-sm">
            <span className="text-ink-muted">{t.dispatch.driver}</span>
            <input
              value={driver}
              onChange={(e) => setDriver(e.target.value)}
              placeholder={t.dispatch.driverPh}
              className="mt-1 w-full rounded-xl border border-line bg-surface px-3 py-2"
            />
          </label>
          <label className="text-sm">
            <span className="text-ink-muted">{t.dispatch.note}</span>
            <input
              value={note}
              onChange={(e) => setNote(e.target.value)}
              className="mt-1 w-full rounded-xl border border-line bg-surface px-3 py-2"
            />
          </label>
        </div>

        {lines.map((l, i) => (
          <div key={i} className="flex flex-wrap items-end gap-2">
            <label className="min-w-[14rem] flex-1 text-sm">
              <select
                value={l.ingredientId}
                onChange={(e) => {
                  const next = [...lines];
                  next[i] = { ...l, ingredientId: e.target.value };
                  setLines(next);
                }}
                className="w-full rounded-xl border border-line bg-surface px-3 py-2"
              >
                <option value="">—</option>
                {stock.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name}
                  </option>
                ))}
              </select>
            </label>
            <div className="w-32">
              <QtyInput
                value={l.qty}
                onValue={(v) => {
                  const next = [...lines];
                  next[i] = { ...l, qty: v };
                  setLines(next);
                }}
              />
            </div>
            {/* ⚠️ What is left on the shelf, beside the box being typed into:
                a storekeeper promising thirty kilos to one branch needs to see
                what that leaves for the next van, at the moment they promise
                it rather than at the next count. */}
            <span className="pb-2 text-xs text-ink-muted">
              {l.ingredientId && byId[l.ingredientId]
                ? t.dispatch.onHand(
                    `${byId[l.ingredientId].qty} ${byId[l.ingredientId].unit}`,
                  )
                : ""}
            </span>
            <button
              type="button"
              onClick={() => setLines(lines.filter((_, j) => j !== i))}
              className="pb-2 text-xs text-ink-muted underline"
            >
              {t.dispatch.remove}
            </button>
          </div>
        ))}

        <div
          className={`flex flex-wrap gap-2 ${
            scope.branch || scope.loading ? "" : "hidden"
          }`}
        >
          <button
            type="button"
            onClick={() => setLines([...lines, { ingredientId: "", qty: "" }])}
            className="rounded-xl border border-line px-4 py-2 text-sm"
          >
            {t.dispatch.pick}
          </button>
          <button
            type="button"
            disabled={busy || !to || lines.length === 0}
            onClick={save}
            className="rounded-xl bg-brand px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
          >
            {t.dispatch.save}
          </button>
        </div>
      </div>

      {/* ---- The language the paper is printed in ---- */}
      <div className="flex flex-wrap items-center gap-2 text-sm">
        <span className="text-ink-muted">{t.dispatch.printLang}:</span>
        {PRINT_LANGS.map((l) => (
          <button
            key={l.code}
            type="button"
            onClick={() => setPrintLang(l.code)}
            className={`rounded-lg border px-3 py-1.5 font-semibold ${
              l.code === printLang ? "border-brand bg-brand/10" : "border-line"
            }`}
          >
            {l.label}
          </button>
        ))}
      </div>

      {/* ---- What went, and what is owed ---- */}
      {rows.length === 0 ? (
        <div className="card p-6 text-center text-sm text-ink-muted">
          {t.dispatch.empty}
        </div>
      ) : (
        days.map((day) => (
          <div key={day} className="space-y-2">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div className="text-sm font-medium">{formatDate(day)}</div>
              {/* One sheet, one morning, four branches. */}
              <button
                type="button"
                onClick={() => printDay(day)}
                className="rounded-xl border border-line px-3 py-1.5 text-sm font-semibold"
              >
                {t.dispatch.print}
              </button>
            </div>
            {rows
              .filter((r) => r.dispatch.at.slice(0, 10) === day)
              .map((r) => {
                const d = r.dispatch;
                // ⚠️ Whether this branch is the sender decides which half of
                // the van the row is showing — and with no branch chosen (an
                // owner looking at the whole company) every row is "ours",
                // which is the honest answer: they are on both ends of it.
                const mine = !scope.branch || d.fromBranchId === scope.branch.id;
                const lost = d.acceptedAt
                  ? d.lines.filter((l) => l.got !== undefined && l.got !== l.qty)
                  : [];
                return (
                  <div key={d.id} className="card p-3">
                    <div className="flex flex-wrap items-baseline justify-between gap-2">
                      <div className="text-sm">
                        <span className="font-semibold">{d.number}</span>
                        <span className="ml-2 text-ink-muted">
                          {mine
                            ? `${t.dispatch.outgoing}: ${r.to}`
                            : `${t.dispatch.incoming}: ${r.from}`}
                        </span>
                      </div>
                      <div className="flex items-center gap-3 text-sm">
                        <span className="tabular-nums text-ink-soft">
                          {formatPrice(d.value)}
                        </span>
                        {/* ⚠️ "Not accepted" is a state worth naming: until
                            somebody signs, the goods are on nobody's shelf. */}
                        <span
                          className={
                            d.acceptedAt
                              ? "text-xs text-ink-muted"
                              : "text-xs font-semibold text-amber-700"
                          }
                        >
                          {d.acceptedAt
                            ? `${t.dispatch.received}: ${d.acceptedBy ?? ""}`
                            : t.dispatch.waiting}
                        </span>
                      </div>
                    </div>

                    <div className="mt-2 text-xs text-ink-muted">
                      {d.lines
                        .map((l) => `${l.name} — ${l.qty} ${l.unit}`)
                        .join(" · ")}
                    </div>
                    {lost.length > 0 && (
                      <div className="mt-1 text-xs font-medium text-danger">
                        {lost
                          .map((l) =>
                            t.dispatch.lost(
                              `${l.name} ${Math.round((l.qty - (l.got ?? 0)) * 1000) / 1000} ${l.unit}`,
                            ),
                          )
                          .join(" · ")}
                      </div>
                    )}

                    {/* Signing for it: only the receiving branch is offered
                        the button, and the server enforces the same. */}
                    {!mine && !d.acceptedAt && (
                      <div className="mt-2">
                        {signing === d.id ? (
                          <div className="space-y-2">
                            <p className="text-xs text-ink-muted">
                              {t.dispatch.acceptHint}
                            </p>
                            {d.lines.map((l) => (
                              <div
                                key={l.ingredientId}
                                className="flex items-center gap-2 text-sm"
                              >
                                <span className="flex-1">
                                  {l.name}{" "}
                                  <span className="text-ink-muted">
                                    {l.qty} {l.unit}
                                  </span>
                                </span>
                                <div className="w-28">
                                  <QtyInput
                                    value={
                                      got[`${d.id}:${l.ingredientId}`] ??
                                      String(l.qty)
                                    }
                                    onValue={(v) =>
                                      setGot({
                                        ...got,
                                        [`${d.id}:${l.ingredientId}`]: v,
                                      })
                                    }
                                  />
                                </div>
                              </div>
                            ))}
                            <button
                              type="button"
                              disabled={busy}
                              onClick={() => accept(r)}
                              className="rounded-xl bg-brand px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
                            >
                              {t.dispatch.acceptSave}
                            </button>
                          </div>
                        ) : (
                          <button
                            type="button"
                            onClick={() => setSigning(d.id)}
                            className="rounded-xl border border-line px-3 py-1.5 text-sm font-semibold"
                          >
                            {t.dispatch.accept}
                          </button>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}
          </div>
        ))
      )}
    </div>
  );
}
