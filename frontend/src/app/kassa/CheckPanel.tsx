"use client";

import { useState } from "react";

import CommentDialog from "@/components/till/CommentDialog";
// One icon at a time (`react-icons/lu`): the top-level entry point is an index
// of several thousand.
import {
  LuChefHat,
  LuMessageSquare,
  LuTrash2,
  LuWallet,
} from "react-icons/lu";

import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { formatPrice } from "@/lib/format";
import type {
  Check,
  CheckLine,
  FloorTable,
  TillPaymentMethod,
} from "@/lib/types";

import PayDialog from "./PayDialog";
import GuestTabs from "@/components/till/GuestTabs";
import { isLocal } from "@/lib/offline/checks";
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
  guest,
  onGuest,
  moving,
  onMoving,
  cancelling,
  onCancelling,
  onChange,
  onClosed,
  onError,
  onOffline,
  onSeen,
  onLocalFire,
  onLocalQty,
  onLocalRemove,
}: {
  check: Check | null;
  currency: string;
  canCashier: boolean;
  /** The room, for moving a party. */
  tables: FloorTable[];
  /** Tables that already have a check on them. */
  busyTables: string[];
  /** Which guest the next dish is for. ⚠️ Owned by the page because the menu
   *  needs it too — the tab is where the dish goes, not a filter on a list. */
  guest: number;
  onGuest: (guest: number) => void;
  /** ⚠️ Opened from the bottom bar, which this panel does not own — but the
   *  dialogs stay here, with the code that knows what to do when they close. */
  moving: boolean;
  onMoving: (open: boolean) => void;
  cancelling: boolean;
  onCancelling: (open: boolean) => void;
  onChange: (next: Check) => void;
  onClosed: () => void;
  onError: (msg: string) => void;
  /** The sale went through here but not to the server — see lib/offline. */
  onOffline: (msg: string) => void;
  /** Whether the last request reached the server. */
  onSeen: (ok: boolean) => void;
  /** The three edits a check this device owns can make on its own. */
  onLocalFire: () => Promise<void>;
  onLocalQty: (lineId: string, qty: number) => Promise<void>;
  onLocalRemove: (lineId: string) => Promise<void>;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [busy, setBusy] = useState(false);
  const [paying, setPaying] = useState(false);
  const [method, setMethod] = useState<TillPaymentMethod>("cash");
  const [voiding, setVoiding] = useState<CheckLine | null>(null);
  const [commenting, setCommenting] = useState<CheckLine | null>(null);
  // The void the server asked a manager to authorise, held so the retry sends
  // the same reason rather than asking the waiter to type it twice.
  const [override, setOverride] = useState<PendingVoid | null>(null);
  const [overrideError, setOverrideError] = useState("");

  if (!check) {
    return (
      <div className="flex w-full flex-col">
        <header className="shrink-0 border-b border-line px-4 py-3.5">
          <h2 className="text-[21px] font-bold leading-tight tracking-tight text-[rgb(var(--till-dim))]">
            {t.till.check}
          </h2>
        </header>
        <div className="flex flex-1 items-center justify-center p-8">
          {/* The empty state names the next move rather than the state: a
              cashier who has just unlocked the screen is looking for what to
              press, not for a description of nothing. */}
          <p className="max-w-[14rem] text-center text-[15px] leading-relaxed text-[rgb(var(--till-dim))]">
            {t.till.emptyCheck}
          </p>
        </div>
      </div>
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
  // ⚠️ **A check this device owns behaves differently, and says so.** The
  // kitchen screen cannot see it, nothing can be printed for it, and the two
  // actions that need the server's judgement — moving a table, cancelling with
  // a reason on the record — are not offered rather than offered and refused.
  const offline = isLocal(check);
  // ⚠️ The whole-table tab shows everything, including what is already assigned
  // to a guest: it is the bill the restaurant is owed, and a screen where the
  // total and the visible lines disagree is a screen nobody can read back.
  const shownLines =
    guest === 0
      ? check.lines
      : check.lines.filter((l) => (l.guest ?? 0) === guest);
  // Courses that still have something to send, in order.
  const waitingCourses = [
    ...new Set(
      live.filter((l) => !l.fired).map((l) => l.course ?? 0),
    ),
  ].sort((a, b) => a - b);

  return (
    <div className="flex min-h-0 w-full flex-col overflow-hidden">
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

      <GuestTabs
        lines={check.lines}
        guests={check.guests ?? 0}
        value={guest}
        onPick={onGuest}
        onAdd={() => onGuest(nextGuest(check))}
      />

      <ul className="min-h-0 flex-1 overflow-y-auto px-2 py-1">
        {live.length === 0 && (
          <li className="px-3 py-8 text-center text-sm text-ink-muted">
            {t.till.emptyCheck}
          </li>
        )}
        {shownLines.map((line, i) => (
          <li
            key={line.lineId}
            // ⚠️ **Two rows, not one.** Everything here used to sit on a single
            // line: the count, the name, the stepper, the sum and two icon
            // buttons. The fixed items alone are wider than the panel on a
            // 1024px monoblock — which is the hardware this runs on — so the
            // name column, the only flexible one, was squeezed to nothing. The
            // dish disappeared entirely and its unit price printed over the
            // "−" button. Measured on the machine, not guessed.
            //
            // So the name gets the width it needs and the controls get their
            // own row beneath. Nothing is smaller: the buttons are still the
            // same 44px targets, they are simply not competing with a dish
            // called "Bahor salati (katta)" for the same 320 pixels.
            className={`flex flex-col gap-1 rounded-[10px] px-1.5 py-2 ${
              line.void ? "opacity-45" : "hover:bg-ink/[0.025]"
            }`}
          >
            {/* ⚠️ The course is written once, above the first dish in it,
                rather than on every line: a badge on all six rows of a course
                is six repetitions of one fact, and the eye stops reading it. */}
            {(line.course ?? 0) > 0 &&
              (line.course ?? 0) !== (shownLines[i - 1]?.course ?? 0) && (
                <span className="till-chip till-chip-info absolute -ml-1 -mt-4">
                  {"I".repeat(line.course ?? 0)}
                </span>
              )}
            {/* ⚠️ **The count in its own square, before the name.** It used to
                sit under the dish as "2 × 30 000", which is where a cashier
                reading a check back to a guest has to find it by parsing a
                sentence. In a fixed column it is scanned down the list, and a
                mistyped quantity — the ordinary mistake on a till — stops being
                something you only notice in the total. */}
            <div className="flex items-start gap-2">
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
            </div>
            {/* ⚠️ Right-aligned under the name, so the sum still lines up down
                the list — that column is read vertically when a cashier checks
                a total against the paper, and left-aligning it would break the
                one reason to look at it. */}
            <div className="flex items-center justify-end gap-1">
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
                      offline
                        ? void onLocalQty(line.lineId, line.qty - 1)
                        : void run(() =>
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
                      offline
                        ? void onLocalQty(line.lineId, line.qty + 1)
                        : void run(() =>
                            api.tillLineQty(id, line.lineId, line.qty + 1),
                          )
                    }
                  >
                    +
                  </button>
                </span>
              )}
              {/* ⚠️ `min-w`, not `w`, and never wrapped. Fixed at 5.5rem a
                  six-figure sum broke across two lines — "84 000" over "so'm" —
                  which on a list of six dishes is six rows of different
                  heights and a column that can no longer be read downwards. */}
              <span className="till-num ml-auto min-w-[5.5rem] whitespace-nowrap text-right text-[15px] font-semibold">
                {formatPrice(line.sum, currency, lang)}
              </span>
              {/* ⚠️ **A target, not an underlined word.** "O'chirish" was a
                  12px text link under the price: on a touchscreen that is a
                  miss, and the miss lands on the price of the line above. An
                  icon button is the same 44px square as everything else here,
                  and it is the only red thing on the panel. */}
              {/* ⚠️ **The note the kitchen needs, written where the order is
                  taken.** The floor screen had this and the till did not: a
                  cashier taking a counter order for somebody saying "no
                  onions" had to remember it and say it out loud at the pass —
                  which is exactly the instruction that gets lost on a busy
                  evening. Offline it is not offered: a local check has no
                  server line to attach it to yet. */}
              {!line.void && !check.closedAt && !offline && (
                <button
                  className="till-btn h-9 w-9 shrink-0 px-0"
                  disabled={busy}
                  aria-label={`${t.till.commentTitle}: ${line.name}`}
                  title={t.till.commentTitle}
                  onClick={() => setCommenting(line)}
                >
                  <LuMessageSquare className="h-4 w-4" aria-hidden />
                </button>
              )}
              {!line.void && !check.closedAt && (
                <button
                  className="till-btn-danger h-9 w-9 shrink-0 px-0"
                  disabled={busy}
                  aria-label={`${t.till.remove}: ${line.name}`}
                  title={t.till.remove}
                  onClick={() =>
                    offline
                      ? void onLocalRemove(line.lineId)
                      : void removeLine(line)
                  }
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
          {/* ⚠️ **On the screen before it is on the bill.** The cashier is
              about to say a number out loud; a service charge that appears
              only on the printed receipt turns that into a correction at the
              door. Named with its rate for the same reason the paper names
              it. */}
          {check.service ? (
            <div className="mt-1 flex justify-between text-[14px]">
              <span className="text-[rgb(var(--till-mid))]">
                {t.till.service}
                {check.servicePercent ? ` ${check.servicePercent}%` : ""}
              </span>
              <span className="till-num text-ink-soft">
                {formatPrice(check.service, currency, lang)}
              </span>
            </div>
          ) : null}
          {/* ⚠️ The largest thing on the panel, because it is the number said
              out loud to the guest. Everything above it is how it was arrived
              at. */}
          {/* ⚠️ **Stacked, because this number may not wrap.** Side by side it
              did: "172 000 so'm" at 30px does not fit beside its label in a
              320px column, and it broke after "172 000" — leaving the largest
              type on the screen showing a number that was not the total. This
              is the figure a cashier reads out loud to a guest, so it gets a
              line of its own and stays whole. */}
          <div
            className="mt-2 border-t border-dashed pt-2"
            style={{ borderColor: "var(--line-strong)" }}
          >
            <span className="block text-[15px] font-bold">{t.till.total}</span>
            <span className="till-total mt-0.5 block whitespace-nowrap text-right">
              {formatPrice(check.total, currency, lang)}
            </span>
          </div>
        </div>

        {/* ---- What this check is waiting for ----

            ⚠️ **One accent control, and it moves with the check.** Unfired
            lines mean the kitchen has not been told, and paying for food nobody
            has started is the mistake this ordering prevents. Once everything
            is away, the accent is the money. */}
        {offline && (
          <p className="mb-2 rounded-[10px] bg-[rgb(var(--till-accent-tint))] px-2.5 py-2 text-[12px] leading-snug text-[rgb(var(--till-accent-ink))]">
            {t.till.offlineKitchen}
          </p>
        )}
        {check.unfired > 0 ? (
          <button
            className="till-btn-accent mt-2.5 min-h-[3.5rem] w-full text-[17px]"
            disabled={busy}
            onClick={() =>
              offline
                ? void onLocalFire()
                : void run(() => api.tillFire(id))
            }
          >
            <LuChefHat className="h-[1.15rem] w-[1.15rem]" aria-hidden />
            {t.till.fireCount.replace("{n}", String(check.unfired))}
          </button>
        ) : null}

        {/* ⚠️ **One button per waiting course, and only when a check has
            them.** Sending the whole check is the ordinary case and keeps the
            big button; courses are the exception, and the exception is exactly
            what must not be sent by accident — starters and mains arriving
            together is the failure the feature exists to prevent. */}
        {waitingCourses.length > 1 && !offline && (
          <div className="mt-1.5 flex flex-wrap gap-1.5">
            {waitingCourses.map((c) => (
              <button
                key={c}
                className="till-btn-quiet flex-1 text-[13px]"
                disabled={busy}
                onClick={() => void run(() => api.tillFire(id, c))}
              >
                {t.till.fireCourse(c)}
              </button>
            ))}
          </div>
        )}

        {/* Payment is a cashier's button. Hidden here as a courtesy; the server
            refuses it either way — see tillDenial. */}
        {canCashier && (
          <>
            {/* ⚠️ The method is chosen **before** the dialog, on the panel the
                cashier is already looking at. The guest says "karta" while the
                check is still being read back, and a dialog that opens on cash
                every time is one more tap on the busiest screen there is. */}
            <div className="mt-2.5 grid grid-cols-3 gap-2">
              {METHODS.map((m) => (
                <button
                  key={m.id}
                  onClick={() => setMethod(m.id)}
                  disabled={busy || live.length === 0}
                  title={t.till[m.full]}
                  className={`min-h-11 truncate rounded-[11px] border px-1 text-[13px] font-semibold transition disabled:opacity-40 ${
                    method === m.id
                      ? "border-[rgb(var(--till-accent))] bg-[rgb(var(--till-accent-tint))] text-[rgb(var(--till-accent-ink))]"
                      : "border-line bg-surface text-ink-soft hover:border-line-strong"
                  }`}
                >
                  {t.till[m.label]}
                </button>
              ))}
            </div>
            <button
              className={`mt-2 min-h-[3.5rem] w-full text-[17px] ${
                check.unfired > 0 ? "till-btn" : "till-btn-accent"
              }`}
              disabled={busy || live.length === 0}
              onClick={() => setPaying(true)}
            >
              <LuWallet className="h-[1.15rem] w-[1.15rem]" aria-hidden />
              {/* ⚠️ "Pay", not "confirm payment" — the dialog behind it is the
                  one that confirms, and two buttons with one name on the same
                  screen is a control nobody can be told to press over the
                  phone. */}
              {t.till.pay}
            </button>
            {/* ⚠️ **Why, not just "no".** A greyed button with nothing beside it
                is read as a broken screen — this one was reported missing while
                it was on screen the whole time, waiting for a dish. */}
            {live.length === 0 && (
              <p className="mt-1.5 text-center text-[12px] text-[rgb(var(--till-dim))]">
                {t.till.payNeedsLines}
              </p>
            )}
          </>
        )}
        {/* ⚠️ Said rather than left blank, for the same reason the floor screen
            names an empty room: a waiter looking for the payment button needs
            to know it is somebody else's, not that the till is broken. */}
        {!canCashier && (
          <p className="mt-2.5 text-center text-[12px] text-[rgb(var(--till-dim))]">
            {t.till.payNeedsCashier}
          </p>
        )}
      </footer>

      {paying && (
        <PayDialog
          check={check}
          currency={currency}
          initialMethod={method}
          onOffline={onOffline}
          onSeen={onSeen}
          onCancel={() => setPaying(false)}
          onPaid={() => {
            setPaying(false);
            onClosed();
          }}
          onError={onError}
        />
      )}
      {commenting && (
        <CommentDialog
          line={commenting}
          onCancel={() => setCommenting(null)}
          onSave={async (comment) => {
            const line = commenting;
            setCommenting(null);
            await run(() => api.tillCommentLine(id, line.lineId, comment));
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
          onCancel={() => onMoving(false)}
          onMove={async (tableId) => {
            onMoving(false);
            await run(() => api.tillUpdateCheck(id, { tableId }));
          }}
        />
      )}
      {cancelling && (
        <VoidDialog
          title={t.till.cancelCheck}
          label={t.till.cancelReason}
          confirmLabel={t.till.confirmCancel}
          onCancel={() => onCancelling(false)}
          onConfirm={async (reason) => {
            onCancelling(false);
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
    </div>
  );
}

/** The next free guest number on a check. */
function nextGuest(check: Check): number {
  const highest = check.lines.reduce((max, l) => Math.max(max, l.guest ?? 0), 0);
  return Math.max(highest, check.guests ?? 0) + 1;
}

/** The three ways a guest pays at the counter. Named here rather than inside
 *  the dialog because the panel now asks first. */
// ⚠️ **The short card label.** "Karta (terminal)" is right in the payment
// dialog, where there is room to say which card machine; in a 300px column
// shared by three chips it wrapped to two lines and spilled out of its own
// button. The full name stays as the tooltip.
const METHODS: {
  id: TillPaymentMethod;
  label: "methodCash" | "methodCardShort" | "methodTransfer";
  full: "methodCash" | "methodCard" | "methodTransfer";
}[] = [
  { id: "cash", label: "methodCash", full: "methodCash" },
  { id: "card", label: "methodCardShort", full: "methodCard" },
  { id: "transfer", label: "methodTransfer", full: "methodTransfer" },
];

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
