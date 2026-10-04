"use client";

import { useCallback, useEffect, useMemo, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { formatDateTime } from "@/lib/orderFlow";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { printReceipt, type PrintOutcome } from "@/lib/print";
import PrintResultDialog from "./PrintResultDialog";
import type { Check } from "@/lib/types";
import { TillPager, usePaged } from "./Pager";

/**
 * The sales list, on the till.
 *
 * ⚠️ **Two tabs because they answer two questions, not because they are two
 * lists.** "What is still open" is the room's question, already answered by the
 * floor — but a counter with no tables has no floor to read, and a list is the
 * only shape that works there. "What has been sold" is the cashier's question,
 * and until now the only way to ask it was a manager's login on a machine
 * standing in a dining room.
 *
 * ⚠️ **Today only.** A till that can page back through the month is a till
 * worth stealing, and that question belongs to the owner's screen anyway.
 */
export default function ChecksScreen({
  open,
  currency,
  onOpenCheck,
  onError,
}: {
  /** The open checks the till already polls for — not fetched twice. */
  open: Check[];
  currency: string;
  onOpenCheck: (c: Check) => void;
  onError: (msg: string) => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [tab, setTab] = useState<"open" | "closed">("open");
  const [closed, setClosed] = useState<Check[]>([]);
  const [totals, setTotals] = useState({
    total: 0,
    refunded: 0,
    owed: 0,
    count: 0,
  });
  const [query, setQuery] = useState("");
  const [picked, setPicked] = useState<Check | null>(null);
  const [busy, setBusy] = useState(false);
  const [printed, setPrinted] = useState<PrintOutcome | null>(null);

  const loadClosed = useCallback(async () => {
    try {
      const res = await api.tillClosedChecks();
      setClosed(res.checks);
      setTotals({
        total: res.total,
        refunded: res.refunded,
        owed: res.owed,
        count: res.count,
      });
    } catch {
      // ⚠️ Silent. A till goes offline several times a service, and this list
      // is the one screen where that costs nothing — the sales are on the
      // server whether or not this tablet can see them right now.
    }
  }, []);

  useEffect(() => {
    if (tab === "closed") void loadClosed();
  }, [tab, loadClosed]);

  // ⚠️ Filtered here, over a list already in hand. The server's own `q` exists
  // for the same search, but typing into a box that goes to the network on
  // every letter is the slowest a screen can feel — and this one is short by
  // construction.
  const rows = useMemo(() => {
    const list = tab === "open" ? open : closed;
    const q = query.trim().toLowerCase();
    if (!q) return list;
    return list.filter(
      (c) =>
        c.number.toLowerCase().includes(q) ||
        (c.tableNumber ?? "").toLowerCase().includes(q) ||
        (c.serverName ?? "").toLowerCase().includes(q),
    );
  }, [tab, open, closed, query]);
  const paged = usePaged(rows);
  // A new tab or a new search starts at its first page.
  const { setPage } = paged;
  useEffect(() => setPage(0), [tab, query, setPage]);

  async function reprint(check: Check) {
    setBusy(true);
    try {
      const res = await api.tillPrint(check.id, "customer");
      // ⚠️ **Answered either way, because reprinting is the whole action.**
      // A queued job went to a branch printer and this screen is not the one
      // that watches it; anything else printed here, and the outcome is all
      // there is to show for the press.
      if (res.queued > 0) {
        setPrinted("printed");
      } else {
        setPrinted(await printReceipt(res.lines, res.widthMM, res.logoUrl));
      }
    } catch (e) {
      onError(e instanceof ApiError ? e.message : t.till.retry);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {printed && (
        <PrintResultDialog outcome={printed} onClose={() => setPrinted(null)} />
      )}
      <div className="flex shrink-0 flex-wrap items-center gap-2 border-b border-line bg-surface px-3 py-2.5">
        <div className="till-seg-track">
          <button
            className={tab === "open" ? "till-seg-on" : "till-seg"}
            onClick={() => setTab("open")}
          >
            {t.till.openChecks} ({open.length})
          </button>
          <button
            className={tab === "closed" ? "till-seg-on" : "till-seg"}
            onClick={() => setTab("closed")}
          >
            {t.till.closedChecks}
          </button>
        </div>
        <input
          className="till-input h-11 min-w-40 flex-1"
          // ⚠️ Not the menu's placeholder: this box searches checks, and a
          // cashier told to "search dishes" types a dish name and concludes
          // the list is broken.
          placeholder={t.till.searchChecks}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
      </div>

      {/* ⚠️ The shift's own running total, and what went back beside it. A list
          of closed checks with no sum at the top is a list somebody adds up by
          hand — and the number they arrive at is the one they act on. */}
      {tab === "closed" && (
        <div className="flex shrink-0 flex-wrap items-baseline gap-x-4 gap-y-1 border-b border-line px-4 py-2 text-sm">
          <span className="text-ink-muted">
            {t.till.closedCount(totals.count)}
          </span>
          <span className="font-display text-lg font-bold tabular-nums">
            {formatPrice(totals.total, currency, lang)}
          </span>
          {/* ⚠️ Beside the total, never inside it: this bar is read standing
              at the drawer, and a slate folded into the takings says money is
              here that is not. */}
          {totals.owed > 0 && (
            <span className="text-ink-muted tabular-nums">
              {t.till.owedSum(formatPrice(totals.owed, currency, lang))}
            </span>
          )}
          {totals.refunded > 0 && (
            <span className="text-danger tabular-nums">
              {t.till.refundedSum(
                formatPrice(totals.refunded, currency, lang),
              )}
            </span>
          )}
        </div>
      )}

      <div className="min-h-0 flex-1 overflow-y-auto p-3">
        {rows.length === 0 && (
          <p className="p-6 text-center text-sm text-ink-muted">
            {tab === "open" ? t.till.noChecks : t.till.noClosedChecks}
          </p>
        )}
        <div className="mx-auto max-w-3xl space-y-2">
          {paged.shown.map((c) => {
            const cancelled = c.status === "cancelled";
            const refunded = !!c.refund;
            return (
              <button
                key={c.id}
                className="till-row min-h-16 w-full border-line py-2"
                // ⚠️ An open check goes to the check screen; a closed one opens
                // a card here. Sending a closed sale to the editing screen
                // would put "add a dish" in front of somebody looking at a
                // receipt that has already been paid.
                onClick={() =>
                  tab === "open" ? onOpenCheck(c) : setPicked(c)
                }
              >
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-medium">
                      {c.tableNumber
                        ? t.till.tableNo(c.tableNumber)
                        : t.till.counter}
                    </span>
                    <span className="text-sm text-ink-muted">{c.number}</span>
                    {/* Both are the reason somebody came looking, so both are
                        on the row rather than inside it. */}
                    {cancelled && (
                      <span className="text-xs font-semibold text-danger">
                        {t.till.cancelledBadge}
                      </span>
                    )}
                    {refunded && (
                      <span className="text-xs font-semibold text-danger">
                        {t.till.refundedBadge}
                      </span>
                    )}
                  </div>
                  {/* ⚠️ The reason on the row itself: a list of red
                      "cancelled" badges with nothing beside them is the list
                      somebody has to open one by one to find the odd one. */}
                  {cancelled && (
                    <div className="truncate text-xs text-danger">
                      {t.till.cancelReasonLabel}:{" "}
                      {c.cancelReason?.trim() || t.till.noReason}
                    </div>
                  )}
                  <div className="text-xs text-ink-muted">
                    {tab === "open"
                      ? [c.serverName, t.till.minutes(c.openMin)]
                          .filter(Boolean)
                          .join(" · ")
                      : [
                          c.closedAt ? formatDateTime(c.closedAt) : "",
                          c.paymentMethod
                            ? (t.till.methodName[c.paymentMethod] ??
                              c.paymentMethod)
                            : "",
                          c.closedBy,
                        ]
                          .filter(Boolean)
                          .join(" · ")}
                  </div>
                </div>
                <span
                  className={`font-display text-lg font-bold tabular-nums ${
                    cancelled || refunded ? "text-ink-muted line-through" : ""
                  }`}
                >
                  {formatPrice(c.total, currency, lang)}
                </span>
              </button>
            );
          })}
        </div>
        <TillPager page={paged.page} pages={paged.pages} onPage={paged.setPage} />
      </div>

      {/* ---- One closed sale ----

          ⚠️ Read-only, and the only action is paper. Refunding is a manager's
          decision with a reason attached and it lives in the panel; putting it
          one tap from a list of every sale in the room would make "the guest
          complained" and "the money went back" the same gesture. */}
      {picked && (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-ink/40 p-4 sm:items-center">
          <div className="till-dialog max-h-[85dvh] w-full max-w-md overflow-y-auto p-4">
            <div className="flex items-baseline justify-between gap-2">
              <h2 className="font-display text-xl font-bold">
                {picked.tableNumber
                  ? t.till.tableNo(picked.tableNumber)
                  : t.till.counter}
              </h2>
              <span className="text-sm text-ink-muted">{picked.number}</span>
            </div>
            {/* ⚠️ How it was paid belongs on the card, not only on the row.
                Half of what this screen is opened for is "did that one go
                through as cash", and an answer that is only visible in the
                list behind the dialog is an answer nobody trusts. */}
            <p className="mt-1 text-sm text-ink-muted">
              {[
                picked.closedAt ? formatDateTime(picked.closedAt) : "",
                picked.paymentMethod
                  ? (t.till.methodName[picked.paymentMethod] ??
                    picked.paymentMethod)
                  : "",
                picked.closedBy,
              ]
                .filter(Boolean)
                .join(" · ")}
            </p>

            <div className="mt-3 space-y-1 text-sm">
              {picked.lines.map((l) => (
                <div
                  key={l.lineId}
                  className={`flex justify-between gap-2 ${
                    l.void ? "text-ink-muted" : ""
                  }`}
                >
                  <span className="min-w-0">
                    <span className={l.void ? "line-through" : ""}>
                      {l.qty} × {l.name}
                    </span>
                    {/* A removed line says why, the same way the check does —
                        and not struck through, or the reason reads as deleted
                        too. */}
                    {l.void?.reason && (
                      <span className="block text-xs">
                        {l.void.reason}
                        {l.void.by ? ` · ${l.void.by}` : ""}
                      </span>
                    )}
                  </span>
                  <span className={`tabular-nums ${l.void ? "line-through" : ""}`}>
                    {formatPrice(l.sum, currency, lang)}
                  </span>
                </div>
              ))}
            </div>

            <div className="mt-3 flex justify-between border-t border-line pt-2 font-display text-lg font-bold">
              <span>{t.till.total}</span>
              <span className="tabular-nums">
                {formatPrice(picked.total, currency, lang)}
              </span>
            </div>
            {/* ⚠️ The reason, not just the amount: "refunded 240 000" with no
                sentence beside it is the line every argument about a shift
                starts from, and the person who could answer has gone home. */}
            {/* ⚠️ The cancellation's own sentence, and who wrote it. The reason
                was required on the way in and then never shown again — a red
                "cancelled" and a struck-through total were all the till could
                say about the one kind of check every evening's questions start
                from. */}
            {picked.status === "cancelled" && (
              <div className="mt-3 rounded-[12px] border border-danger/30 bg-danger/[0.06] px-3 py-2 text-sm">
                <p className="font-semibold text-danger">{t.till.cancelledBadge}</p>
                <p className="mt-0.5">
                  <span className="text-ink-muted">{t.till.cancelReasonLabel}: </span>
                  {picked.cancelReason?.trim() || t.till.noReason}
                </p>
                {picked.closedBy && (
                  <p className="mt-0.5">
                    <span className="text-ink-muted">{t.till.cancelledBy}: </span>
                    {picked.closedBy}
                  </p>
                )}
              </div>
            )}
            {picked.refund && (
              <p className="mt-2 text-sm text-danger">
                {t.till.refundedBadge}: {picked.refund.reason} ·{" "}
                {formatPrice(picked.refund.amount, currency, lang)}
              </p>
            )}

            <div className="mt-4 flex gap-2">
              <button
                className="till-btn flex-1"
                onClick={() => setPicked(null)}
              >
                {t.till.back}
              </button>
              <button
                className="till-btn-primary flex-1"
                disabled={busy}
                onClick={() => void reprint(picked)}
              >
                {t.till.printReceipt}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
