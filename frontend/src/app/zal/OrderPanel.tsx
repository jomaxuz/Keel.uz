"use client";

import { useState } from "react";
// One icon at a time (`react-icons/lu`): the top-level entry point is an index
// of several thousand.
import {
  LuArrowRightLeft,
  LuChefHat,
  LuLayoutGrid,
  LuPencil,
  LuPlus,
  LuReceipt,
  LuTrash2,
} from "react-icons/lu";

import { api, ApiError } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { printReceipt } from "@/lib/print";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import GuestTabs from "@/components/till/GuestTabs";
import MoveTableDialog from "@/components/till/MoveTableDialog";
import VoidDialog from "@/components/till/VoidDialog";
import OverrideDialog from "@/components/till/OverrideDialog";
import type { Check, CheckLine, FloorTable } from "@/lib/types";

/**
 * The waiter's check.
 *
 * ⚠️ **Its own component, not the till's with the payment hidden.** The two
 * screens answer different questions and are held in different hands: the till
 * is a monoblock on a counter ending a sale, this is a tablet carried between
 * tables starting one. Sharing a component would mean every change to the till
 * had to be checked against a screen nobody was thinking about — and it is the
 * waiter's screen that will move to a phone (see docs/pos-reja.md §3).
 *
 * ⚠️ **The unfired count is the whole point of the layout.** A waiter's only
 * real failure mode is typing an order and walking away without sending it, and
 * the guest finds out about that twenty minutes later. So "send to the kitchen"
 * is the largest thing on the screen whenever there is anything to send, and it
 * is the only control that changes colour.
 */
