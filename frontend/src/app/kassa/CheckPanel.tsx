"use client";

import { useState } from "react";

import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { formatPrice } from "@/lib/format";
import type { Check, CheckLine, FloorTable } from "@/lib/types";

import PayDialog from "./PayDialog";
import VoidDialog from "@/components/till/VoidDialog";
import OverrideDialog from "@/components/till/OverrideDialog";
import MoveTableDialog from "@/components/till/MoveTableDialog";

/**
 * The right-hand column: what this table has, and what happens next.
 *
 * ⚠️ **Two buttons, and they are not the same button.** "Send to the kitchen"
 * commits food; "Pay" commits money. Every till that merges them ends up either
 * cooking things nobody has paid for or charging for things nobody cooked, and
 * the restaurant discovers which one it is a week later.
 *
 * ⚠️ **Fired and unfired lines look different on purpose.** Once a line is with
 * the kitchen, taking it off costs the restaurant food and needs a cashier and a
 * reason; before that it is a typo. The screen has to make the moment of no
 * return visible, because it is not visible in the room.
 */
export default function CheckPanel({
  check,
  currency,
  canCashier,
  tables,
  busyTables,
  onChange,
  onClosed,
  onError,
}: {
  check: Check | null;
  currency: string;
  canCashier: boolean;
  /** The room, for moving a party. */
  tables: FloorTable[];
  /** Tables that already have a check on them. */
  busyTables: string[];
  onChange: (next: Check) => void;
  onClosed: () => void;
  onError: (msg: string) => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [busy, setBusy] = useState(false);
  const [paying, setPaying] = useState(false);
  const [voiding, setVoiding] = useState<CheckLine | null>(null);
  // The void the server asked a manager to authorise, held so the retry sends
  // the same reason rather than asking the waiter to type it twice.
  const [override, setOverride] = useState<PendingVoid | null>(null);
  const [overrideError, setOverrideError] = useState("");
  const [cancelling, setCancelling] = useState(false);
  const [moving, setMoving] = useState(false);

  if (!check) {
    return (
      <aside className="flex w-full items-center justify-center p-6">
        <p className="text-center text-sm text-ink-muted">
          {t.till.emptyCheck}
        </p>
      </aside>
    );
  }

  async function run(fn: () => Promise<Check>) {
    setBusy(true);
    try {
      onChange(await fn());
    } catch (err) {
      onError(err instanceof ApiError ? err.message : t.till.retry);
    } finally {
      setBusy(false);
    }
  }

  const id = check.id;

  /** Void a line, asking for a manager's code if this person may not.
   *
   *  ⚠️ **The reason survives the refusal.** Making somebody retype why a steak
   *  was burnt because they were not allowed the first time is how reasons
   *  become "." — and the reason is the entire point of the record.
   */
  async function tryVoid(
    lineId: string,
    reason: string,
    wasted: boolean,
    pin: string,
  ) {
    setBusy(true);
    setOverrideError("");
    try {
      onChange(await api.tillVoidLine(id, lineId, { reason, wasted, pin }));
      setOverride(null);
    } catch (err) {
      const perm = err instanceof ApiError ? err.needsOverride : null;
      if (perm) {
        // Asked again with a wrong or unprivileged code: keep the dialog open
        // and say so, rather than closing it and losing the reason.
        if (pin) setOverrideError(t.till.overrideWrong);
        setOverride({
          lineId,
          reason,
          wasted,
          permissionName: err instanceof ApiError ? err.permissionName : "",
        });
      } else {
        onError(err instanceof ApiError ? err.message : t.till.retry);
        setOverride(null);
      }
    } finally {
      setBusy(false);
    }
  }

  async function removeLine(line: CheckLine) {
    // An unfired line is a typo being corrected — no dialog, no reason. Asking
    // a waiter to justify fixing their own mistyping is how the reasons on the
    // real voids become worthless.
    if (!line.fired) {
      await run(() => api.tillVoidLine(id, line.lineId));
      return;
    }
    setVoiding(line);
  }

  const live = check.lines.filter((l) => !l.void);

  return (
    <aside className="till-panel flex w-full flex-col overflow-hidden">
      <header className="shrink-0 border-b border-line px-3 py-2">
        <div className="flex items-baseline justify-between gap-2">
          <h2 className="font-display text-base font-bold">
            {check.tableNumber
              ? `${check.tableNumber}-${t.till.table.toLowerCase()}`
              : t.till.counter}
          </h2>
          <span className="text-xs text-ink-muted">{check.number}</span>
        </div>
        <p className="text-xs text-ink-muted">
          {check.serverName}
          {check.guests ? ` · ${t.till.guests}: ${check.guests}` : ""}
          {` · ${check.openMin} ${t.till.minShort}`}
        </p>
      </header>

      <ul className="min-h-0 flex-1 overflow-y-auto px-3">
        {live.length === 0 && (
          <li className="py-6 text-center text-sm text-ink-muted">
            {t.till.emptyCheck}
          </li>
        )}
        {check.lines.map((line) => (
          <li
            key={line.lineId}
            className={`flex items-start gap-2 border-b border-line py-2 ${
              line.void ? "opacity-50" : ""
            }`}
          >
            <div className="min-w-0 flex-1">
              <div className="flex items-baseline gap-1.5">
                <span
                  className={`truncate text-sm ${
                    line.void ? "line-through" : "font-medium"
                  }`}
                >
                  {line.name}
                </span>
                {!line.void && !line.fired && (
                  // The one badge on the screen. A line the kitchen has not
                  // seen is the thing that gets forgotten during a rush.
                  <span className="badge shrink-0 text-[10px]">
                    {t.till.pendingLabel}
                  </span>
                )}
              </div>
              <div className="text-xs text-ink-muted">
                {line.qty} × {formatPrice(line.price, currency, lang)}
              </div>
              {/* ⚠️ **Which option was chosen, on the line.** Without it two
                  "Osh (palov)" rows at different prices look like a pricing
                  bug, and the cashier reading the check back to a guest cannot
                  say what the difference was. The printed receipt has always
                  carried this; the screen the receipt is made from did not. */}
              {line.options && line.options.length > 0 && (
                <div className="text-xs text-ink-soft">
                  {line.options.map((o) => o.choice).join(", ")}
                </div>
              )}
              {line.comment && (
                <div className="text-xs text-ink-soft">“{line.comment}”</div>
              )}
              {line.void && (
                // Kept visible and named: a total that silently drops a line is
                // a total nobody can explain to the guest standing there.
                <div className="text-xs text-ink-muted">
                  {t.till.remove}: {line.void.reason}
                </div>
              )}
            </div>
            <div className="shrink-0 text-right">
              <div className="text-sm">
                {formatPrice(line.sum, currency, lang)}
              </div>
              {!line.void && !check.closedAt && (
                <button
                  className="mt-0.5 text-xs text-ink-muted underline"
                  disabled={busy}
                  onClick={() => void removeLine(line)}
                >
                  {t.till.remove}
                </button>
              )}
            </div>
          </li>
        ))}
      </ul>

      <footer className="shrink-0 border-t border-line bg-ink/[0.02] p-2.5">
        <div className="flex justify-between text-[13px] tabular-nums">
          <span className="text-ink-muted">{t.till.subtotal}</span>
          <span className="text-ink-soft">
            {formatPrice(check.subtotal, currency, lang)}
          </span>
        </div>
        {/* ⚠️ The largest thing on the panel, because it is the number said out
            loud to the guest. Everything above it is how it was arrived at. */}
        <div className="mt-0.5 flex items-baseline justify-between gap-2">
          <span className="till-label">{t.till.total}</span>
          <span className="font-display text-[22px] font-bold leading-none tabular-nums">
            {formatPrice(check.total, currency, lang)}
          </span>
        </div>

        {check.unfired > 0 && (
          <button
            // ⚠️ **The filled control is whichever step the check is on.**
            // Unfired lines mean the kitchen has not been told, and that is the
            // only thing worth doing next — paying for food nobody has started
            // is the mistake this button prevents. Once everything is fired the
            // button is gone and Pay takes the fill.
            //
            // It used to be charcoal, which on the dark theme is the page
            // background: the most urgent action on the screen was the least
            // visible thing on it.
            className="till-btn-primary mt-2.5 min-h-12 w-full text-base"
            disabled={busy}
            onClick={() => void run(() => api.tillFire(id))}
          >
            {t.till.fireCount.replace("{n}", String(check.unfired))}
          </button>
        )}

        {/* Payment is a cashier's button. Hidden here as a courtesy; the server
            refuses it either way — see tillDenial. */}
        {canCashier && (
          <button
            className={`mt-1.5 min-h-12 w-full text-base ${
              check.unfired > 0 ? "till-btn" : "till-btn-primary"
            }`}
            disabled={busy || live.length === 0}
            onClick={() => setPaying(true)}
          >
            {t.till.pay}
          </button>
        )}
        {/* ⚠️ No permission and no reason asked for. Moving a party from table
            four to table six takes nothing off the bill and nothing out of the
            kitchen — it corrects a fact about the room. Guarding it would put a
            manager between a waiter and the ordinary business of seating
            people, which is how permissions get switched off. */}
        <button
          className="till-btn mt-1.5 w-full"
          disabled={busy}
          onClick={() => setMoving(true)}
        >
          {t.till.moveTable}
        </button>
        {canCashier && (
          <button
            className="till-btn mt-1.5 w-full text-ink-muted"
            disabled={busy}
            onClick={() => setCancelling(true)}
          >
            {t.till.cancelCheck}
          </button>
        )}
      </footer>

      {paying && (
        <PayDialog
          check={check}
          currency={currency}
          onCancel={() => setPaying(false)}
          onPaid={() => {
            setPaying(false);
            onClosed();
          }}
          onError={onError}
        />
      )}
      {voiding && (
        <VoidDialog
          line={voiding}
          onCancel={() => setVoiding(null)}
          onConfirm={async (reason, wasted) => {
            const line = voiding;
            setVoiding(null);
            // ⚠️ The refusal here is a **request**, not a verdict: the person
            // may not write off cooked food, so the screen asks somebody who
            // may rather than stopping. See OverrideDialog.
            await tryVoid(line.lineId, reason, wasted, "");
          }}
        />
      )}
      {override && (
        <OverrideDialog
          permissionName={override.permissionName}
          busy={busy}
          error={overrideError}
          onCancel={() => {
            setOverride(null);
            setOverrideError("");
          }}
          onSubmit={(pin) =>
            void tryVoid(override.lineId, override.reason, override.wasted, pin)
          }
        />
      )}
      {moving && (
        <MoveTableDialog
          tables={tables}
          busyTables={busyTables}
          current={check.tableId ?? ""}
          onCancel={() => setMoving(false)}
          onMove={async (tableId) => {
            setMoving(false);
            await run(() => api.tillUpdateCheck(id, { tableId }));
          }}
        />
      )}
      {cancelling && (
        <VoidDialog
          title={t.till.cancelCheck}
          label={t.till.cancelReason}
          confirmLabel={t.till.confirmCancel}
          onCancel={() => setCancelling(false)}
          onConfirm={async (reason) => {
            setCancelling(false);
            setBusy(true);
            try {
              await api.tillCancel(id, reason);
              onClosed();
            } catch (err) {
              onError(err instanceof ApiError ? err.message : t.till.retry);
            } finally {
              setBusy(false);
            }
          }}
        />
      )}
    </aside>
  );
}

/** A void waiting for somebody with the permission to authorise it. */
interface PendingVoid {
  lineId: string;
  reason: string;
  wasted: boolean;
  permissionName: string;
}
