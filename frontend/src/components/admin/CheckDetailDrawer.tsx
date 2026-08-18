"use client";

// One dining-room sale, opened.
//
// ⚠️ **What this screen is for is disputes, not curiosity.** Somebody opens it
// because a guest rang about a bill, because a shift came up short, or because
// a manager wants to know why a table of two was charged for eight. So the
// three things it must never soften are the voids, the discount and who did
// each of them — everything else on the check is arithmetic anybody can redo.
//
// ⚠️ A drawer rather than a page: the answer is nearly always one row, and a
// full navigation would lose the filter that found it — the person is usually
// about to open the next one.

import { useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatPrice, formatTime } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { paymentLabel } from "@/lib/checkMethod";
import { printReceipt } from "@/lib/print";
import type { CheckDetail } from "@/lib/types";

export default function CheckDetailDrawer({
  id,
  onClose,
  onRefunded,
}: {
  id: string;
  onClose: () => void;
  /** So the list behind can re-read: its totals just changed. */
  onRefunded?: () => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [data, setData] = useState<CheckDetail | null>(null);
  const [error, setError] = useState("");
  const [printing, setPrinting] = useState(false);
  const [sent, setSent] = useState("");
  // ⚠️ The reason is typed before the button appears, not asked for in a
  // confirm afterwards. A dialog that asks "are you sure?" gets pressed; a
  // field that has to be filled in makes the person say what happened.
  const [refunding, setRefunding] = useState(false);
  const [reason, setReason] = useState("");

  const money = (n: number) => formatPrice(n, "UZS", lang);

  useEffect(() => {
    let alive = true;
    setData(null);
    setError("");
    api
      .adminCheck(id)
      .then((d) => alive && setData(d))
      .catch(
        (e) =>
          alive &&
          setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      );
    return () => {
      alive = false;
    };
  }, [id, t.common.loadFailed]);

  // Escape closes it. This opens and shuts dozens of times while somebody
  // works through an evening, and reaching for a small × every time is the
  // difference between a screen that gets used and one that gets avoided.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  async function print(toPrinter: boolean) {
    if (!data) return;
    setPrinting(true);
    setSent("");
    try {
      const res = await api.adminPrintCheck(data.id, toPrinter);
      // ⚠️ The lines come back either way. Sending to the counter is not a
      // substitute for seeing it: the person who pressed the button is not
      // standing next to that printer and has no way of knowing what came out.
      if (toPrinter) {
        setSent(res.queued > 0 ? t.sales.sentToPrinter : t.sales.noPrinter);
      } else {
        printReceipt(res.lines, res.widthMM, res.logoUrl);
      }
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
    } finally {
      setPrinting(false);
    }
  }

  async function refund() {
    if (!data) return;
    setPrinting(true);
    try {
      const res = await api.adminRefundCheck(data.id, reason.trim());
      // Re-read rather than patched here: whether it went through is the
      // server's answer, and the row behind this drawer has to agree.
      setData({ ...data, refunded: true, refund: res.refund });
      setRefunding(false);
      onRefunded?.();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
    } finally {
      setPrinting(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div
        className="absolute inset-0 bg-ink/30"
        onClick={onClose}
        aria-hidden
      />
      <div className="relative flex h-full w-full max-w-lg flex-col overflow-y-auto bg-surface shadow-xl">
        <div className="flex items-center justify-between gap-2 border-b border-line px-4 py-3">
          <div className="min-w-0">
            <div className="truncate font-semibold">
              {data?.number ?? t.common.loading}
            </div>
            {data && (
              <div className="truncate text-xs text-ink-muted">
                {data.table ? t.sales.table(data.table) : t.sales.counter}
                {data.guests ? ` · ${t.sales.guestsN(data.guests)}` : ""}
                {data.server ? ` · ${data.server}` : ""}
              </div>
            )}
          </div>
          <div className="flex shrink-0 items-center gap-2">
            {/* ⚠️ Two buttons, because they do different things and one of
                them happens in another building. The browser's dialog is also
                where "save as PDF" lives, which is what a bill emailed to a
                guest actually is. */}
            <button
              className="btn-ghost px-3 py-1.5"
              disabled={!data || printing}
              onClick={() => print(false)}
            >
              {t.sales.print}
            </button>
            <button className="btn-ghost px-3 py-1.5" onClick={onClose}>
              {t.common.close}
            </button>
          </div>
        </div>

        {error && <p className="p-4 text-sm text-danger">{error}</p>}

        {data && (
          <div className="space-y-4 p-4">
            <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
              <Fact label={t.sales.openedAt}>
                {formatTime(data.openedAt)}
                {data.openedBy ? ` · ${data.openedBy}` : ""}
              </Fact>
              {/* When the bill was printed for the table. The one fact the
                  floor screen cannot work out for itself, and the one a guest
                  disputing a wait will ask about. */}
              {data.precheckAt && (
                <Fact label={t.sales.precheckAt}>
                  {formatTime(data.precheckAt)}
                </Fact>
              )}
              <Fact label={t.sales.closedAt}>
                {data.closedAt ? (
                  <>
                    {formatTime(data.closedAt)}
                    {data.closedBy ? ` · ${data.closedBy}` : ""}
                  </>
                ) : (
                  <span className="text-ink-muted">{t.sales.stillOpen}</span>
                )}
              </Fact>
              {!data.open && (
                <Fact label={t.sales.method}>
                  {paymentLabel(data.paymentMethod, t)}
                </Fact>
              )}
            </dl>

            <ul className="divide-y divide-line">
              {data.lines.map((l, i) => {
                const voided = !!l.voidReason || (l.sum === 0 && !!l.voidedAt);
                return (
                  <li key={i} className="py-2 text-sm">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <span
                          className={voided ? "line-through opacity-60" : ""}
                        >
                          {l.qty} × {l.name}
                        </span>
                        {/* Guest and course only when they were used: a
                            counter selling coffee never numbers either, and a
                            "guest 0, course 0" on every row is noise that
                            teaches people to skip the line. */}
                        {(l.guest || l.course) && (
                          <span className="ml-1.5 text-xs text-ink-muted">
                            {l.guest ? t.sales.guestNo(l.guest) : ""}
                            {l.guest && l.course ? " · " : ""}
                            {l.course ? t.sales.courseNo(l.course) : ""}
                          </span>
                        )}
                        {l.options?.length ? (
                          <div className="text-xs text-ink-muted">
                            {l.options.map((o) => o.choice).join(", ")}
                          </div>
                        ) : null}
                        {l.comment && (
                          <div className="text-xs text-ink-soft">
                            « {l.comment} »
                          </div>
                        )}
                        {voided && (
                          <div className="text-xs text-danger">
                            {t.sales.voided}: {l.voidReason}
                            {l.voidedBy ? ` · ${l.voidedBy}` : ""}
                            {/* Whether the food was actually made. A kitchen
                                that caught it in time and a plate in the bin
                                are different losses. */}
                            {l.wasted ? ` · ${t.sales.wasted}` : ""}
                          </div>
                        )}
                      </div>
                      <span
                        className={`shrink-0 tabular-nums ${voided ? "text-ink-muted line-through" : ""}`}
                      >
                        {money(voided ? l.price * l.qty : l.sum)}
                      </span>
                    </div>
                  </li>
                );
              })}
            </ul>

            <dl className="space-y-1 border-t border-line pt-3 text-sm">
              <Line label={t.sales.subtotal} value={money(data.subtotal)} />
              {/* ⚠️ Between the subtotal and the total, where the arithmetic
                  happens. A bill whose parts do not add up is the one thing on
                  this screen a guest rings about — and the person answering is
                  reading exactly this. */}
              {data.service ? (
                <Line
                  label={`${t.sales.service}${
                    data.servicePercent ? ` ${data.servicePercent}%` : ""
                  }`}
                  value={money(data.service)}
                  muted
                />
              ) : null}
              {data.discounts?.map((d, i) => (
                <Line
                  key={i}
                  label={d.name}
                  value={`−${money(d.amount)}`}
                  muted
                />
              ))}
              <Line label={t.sales.total} value={money(data.total)} strong />
            </dl>

            {/* An unfiled sale is the only thing here somebody has to act on,
                and the register's own words usually name something fixable in
                seconds. */}
            {data.fiscalError && (
              <p className="rounded-xl border border-danger/40 bg-danger/5 p-2 text-xs text-danger">
                {t.sales.unfiled}: {data.fiscalError}
              </p>
            )}
            {/* ---- Handing the money back ----

                ⚠️ Only on a settled sale, and never twice: the server guards
                both, but a button that is there and refuses is a button people
                learn to press twice. */}
            {!data.open && !data.refunded && (
              <div className="border-t border-line pt-3">
                {refunding ? (
                  <div className="space-y-2">
                    <input
                      className="input"
                      autoFocus
                      placeholder={t.sales.refundReason}
                      value={reason}
                      onChange={(e) => setReason(e.target.value)}
                    />
                    <div className="flex gap-2">
                      <button
                        className="btn-ghost px-3 py-1.5 text-sm"
                        onClick={() => setRefunding(false)}
                      >
                        {t.common.cancel}
                      </button>
                      <button
                        className="btn-primary px-3 py-1.5 text-sm"
                        disabled={!reason.trim() || printing}
                        onClick={refund}
                      >
                        {t.sales.refundConfirm(money(data.total))}
                      </button>
                    </div>
                  </div>
                ) : (
                  <button
                    className="btn-ghost px-3 py-1.5 text-sm text-danger"
                    onClick={() => setRefunding(true)}
                  >
                    {t.sales.refund}
                  </button>
                )}
              </div>
            )}

            {/* What was handed back, and why. The sentence is the record. */}
            {data.refund && (
              <p className="rounded-xl border border-line bg-ink/[0.03] p-2 text-xs text-ink-soft">
                {t.sales.refunded}: {money(data.refund.amount)} ·{" "}
                {formatTime(data.refund.at)}
                {data.refund.by ? ` · ${data.refund.by}` : ""}
                <br />
                {data.refund.reason}
              </p>
            )}

            <div className="flex flex-wrap items-center gap-2 border-t border-line pt-3">
              <button
                className="btn-ghost px-3 py-1.5 text-sm"
                disabled={printing}
                onClick={() => print(true)}
              >
                {t.sales.printAtBranch}
              </button>
              {sent && <span className="text-xs text-ink-muted">{sent}</span>}
            </div>

            {data.fiscalSign && (
              <p className="text-xs text-ink-muted">
                {t.sales.fiscalSign}: {data.fiscalSign}
              </p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function Fact({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <dt className="text-xs text-ink-muted">{label}</dt>
      <dd>{children}</dd>
    </div>
  );
}

function Line({
  label,
  value,
  muted,
  strong,
}: {
  label: string;
  value: string;
  muted?: boolean;
  strong?: boolean;
}) {
  return (
    <div
      className={`flex justify-between gap-3 ${muted ? "text-ink-muted" : ""} ${
        strong ? "font-semibold" : ""
      }`}
    >
      <span className="min-w-0 truncate">{label}</span>
      <span className="shrink-0 tabular-nums">{value}</span>
    </div>
  );
}