export default function OrderPanel({
  check,
  currency,
  tables,
  busyTables,
  guest,
  onGuest,
  onChange,
  onBack,
  onAddDish,
  onError,
}: {
  check: Check;
  currency: string;
  tables: FloorTable[];
  busyTables: string[];
  onChange: (next: Check) => void;
  onBack: () => void;
  /** Which guest the next dish is for — the tab is where it goes. */
  guest: number;
  onGuest: (guest: number) => void;
  /** Open the menu to put something else on this table. */
  onAddDish: () => void;
  onError: (msg: string) => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [busy, setBusy] = useState(false);
  const [voiding, setVoiding] = useState<CheckLine | null>(null);
  const [moving, setMoving] = useState(false);
  const [override, setOverride] = useState<Pending | null>(null);
  const [overrideError, setOverrideError] = useState("");
  const [commenting, setCommenting] = useState<CheckLine | null>(null);

  const live = check.lines.filter((l) => !l.void);
  // The table tab shows the whole bill; a guest tab shows one person's share.
  const shown =
    guest === 0 ? live : live.filter((l) => (l.guest ?? 0) === guest);

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

  /** Take a line off, asking a manager if this person may not.
   *
   *  ⚠️ An unfired line goes without a dialog at all — it is a typo being
   *  corrected, and asking a waiter to justify their own mistyping is how the
   *  reasons on the real voids turn into ".". */
  /** Hand the table its bill, and record that they asked for it. */
  async function printBill() {
    setBusy(true);
    try {
      const res = await api.tillPrint(check.id, "precheck");
      onChange(res.check);
      // The browser prints only when no printer of the branch's own took it.
      if (res.queued === 0) printReceipt(res.lines, res.widthMM, res.logoUrl);
    } catch (err) {
      onError(err instanceof ApiError ? err.message : t.till.retry);
    } finally {
      setBusy(false);
    }
  }

  async function remove(line: CheckLine) {
    if (!line.fired) {
      await run(() => api.tillVoidLine(check.id, line.lineId));
      return;
    }
    setVoiding(line);
  }

  async function tryVoid(
    lineId: string,
    reason: string,
    wasted: boolean,
    pin: string,
  ) {
    setBusy(true);
    setOverrideError("");
    try {
      onChange(
        await api.tillVoidLine(check.id, lineId, { reason, wasted, pin }),
      );
      setOverride(null);
    } catch (err) {
      const perm = err instanceof ApiError ? err.needsOverride : null;
      if (perm) {
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

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <GuestTabs
        lines={check.lines}
        guests={check.guests ?? 0}
        value={guest}
        onPick={onGuest}
        onAdd={() =>
          onGuest(
            Math.max(
              check.lines.reduce((m, l) => Math.max(m, l.guest ?? 0), 0),
              check.guests ?? 0,
            ) + 1,
          )
        }
      />

      <div className="min-h-0 flex-1 overflow-y-auto px-3">
        {live.length === 0 && (
          <p className="py-8 text-center text-sm text-ink-muted">
            {t.till.emptyCheck}
          </p>
        )}
        <ul className="py-1">
          {shown.map((l) => (
            <li key={l.lineId} className="rounded-[10px] px-1.5 py-2.5">
              <div className="flex items-start justify-between gap-2">
                <div className="flex min-w-0 gap-2.5">
                  {/* ⚠️ The count in its own square, the same as the till's:
                      read down a column rather than parsed out of a sentence,
                      and tinted while the kitchen has not seen it. */}
                  <span
                    className={`mt-px flex h-8 w-8 shrink-0 items-center justify-center rounded-[8px] text-sm font-bold tabular-nums ${
                      l.fired ? "bg-ink/[0.06] text-ink-soft" : "text-ink"
                    }`}
                    style={
                      l.fired
                        ? undefined
                        : { background: "rgb(var(--till-accent-tint))" }
                    }
                  >
                    {l.qty}
                  </span>
                  <div className="min-w-0">
                  <div className="font-semibold">{l.name}</div>
                  {/* ⚠️ The comment is the reason a waiter uses this screen
                      rather than shouting across the room, so it is shown on
                      the line rather than behind a tap. */}
                  {/* ⚠️ The chosen option, on the line. Two rows of the same
                      dish at two prices are otherwise unexplainable — to the
                      waiter reading the check back, and to the guest. */}
                  {l.options && l.options.length > 0 && (
                    <div className="text-sm text-ink-soft">
                      {l.options.map((o) => o.choice).join(", ")}
                    </div>
                  )}
                  {l.comment && (
                    <div className="text-sm font-medium text-ink-soft">
                      “{l.comment}”
                    </div>
                  )}
                  {/* Fired or not is the only state on this list: it is the
                      moment of no return, and it is invisible in the room. */}
                  {/* ⚠️ The state in colour, because it is the only one on this
                      list and it is invisible in the room: blue means the
                      kitchen has it, amber means it is still a draft on this
                      tablet — and a waiter who walks away from a draft is this
                      screen's one real failure. */}
                  <div className="flex items-center gap-2">
                    <span
                      className="text-xs font-semibold"
                      style={{
                        color: l.fired
                          ? "rgb(var(--till-info))"
                          : "rgb(var(--till-accent-ink))",
                      }}
                    >
                      {l.fired ? t.till.firedLabel : t.till.pendingLabel}
                    </span>
                    {(l.course ?? 0) > 0 && (
                      <span className="till-chip till-chip-info">
                        {"I".repeat(l.course ?? 0)}
                      </span>
                    )}
                  </div>
                  </div>
                </div>
                <div className="flex shrink-0 items-center gap-1">
                  <span className="till-num w-[5.5rem] text-right text-[15px] font-semibold">
                    {formatPrice(l.sum, currency, lang)}
                  </span>
                  {/* ⚠️ Hidden once the line is with the kitchen, because the
                      server refuses it there: the printed ticket cannot be
                      edited. A button that always answers "no" is a button
                      people learn to distrust. */}
                  {/* ⚠️ Only before the kitchen has it: after that the paper at
                      the pass carries the old number, and a silent change
                      leaves the screen and the kitchen disagreeing about the
                      same dish. */}
                  {!l.fired && (
                    <>
                      <button
                        className="till-btn h-10 w-10 shrink-0 px-0 text-base"
                        disabled={busy || l.qty <= 1}
                        aria-label={`${l.name} −`}
                        onClick={() =>
                          void run(() =>
                            api.tillLineQty(check.id, l.lineId, l.qty - 1),
                          )
                        }
                      >
                        −
                      </button>
                      <button
                        className="till-btn h-10 w-10 shrink-0 px-0 text-base"
                        disabled={busy}
                        aria-label={`${l.name} +`}
                        onClick={() =>
                          void run(() =>
                            api.tillLineQty(check.id, l.lineId, l.qty + 1),
                          )
                        }
                      >
                        +
                      </button>
                    </>
                  )}
                  {!l.fired && (
                    <button
                      className="till-btn-ghost h-10 w-10 shrink-0 px-0"
                      disabled={busy}
                      aria-label={t.till.commentTitle}
                      title={t.till.commentTitle}
                      onClick={() => setCommenting(l)}
                    >
                      <LuPencil className="h-4 w-4" aria-hidden />
                    </button>
                  )}
                  <button
                    className="till-btn-danger h-10 w-10 shrink-0 px-0"
                    disabled={busy}
                    aria-label={`${t.till.remove}: ${l.name}`}
                    title={t.till.remove}
                    onClick={() => void remove(l)}
                  >
                    <LuTrash2 className="h-4 w-4" aria-hidden />
                  </button>
                </div>
              </div>
            </li>
          ))}
        </ul>
      </div>

      <footer className="till-sunken shrink-0 border-t border-line p-3">
        <div className="flex items-baseline justify-between rounded-[10px] border border-line bg-surface px-3 py-2">
          <span className="till-label">{t.till.total}</span>
          <span className="till-total">
            {formatPrice(check.total, currency, lang)}
          </span>
        </div>

        {/* ⚠️ The one control that changes colour, and only when there is
            something to send. A waiter who types an order and walks away
            without firing it is this screen's only real failure, and the guest
            finds out twenty minutes later. */}
        {/* ⚠️ **Two quiet actions, then one that is not.** The design puts the
            accent on the bottom bar and the ordinary work above it, and the
            accent is whichever step this table is on: anything untold to the
            kitchen and that is the only thing worth pressing — a waiter who
            types an order and walks away without sending it is this screen's
            one real failure, and the guest finds out twenty minutes later.
            Once everything is away, the next thing a table wants is more. */}
        <div className="mt-2.5 grid grid-cols-3 gap-2">
          <button
            className="till-btn-quiet"
            disabled={busy}
            onClick={onBack}
          >
            <LuLayoutGrid className="h-4 w-4" aria-hidden />
            {t.till.tables}
          </button>
          <button
            className="till-btn-quiet"
            disabled={busy}
            onClick={() => setMoving(true)}
          >
            <LuArrowRightLeft className="h-4 w-4" aria-hidden />
            {t.till.moveTable}
          </button>
          {/* ⚠️ The bill belongs on the waiter's screen more than anywhere: the
              guest asks *them*, at the table, and a waiter who has to walk to
              the counter to have it printed is the reason tables wait. */}
          <button
            className="till-btn-quiet"
            disabled={busy}
            onClick={() => void printBill()}
          >
            <LuReceipt className="h-4 w-4" aria-hidden />
            {t.till.precheck}
          </button>
        </div>

        <button
          className="till-btn-accent mt-2 min-h-14 w-full text-[17px]"
          disabled={busy}
          onClick={
            check.unfired > 0
              ? () => void run(() => api.tillFire(check.id))
              : onAddDish
          }
        >
          {check.unfired > 0 ? (
            <>
              <LuChefHat className="h-[1.15rem] w-[1.15rem]" aria-hidden />
              {t.till.fireCount.replace("{n}", String(check.unfired))}
            </>
          ) : (
            <>
              <LuPlus className="h-[1.15rem] w-[1.15rem]" aria-hidden />
              {t.till.addDish}
            </>
          )}
        </button>

        {/* ⚠️ No payment button, and not because it is hidden: a waiter carrying
            a tablet has no drawer, no printer and no bank terminal. Offering it
            here would be offering something that ends at the counter anyway,
            and a control that always sends you somewhere else is a control that
            teaches people the screen is wrong. */}
        <p className="mt-2 text-center text-xs text-ink-muted">
          {t.till.payAtTill}
        </p>
      </footer>

      {commenting && (
        <CommentDialog
          line={commenting}
          onCancel={() => setCommenting(null)}
          onSave={async (comment) => {
            const line = commenting;
            setCommenting(null);
            await run(() =>
              api.tillCommentLine(check.id, line.lineId, comment),
            );
          }}
        />
      )}
      {voiding && (
        <VoidDialog
          line={voiding}
          onCancel={() => setVoiding(null)}
          onConfirm={async (reason, wasted) => {
            const line = voiding;
            setVoiding(null);
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
            await run(() => api.tillUpdateCheck(check.id, { tableId }));
          }}
        />
      )}
    </div>
  );
}

/** Writing "no onion" against a dish. */
function CommentDialog({
  line,
  onCancel,
  onSave,
}: {
  line: CheckLine;
  onCancel: () => void;
  onSave: (comment: string) => void | Promise<void>;
}) {
  const t = useAdminT();
  const [text, setText] = useState(line.comment ?? "");
  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-ink/40 p-4 sm:items-center">
      <div className="till-dialog w-full max-w-sm p-4">
        <h2 className="font-display text-lg font-bold">
          {t.till.commentTitle}
        </h2>
        <p className="mt-1 text-sm text-ink-soft">{line.name}</p>
        <input
          className="till-input mt-3 h-12"
          autoFocus
          placeholder={t.till.commentPh}
          value={text}
          onChange={(e) => setText(e.target.value)}
        />
        <div className="mt-4 flex gap-2">
          <button className="till-btn flex-1" onClick={onCancel}>
            {t.common.cancel}
          </button>
          <button
            className="till-btn-primary flex-1"
            onClick={() => void onSave(text.trim())}
          >
            {t.common.save}
          </button>
        </div>
      </div>
    </div>
  );
}

interface Pending {
  lineId: string;
  reason: string;
  wasted: boolean;
  permissionName: string;
}
