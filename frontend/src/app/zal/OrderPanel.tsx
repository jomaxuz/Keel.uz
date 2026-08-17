"use client";

import { useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
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
  onChange,
  onBack,
  onError,
}: {
  check: Check;
  currency: string;
  tables: FloorTable[];
  busyTables: string[];
  onChange: (next: Check) => void;
  onBack: () => void;
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
      <div className="min-h-0 flex-1 overflow-y-auto px-3">
        {live.length === 0 && (
          <p className="py-8 text-center text-sm text-ink-muted">
            {t.till.emptyCheck}
          </p>
        )}
        <ul className="divide-y divide-line">
          {live.map((l) => (
            <li key={l.lineId} className="py-3">
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <div className="font-medium">
                    {l.qty} × {l.name}
                  </div>
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
                    <div className="text-sm text-brand">* {l.comment}</div>
                  )}
                  {/* Fired or not is the only colour distinction: it is the
                      moment of no return, and it is invisible in the room. */}
                  <div className="text-xs text-ink-muted">
                    {l.fired ? t.till.firedLabel : t.till.pendingLabel}
                  </div>
                </div>
                <div className="flex shrink-0 items-center gap-1">
                  <span className="text-sm">
                    {formatPrice(l.sum, currency, lang)}
                  </span>
                  {/* ⚠️ Hidden once the line is with the kitchen, because the
                      server refuses it there: the printed ticket cannot be
                      edited. A button that always answers "no" is a button
                      people learn to distrust. */}
                  {!l.fired && (
                    <button
                      className="btn-ghost px-2 text-sm"
                      disabled={busy}
                      onClick={() => setCommenting(l)}
                    >
                      ✎
                    </button>
                  )}
                  <button
                    className="btn-ghost px-2 text-sm text-danger"
                    disabled={busy}
                    onClick={() => void remove(l)}
                  >
                    ✕
                  </button>
                </div>
              </div>
            </li>
          ))}
        </ul>
      </div>

      <footer className="shrink-0 border-t border-line p-3">
        <div className="flex items-baseline justify-between">
          <span className="text-sm text-ink-muted">{t.till.total}</span>
          <span className="font-display text-xl font-bold">
            {formatPrice(check.total, currency, lang)}
          </span>
        </div>

        {/* ⚠️ The one control that changes colour, and only when there is
            something to send. A waiter who types an order and walks away
            without firing it is this screen's only real failure, and the guest
            finds out twenty minutes later. */}
        <button
          className={`mt-2.5 min-h-14 w-full rounded-[12px] text-base font-bold ${
            check.unfired > 0
              ? "bg-brand text-white"
              : "border border-line text-ink-muted"
          }`}
          disabled={busy || check.unfired === 0}
          onClick={() => void run(() => api.tillFire(check.id))}
        >
          {check.unfired > 0
            ? t.till.fireCount.replace("{n}", String(check.unfired))
            : t.till.allFired}
        </button>

        <div className="mt-2 flex gap-2">
          <button className="btn flex-1" disabled={busy} onClick={onBack}>
            {t.till.tables}
          </button>
          <button
            className="btn flex-1"
            disabled={busy}
            onClick={() => setMoving(true)}
          >
            {t.till.moveTable}
          </button>
        </div>

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
      <div className="w-full max-w-sm rounded-[14px] border border-line bg-surface p-4 shadow-card">
        <h2 className="font-display text-lg font-bold">
          {t.till.commentTitle}
        </h2>
        <p className="mt-1 text-sm text-ink-soft">{line.name}</p>
        <input
          className="input mt-3 h-12"
          autoFocus
          placeholder={t.till.commentPh}
          value={text}
          onChange={(e) => setText(e.target.value)}
        />
        <div className="mt-4 flex gap-2">
          <button className="btn flex-1" onClick={onCancel}>
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
