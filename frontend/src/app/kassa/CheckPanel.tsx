"use client";

import { useState } from "react";
// One icon at a time (`react-icons/lu`): the top-level entry point is an index
// of several thousand.
import {
  LuArrowRightLeft,
  LuChefHat,
  LuTrash2,
  LuWallet,
  LuX,
} from "react-icons/lu";

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
      {/* ⚠️ **Whose bill this is, at the size of a heading.** The panel is read
          from the side while the cashier is looking at the room or the menu,
          and the one thing that must never be in doubt is which table they are
          about to charge. Everything else here is how it was arrived at. */}
      <header className="shrink-0 border-b border-line px-4 py-3.5">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <h2 className="truncate text-[21px] font-bold leading-tight tracking-tight">
              {check.tableNumber
                ? `${check.tableNumber}-${t.till.table.toLowerCase()}`
                : t.till.counter}
              {check.guests ? (
                <span className="text-ink-muted"> · {check.guests}</span>
              ) : null}
            </h2>
            <p className="mt-1 truncate text-[13px] text-[rgb(var(--till-dim))]">
              {check.serverName ? `${check.serverName} · ` : ""}
              {check.openMin} {t.till.minShort}
            </p>
          </div>
          <div className="flex shrink-0 flex-col items-end gap-1.5">
            {/* The printed number, quiet: it is what a guest reads back when
                they come to ask, not something the cashier scans for. */}
            <span className="till-num text-[13px] text-[rgb(var(--till-dim))]">
              #{check.number}
            </span>
            {/* ⚠️ Coloured only once it means something. A chip on every check
                is a chip nobody reads; this one appears when the table has been
                sitting long enough that somebody should look at it. */}
            {check.openMin >= LATE_MIN && (
              <span className="till-chip till-chip-late">
                {check.openMin} {t.till.minShort}
              </span>
            )}
          </div>
        </div>
      </header>

      <ul className="min-h-0 flex-1 overflow-y-auto px-2 py-1">
        {live.length === 0 && (
          <li className="px-3 py-8 text-center text-sm text-ink-muted">
            {t.till.emptyCheck}
          </li>
        )}
        {check.lines.map((line) => (
          <li
            key={line.lineId}
            className={`flex items-start gap-2 rounded-[10px] px-1.5 py-2 ${
              line.void ? "opacity-45" : "hover:bg-ink/[0.025]"
            }`}
          >
            {/* ⚠️ **The count in its own square, before the name.** It used to
                sit under the dish as "2 × 30 000", which is where a cashier
                reading a check back to a guest has to find it by parsing a
                sentence. In a fixed column it is scanned down the list, and a
                mistyped quantity — the ordinary mistake on a till — stops being
                something you only notice in the total. */}
            <span
              className={`mt-px flex h-8 w-8 shrink-0 items-center justify-center rounded-[8px] text-[13px] font-bold tabular-nums ${
                line.fired ? "bg-ink/[0.06] text-ink-soft" : ""
              }`}
              style={
                line.fired
                  ? undefined
                  : { background: "rgb(var(--till-accent-tint))" }
              }
              // ⚠️ Fired or not is the only state on this list, and it is
              // invisible in the room: once the kitchen has a line, taking it
              // off costs food and needs a reason.
              title={line.fired ? t.till.firedLabel : t.till.pendingLabel}
            >
              {line.qty}
            </span>
            <div className="min-w-0 flex-1">
              <span
                className={`block truncate text-sm ${
                  line.void ? "line-through" : "font-semibold"
                }`}
              >
                {line.name}
              </span>
              <div className="text-[11px] text-[rgb(var(--till-dim))]">
                <span className="till-num">
                  {formatPrice(line.price, currency, lang)}
                </span>
                {!line.void && !line.fired ? ` · ${t.till.pendingLabel}` : ""}
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
                <div className="text-xs font-medium text-ink-soft">
                  “{line.comment}”
                </div>
              )}
              {line.void && (
                // Kept visible and named: a total that silently drops a line is
                // a total nobody can explain to the guest standing there.
                <div className="text-xs text-ink-muted">
                  {t.till.remove}: {line.void.reason}
                </div>
              )}
            </div>
            <div className="flex shrink-0 items-center gap-1">
              {/* ⚠️ **Only before the kitchen has it.** After that the paper at
                  the pass carries the old number, and a quantity that changes
                  silently leaves the screen and the kitchen disagreeing about
                  the same dish — with the guest finding out. A fired line keeps
                  the remove button, which asks for a reason. */}
              {!line.void && !line.fired && !check.closedAt && (
                <span className="mr-1 flex items-center gap-1">
                  <button
                    className="till-btn h-9 w-9 shrink-0 px-0 text-base"
                    disabled={busy || line.qty <= 1}
                    aria-label={`${line.name} −`}
                    onClick={() =>
                      void run(() =>
                        api.tillLineQty(id, line.lineId, line.qty - 1),
                      )
                    }
                  >
                    −
                  </button>
                  <button
                    className="till-btn h-9 w-9 shrink-0 px-0 text-base"
                    disabled={busy}
                    aria-label={`${line.name} +`}
                    onClick={() =>
                      void run(() =>
                        api.tillLineQty(id, line.lineId, line.qty + 1),
                      )
                    }
                  >
                    +
                  </button>
                </span>
              )}
              <span className="till-num w-[5.5rem] text-right text-[15px] font-semibold">
                {formatPrice(line.sum, currency, lang)}
              </span>
              {/* ⚠️ **A target, not an underlined word.** "O'chirish" was a
                  12px text link under the price: on a touchscreen that is a
                  miss, and the miss lands on the price of the line above. An
                  icon button is the same 44px square as everything else here,
                  and it is the only red thing on the panel. */}
              {!line.void && !check.closedAt && (
                <button
                  className="till-btn-danger h-9 w-9 shrink-0 px-0"
                  disabled={busy}
                  aria-label={`${t.till.remove}: ${line.name}`}
                  title={t.till.remove}
                  onClick={() => void removeLine(line)}
                >
                  <LuTrash2 className="h-4 w-4" aria-hidden />
                </button>
              )}
            </div>
          </li>
        ))}
      </ul>

      <footer className="till-sunken shrink-0 border-t border-line p-2.5">
        {/* ⚠️ **The money sits in its own block, not in the button stack.** The
            footer used to run subtotal, total and four full-width buttons down
            one column, which made the number said out loud to the guest look
            like the first row of a menu of actions. */}
        <div className="rounded-[10px] border border-line bg-surface px-3 py-2">
          <div className="flex justify-between text-[14px]">
            <span className="text-[rgb(var(--till-mid))]">
              {t.till.subtotal}
            </span>
            <span className="till-num text-ink-soft">
              {formatPrice(check.subtotal, currency, lang)}
            </span>
          </div>
          {/* ⚠️ The largest thing on the panel, because it is the number said
              out loud to the guest. Everything above it is how it was arrived
              at. */}
          <div className="mt-2 flex items-baseline justify-between gap-2 border-t border-dashed pt-2"
            style={{ borderColor: "var(--line-strong)" }}
          >
            <span className="text-[17px] font-bold">{t.till.total}</span>
            <span className="till-total">
              {formatPrice(check.total, currency, lang)}
            </span>
          </div>
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
            className="till-btn-primary mt-2.5 min-h-[3.25rem] w-full text-base"
            disabled={busy}
            onClick={() => void run(() => api.tillFire(id))}
          >
            <LuChefHat className="h-[1.1rem] w-[1.1rem]" aria-hidden />
            {t.till.fireCount.replace("{n}", String(check.unfired))}
          </button>
        )}

        {/* Payment is a cashier's button. Hidden here as a courtesy; the server
            refuses it either way — see tillDenial. */}
        {canCashier && (
          <button
            className={`mt-1.5 min-h-[3.25rem] w-full text-base ${
              check.unfired > 0 ? "till-btn" : "till-btn-primary"
            }`}
            disabled={busy || live.length === 0}
            onClick={() => setPaying(true)}
          >
            <LuWallet className="h-[1.1rem] w-[1.1rem]" aria-hidden />
            {t.till.pay}
          </button>
        )}

        {/* ---- The rest ----

            ⚠️ **Quiet, side by side, and smaller.** These were two more
            full-width buttons under the two that matter, so the footer was four
            identical bars and the shape of the screen no longer said which one
            the check was waiting for. Moving a table and cancelling a check are
            real actions and rare ones; they get a row, not a rank.

            ⚠️ Moving asks no permission and no reason: it takes nothing off the
            bill and nothing out of the kitchen — it corrects a fact about the
            room. Guarding it would put a manager between a waiter and the
            ordinary business of seating people, which is how permissions get
            switched off altogether. */}
        <div className="mt-1.5 flex gap-1.5">
          <button
            className="till-btn-ghost flex-1 text-[13px]"
            disabled={busy}
            onClick={() => setMoving(true)}
          >
            <LuArrowRightLeft className="h-4 w-4" aria-hidden />
            {t.till.moveTable}
          </button>
          {canCashier && (
            <button
              className="till-btn-danger flex-1 text-[13px]"
              disabled={busy}
              onClick={() => setCancelling(true)}
            >
              <LuX className="h-4 w-4" aria-hidden />
              {t.till.cancelCheck}
            </button>
          )}
        </div>
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

/** When a table has been sitting long enough to be worth a colour. Matches the
 *  floor tile's threshold — one fact, one number, wherever you are standing. */
const LATE_MIN = 45;

/** A void waiting for somebody with the permission to authorise it. */
interface PendingVoid {
  lineId: string;
  reason: string;
  wasted: boolean;
  permissionName: string;
}
