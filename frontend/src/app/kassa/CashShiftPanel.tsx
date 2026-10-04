"use client";

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatDateTime, formatPrice } from "@/lib/format";
import { LuChevronDown } from "react-icons/lu";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import OverrideDialog from "@/components/till/OverrideDialog";
import { printReceipt, type PrintOutcome } from "@/lib/print";
import type { TillPayee } from "@/lib/types";
import { TillPager, usePaged } from "@/components/till/Pager";
import PrintResultDialog from "@/components/till/PrintResultDialog";
import type {
  CashEntry,
  CashFigures,
  CashShift,
  ShiftAge,
} from "@/lib/types";

/**
 * The drawer, on the screen standing in front of it.
 *
 * ⚠️ **Counting cash belongs here, not in the admin panel.** The alternative was
 * giving every cashier a panel login — the customer base, the payment keys, the
 * reports — or having the manager come in next morning to count a drawer
 * somebody else emptied. Both are worse than what they avoided.
 *
 * ⚠️ **The expected figure is not shown until a number has been typed.** Putting
 * it beside an empty box invites it to be copied, and a count that agrees
 * because it was read off the screen is the one record this whole thing exists
 * to prevent. It is the same rule the panel's screen follows, for the same
 * reason, and it is the only part of this that is not shared code.
 */
export default function CashShiftPanel({
  currency,
  onError,
  onChanged,
}: {
  currency: string;
  onError: (msg: string) => void;
  /** Told when the drawer opened or closed, so the screen's own gate agrees
   *  with this panel. ⚠️ Two readers of one fact drift: without this the
   *  cashier closes the shift here and keeps taking orders on a floor that
   *  still believes it is open. */
  onChanged?: () => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [shift, setShift] = useState<CashShift | null>(null);
  /** How long it has been open. ⚠️ From the server, not worked out here: the
   *  line it is measured against is a branch setting, and a till that decided
   *  for itself would disagree with the message the owner gets. */
  const [age, setAge] = useState<ShiftAge | null>(null);
  const [figures, setFigures] = useState<CashFigures | null>(null);
  // Hand-entered movements of this shift, listed under the form that makes
  // them. ⚠️ Shown rather than only totalled: when the count comes out short
  // the first thing anybody wants is the list, not the sum.
  const [entries, setEntries] = useState<CashEntry[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [printed, setPrinted] = useState<PrintOutcome | null>(null);

  const [float_, setFloat] = useState("");
  const [counted, setCounted] = useState("");
  const [varianceNote, setVarianceNote] = useState("");

  // The action a manager was asked to authorise, kept so the retry sends the
  // same numbers — retyping a count is how a drawer gets closed at a figure
  // nobody approved.
  const [override, setOverride] = useState<Pending | null>(null);
  const [overrideError, setOverrideError] = useState("");

  const money = (n: number) => formatPrice(n, "UZS", lang);

  const load = useCallback(async () => {
    try {
      const d = await api.tillCashShift();
      setAge(d.age ?? null);
      setShift(d.open);
      setFigures(d.figures ?? null);
      setEntries(d.entries ?? []);
    } catch {
      // Silent: a till that could not read the drawer still sells food, and
      // the panel simply stays collapsed.
    } finally {
      setLoaded(true);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function act(kind: Pending["kind"], pin: string, entry?: EntryDraft) {
    setBusy(true);
    setOverrideError("");
    try {
      if (kind === "entry") {
        if (!entry) return;
        const res = await api.tillAddCashEntry({ ...entry, pin: pin || undefined });
        setFigures(res.figures);
        setEntries(res.entries);
        setOverride(null);
        return;
      }
      if (kind === "open") {
        await api.tillOpenCashShift({
          openingFloat: Number(float_) || 0,
          pin: pin || undefined,
        });
        setFloat("");
      } else {
        const res = await api.tillCloseCashShift({
          counted: Number(counted) || 0,
          varianceNote: varianceNote.trim() || undefined,
          pin: pin || undefined,
          // ⚠️ The Z comes back from this call rather than being fetched after
          // it, so the language has to travel with the close — there is no
          // second request to attach it to.
          lang,
        });
        setCounted("");
        setVarianceNote("");
        // ⚠️ Printed here, immediately, and not offered as a button on a screen
        // the cashier is about to leave. The Z report is the paper for the
        // shift that just ended; asking somebody at 2am to remember one more
        // tap produces days with no Z report and no way to make one.
        if (res.lines?.length && !(res.queued ?? 0)) {
          void printReceipt(res.lines, res.widthMM);
        }
        // ⚠️ The register's day may have refused to end — usually because
        // receipts are still unfiled. Reported rather than swallowed: the
        // count succeeded either way, and the cashier is standing next to the
        // machine that can fix it.
        if (res.fiscalNote) onError(res.fiscalNote);
      }
      setOverride(null);
      await load();
      onChanged?.();
    } catch (err) {
      const perm = err instanceof ApiError ? err.needsOverride : null;
      if (perm) {
        if (pin) setOverrideError(t.till.overrideWrong);
        setOverride({
          kind,
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

  // ⚠️ The X report changes nothing, so it needs no permission beyond seeing
  // the drawer — it is the numbers already on this screen, on paper. Requiring
  // a manager to print what is already displayed teaches people that
  // permissions are theatre.
  async function printX() {
    setBusy(true);
    try {
      const res = await api.tillShiftReport(lang);
      setPrinted(await deliver(res));
    } catch (err) {
      onError(err instanceof ApiError ? err.message : t.till.retry);
    } finally {
      setBusy(false);
    }
  }

  // What became of the last print, when printing was the whole point.
  const printDialog = printed && (
    <PrintResultDialog outcome={printed} onClose={() => setPrinted(null)} />
  );

  if (!loaded) return null;

  const typed = counted !== "";
  const variance = (Number(counted) || 0) - (figures?.expected ?? 0);
  const needsNote = typed && variance !== 0 && !varianceNote.trim();

  return (
    <div className="rounded-2xl border border-line bg-surface p-3">
      <FoldHead
        title={shift ? t.till.shiftOpen : t.till.shiftClosed}
        open={open}
        onToggle={() => setOpen(!open)}
      />

      {/* ⚠️ **Said whether the panel is open or shut, and coloured.** A till
          with an open shift behaves exactly like a till working normally —
          checks open, money is taken, receipts print — which is why a drawer
          that stopped being counted three days ago goes unnoticed until the
          eventual close produces a variance nobody can attribute to a shift, a
          person or a day. This line is the only thing on the screen that says
          so. */}
      {age?.overdue && (
        <div className="mt-1 text-xs font-semibold text-danger">
          {t.till.shiftOverdue(age.hours)}
        </div>
      )}

      {/* Even collapsed, the one number worth a glance: what should be in the
          drawer right now. */}
      {shift && figures && !open && (
        <div className="mt-1 text-xs text-ink-muted">
          {t.cash.expected}: {money(figures.expected)}
        </div>
      )}

      {open && !shift && (
        <div className="mt-3">
          <label className="block text-sm">
            <span className="text-ink-muted">{t.cash.openingFloat}</span>
            <input
              className="till-input mt-1 h-11"
              inputMode="numeric"
              value={float_}
              onChange={(e) => setFloat(e.target.value.replace(/\D/g, ""))}
            />
          </label>
          <button
            className="till-btn-primary mt-2.5 w-full"
            disabled={busy}
            onClick={() => void act("open", "")}
          >
            {t.cash.openShift}
          </button>
        </div>
      )}

      {open && shift && figures && (
        <div className="mt-3 space-y-2 text-sm">
          <Row
            label={t.cash.openingFloat}
            value={money(figures.openingFloat)}
          />
          <Row label={t.cash.counterCash} value={money(figures.counterCash)} />
          {figures.settlements > 0 && (
            <Row
              label={t.cash.settlements}
              value={money(figures.settlements)}
            />
          )}
          {/* ⚠️ **Inside the counter total, and said so.** A guest paying off
              last Tuesday's slate puts money in today's drawer against a sale
              that belongs to another evening — so the box holds more than this
              shift sold, and without this line the difference is a number
              nobody standing at the till can explain. */}
          {(figures.debtPaid ?? 0) > 0 && (
            <Row
              label={`— ${t.cash.debtPaid}`}
              value={money(figures.debtPaid ?? 0)}
              quiet
            />
          )}
          {/* ⚠️ Hand-entered movements, shown even at zero once anything has
              been recorded: a payout the cashier made an hour ago is the first
              thing they look for when the count comes out short. */}
          {figures.manualIn > 0 && (
            <Row label={t.cash.manualIn} value={money(figures.manualIn)} />
          )}
          {figures.manualOut > 0 && (
            <Row
              label={t.cash.manualOut}
              value={`− ${money(figures.manualOut)}`}
            />
          )}
          <div className="flex justify-between gap-2 border-t border-line pt-2 font-semibold">
            <span>{t.cash.expected}</span>
            <span>{money(figures.expected)}</span>
          </div>
          {/* ⚠️ Apart from the list, because it is **not** in the total: cash a
              courier is still carrying is real money that is not in this
              drawer. Adding it would double every delivery. */}
          {figures.withCouriers > 0 && (
            <div className="rounded-xl bg-ink/5 p-2 text-xs">
              {t.cash.withCouriers}: {money(figures.withCouriers)}
            </div>
          )}

          {/* ⚠️ **Above the count, not beside the close button.** X is the
              report somebody prints *before* touching the drawer — to check
              the till mid-shift, or to see what the numbers say before they
              count. Putting it next to "close the shift" is where a tired
              cashier presses the wrong one. */}
          <button
            className="till-btn w-full"
            disabled={busy}
            onClick={() => void printX()}
          >
            {t.till.xReport}
          </button>

          <label className="block pt-2">
            <span className="text-ink-muted">{t.cash.counted}</span>
            <input
              className="till-input mt-1 h-11"
              inputMode="numeric"
              value={counted}
              onChange={(e) => setCounted(e.target.value.replace(/\D/g, ""))}
            />
          </label>

          {typed && (
            <div
              className={`rounded-xl p-2 text-sm ${
                variance === 0 ? "bg-success/10" : "bg-danger/10"
              }`}
            >
              <div className="flex justify-between text-xs text-ink-muted">
                <span>{t.cash.expected}</span>
                <span>{money(figures.expected)}</span>
              </div>
              <div className="mt-1 flex justify-between font-medium">
                <span>{t.cash.variance}</span>
                <span>
                  {variance > 0 ? "+" : ""}
                  {money(variance)}
                </span>
              </div>
            </div>
          )}

          {typed && variance !== 0 && (
            <input
              className="till-input h-11"
              placeholder={t.cash.varianceNote}
              value={varianceNote}
              onChange={(e) => setVarianceNote(e.target.value)}
            />
          )}

          <button
            className="till-btn-primary w-full"
            disabled={busy || !typed || needsNote}
            onClick={() => void act("close", "")}
          >
            {t.cash.closeShift}
          </button>

          <CashEntries
            entries={entries}
            busy={busy}
            onAdd={(body) => act("entry", "", body)}
          />
        </div>
      )}

      {/* ---- Yesterday's paper ----

          ⚠️ **Outside the open-shift block, because it is asked for when there
          is no shift.** "Print last night's again" is a question somebody has
          at nine in the morning, before the drawer is opened — and a control
          that only existed while a shift was running would be missing at
          exactly the hour it is wanted. */}
      {open && <ClosedShifts busy={busy} onError={onError} />}

      {override && (
        <OverrideDialog
          permissionName={override.permissionName}
          busy={busy}
          error={overrideError}
          onCancel={() => {
            setOverride(null);
            setOverrideError("");
          }}
          onSubmit={(pin) => void act(override.kind, pin)}
        />
      )}
    </div>
  );
}

function Row({
  label,
  value,
  quiet,
}: {
  label: string;
  value: string;
  /** A line that explains part of the one above rather than adding to it. */
  quiet?: boolean;
}) {
  return (
    <div className={`flex justify-between gap-2 ${quiet ? "text-xs" : ""}`}>
      <span className="text-ink-muted">{label}</span>
      <span className={quiet ? "text-ink-muted" : ""}>{value}</span>
    </div>
  );
}

/** Whether a reason chip means a wage, in any of the three languages. */
function wageWord(r: string): boolean {
  return /^(maosh|ish haqi|зарплата|wages?)$/i.test(r.trim());
}

/** What a cash movement needs before it can be written. */
export interface EntryDraft {
  kind: "in" | "out";
  category: string;
  amount: number;
  note?: string;
  /** ⚠️ Whether the notes moved between the drawer and the office box. Asked,
   *  never inferred from the direction: money out of the drawer goes to a
   *  supplier as often as it goes to the safe. */
  toSafe?: boolean;
  /** Who a wage was handed to, picked from the list. */
  personKind?: "staff" | "courier";
  personId?: string;
}

interface Pending {
  kind: "open" | "close" | "entry";
  permissionName: string;
}

/** Money in and out of the drawer, and what has already gone through it.
 *
 *  ⚠️ **On the till, because the cashier is the one who does it.** A supplier
 *  paid at the door, change brought in at six, a courier's float — all of it
 *  happens at the counter, and until now the only place to record one was the
 *  admin panel. That meant either a panel login on a till (the customer base,
 *  the payment keys, the reports) or a drawer whose figure nobody could
 *  reconcile because half its movements were never written down.
 *
 *  ⚠️ **Folded away.** It is used a few times an evening, and unfolded it would
 *  sit between the expected total and the count — the two things this panel is
 *  opened for. */
function CashEntries({
  entries,
  busy,
  onAdd,
}: {
  entries: CashEntry[];
  busy: boolean;
  onAdd: (body: EntryDraft) => void | Promise<void>;
}) {
  const t = useAdminT();
  const [open, setOpen] = useState(false);
  const [kind, setKind] = useState<"in" | "out">("out");
  const [category, setCategory] = useState("");
  const [amount, setAmount] = useState("");
  const [note, setNote] = useState("");
  const [toSafe, setToSafe] = useState(false);
  // Who the money was handed to, when it is a wage.
  //
  // ⚠️ Fetched only once this section is opened: it is used a few times an
  // evening, and a counter should not load a staff list to sell a lagmon.
  const [payees, setPayees] = useState<TillPayee[]>([]);
  const [payee, setPayee] = useState("");

  const ready = category.trim() !== "" && Number(amount) > 0;
  const picked = payees.find((p) => `${p.kind}:${p.id}` === payee);
  // ⚠️ **The staff list belongs to the wage, and to nothing else.** It used to
  // sit under every "out" — a supplier paid at the door, change for the
  // drawer, a bag of onions — asking "to whom?" about money that was going to
  // no employee at all. Shown when the reason is a wage: the chip, or the same
  // word typed.
  const isWage = kind === "out" && wageWord(category);
  const paged = usePaged(entries, 8);

  useEffect(() => {
    if (!open || payees.length > 0) return;
    api
      .tillPayees()
      .then((r) => setPayees(r.payees))
      // Silent: paying somebody without naming them still records the cash,
      // which is the fact that matters most at a counter.
      .catch(() => setPayees([]));
  }, [open, payees.length]);

  return (
    <Fold
      title={t.cash.entryTitle}
      open={open}
      onToggle={() => setOpen(!open)}
    >
      {open && (
        <div className="space-y-2">
          {/* ⚠️ Two buttons rather than a select. Which direction the money is
              going is the one thing that must not be got wrong here, and a
              dropdown showing the wrong side of it looks exactly like the right
              one. */}
          <div className="till-seg-track">
            <button
              className={kind === "in" ? "till-seg-on" : "till-seg"}
              onClick={() => setKind("in")}
            >
              {t.cash.entryIn}
            </button>
            <button
              className={kind === "out" ? "till-seg-on" : "till-seg"}
              onClick={() => setKind("out")}
            >
              {t.cash.entryOut}
            </button>
          </div>

          {/* Ready-made reasons, and a free field beside them: every kitchen
              spends money on something the next one does not, and a fixed list
              sends all of it to "other" — the same judgement the panel's
              categories are built on. */}
          <div className="flex flex-wrap gap-1.5">
            {t.cash.entryReasons.map((r) => (
              <button
                key={r}
                className={category === r ? "till-chip-btn-on" : "till-chip-btn"}
                onClick={() => {
                  setCategory(r);
                  // Leaving the wage forgets the person: a supplier payment
                  // must not quietly go to somebody's payroll.
                  if (!wageWord(r)) setPayee("");
                }}
              >
                {r}
              </button>
            ))}
          </div>
          <input
            className="till-input h-11"
            placeholder={t.cash.entryReason}
            value={category}
            onChange={(e) => setCategory(e.target.value)}
          />
          <input
            className="till-input h-11"
            inputMode="numeric"
            placeholder={t.cash.entryAmount}
            value={amount}
            onChange={(e) => setAmount(e.target.value.replace(/\D/g, ""))}
          />
          <input
            className="till-input h-11"
            placeholder={t.cash.entryNote}
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
          {/* ---- Who was paid ----

              ⚠️ **Only on the way out, and only from the list.** A wage handed
              over the counter used to leave the drawer correct and payroll
              wrong: the courier was still owed his whole month, so he was
              either paid twice or argued with about a Tuesday nobody could
              remember. Picking the person writes the wage document too.

              ⚠️ Names are never typed here. "Aziz", "aziz", "Азиз" and "Aziz
              kuryer" all appear within a week, and none can be matched to the
              person payroll owes. */}
          {isWage && payees.length > 0 && (
            <select
              className="till-input h-11"
              value={payee}
              onChange={(e) => {
                setPayee(e.target.value);
                const p = payees.find(
                  (x) => `${x.kind}:${x.id}` === e.target.value,
                );
                // The reason writes itself when a person is named: choosing a
                // courier has already said what this is.
                if (p) setCategory(t.cash.entryWage);
              }}
            >
              <option value="">{t.cash.entryWhoNone}</option>
              {payees.map((p) => (
                <option key={`${p.kind}:${p.id}`} value={`${p.kind}:${p.id}`}>
                  {p.name}
                  {p.kind === "courier" ? ` · ${t.cash.entryCourier}` : ""}
                </option>
              ))}
            </select>
          )}

          {/* ⚠️ The collection. Emptying the drawer into the office box is the
              one movement here that costs the restaurant nothing — the money
              only changed shelves — and it was also the one nobody recorded,
              which is why the safe used to be short by exactly a day's
              takings. */}
          <label className="flex items-center gap-2 px-1 py-1 text-sm text-ink-muted">
            <input
              type="checkbox"
              className="h-5 w-5"
              checked={toSafe}
              onChange={(e) => setToSafe(e.target.checked)}
            />
            {kind === "out" ? t.cash.toSafe : t.cash.fromSafe}
          </label>
          <button
            className="till-btn w-full"
            disabled={busy || !ready}
            onClick={async () => {
              // ⚠️ **The person goes with the entry.** It was picked and
              // then never sent: the drawer was right, payroll still owed the
              // whole month, and the comment above promising otherwise was
              // describing a field that never left this component.
              const who = isWage ? picked : undefined;
              await onAdd({
                kind,
                category: category.trim(),
                amount: Number(amount) || 0,
                note: note.trim() || undefined,
                toSafe,
                personKind: who?.kind,
                personId: who?.id,
              });
              setAmount("");
              setNote("");
              setCategory("");
              setToSafe(false);
              setPayee("");
            }}
          >
            {t.cash.entrySave}
          </button>

          {entries.length > 0 && (
            <ul className="space-y-1 pt-1 text-xs">
              {paged.shown.map((e) => (
                <li key={e.id} className="flex justify-between gap-2">
                  <span className="truncate text-ink-muted">
                    {e.category}
                    {e.note ? ` · ${e.note}` : ""}
                  </span>
                  <span
                    className={
                      e.kind === "out" ? "text-danger" : "text-[rgb(var(--till-ok))]"
                    }
                  >
                    {e.kind === "out" ? "−" : "+"}
                    {formatPrice(e.amount)}
                  </span>
                </li>
              ))}
            </ul>
          )}
          <TillPager page={paged.page} pages={paged.pages} onPage={paged.setPage} />
        </div>
      )}
    </Fold>
  );
}

/** The last few closed shifts, each with its Z report.
 *
 *  ⚠️ **A Z report is printed once, and once is not always enough.** The roll
 *  jams, the paper runs out, the accountant asks for Tuesday's in March. Until
 *  this existed the only copy was the one that came out of the printer at the
 *  moment the shift closed — and re-deriving those figures by hand is how two
 *  different answers about one evening start existing.
 *
 *  ⚠️ **Loaded when it is unfolded, not with the panel.** This is opened a few
 *  times a month; fetching it on every till that shows the drawer would be a
 *  query per screen per load for a list almost nobody is looking at. */
function ClosedShifts({
  busy,
  onError,
}: {
  busy: boolean;
  onError: (message: string) => void;
}) {
  const t = useAdminT();
  // ⚠️ The Z is read by whoever pressed the button and nobody else, so it is
  // printed in the language of the screen they pressed it on.
  const { lang } = useI18n();
  const [open, setOpen] = useState(false);
  const [rows, setRows] = useState<CashShift[] | null>(null);
  // ⚠️ Its own, not the panel's: this is a separate component and a shared one
  // would show the wrong dialog over the wrong list.
  const [printed, setPrinted] = useState<PrintOutcome | null>(null);
  const [working, setWorking] = useState(false);
  // ⚠️ Paged: a till open for a few months has a long tail of shifts, and
  // the fold had become a column longer than the screen.
  const paged = usePaged(rows ?? [], 8);

  async function toggle() {
    const next = !open;
    setOpen(next);
    if (!next || rows) return;
    try {
      const res = await api.tillClosedShifts();
      setRows(res.shifts);
    } catch (err) {
      onError(err instanceof ApiError ? err.message : t.till.retry);
      setRows([]);
    }
  }

  async function printZ(id: string) {
    setWorking(true);
    try {
      const res = await api.tillShiftZReport(id, lang);
      setPrinted(await deliver(res));
    } catch (err) {
      onError(err instanceof ApiError ? err.message : t.till.retry);
    } finally {
      setWorking(false);
    }
  }

  return (
    <>
      {printed && (
        <PrintResultDialog outcome={printed} onClose={() => setPrinted(null)} />
      )}
    <Fold title={t.cash.zTitle} open={open} onToggle={() => void toggle()}>
      {open && (
        <div className="space-y-1.5">
          {rows?.length === 0 && (
            <p className="text-xs text-ink-muted">{t.cash.zNone}</p>
          )}
          {paged.shown.map((sh) => (
            <div
              key={sh.id}
              className="flex items-center justify-between gap-2 rounded-xl bg-ink/[0.03] p-2"
            >
              <span className="min-w-0 text-xs">
                <span className="block truncate">
                  {sh.closedAt ? t.cash.zClosedAt(formatDateTime(sh.closedAt)) : ""}
                </span>
                {/* The number an owner scans this list for: what the drawer was
                    out by. Silent when it balanced — a zero on every row would
                    bury the one row that is not. */}
                {typeof sh.variance === "number" && sh.variance !== 0 && (
                  <span className="text-danger">
                    {sh.variance > 0 ? "+" : ""}
                    {formatPrice(sh.variance)}
                  </span>
                )}
              </span>
              <button
                className="till-btn shrink-0 px-3"
                disabled={busy || working}
                onClick={() => void printZ(sh.id)}
              >
                {t.cash.zPrint}
              </button>
            </div>
          ))}
          <TillPager page={paged.page} pages={paged.pages} onPage={paged.setPage} />
        </div>
      )}
    </Fold>
    </>
  );
}

/** The header of a fold, when the section already has a card of its own.
 *
 *  ⚠️ Same shape as the one inside `Fold` rather than a second design: these
 *  three controls sit within a few centimetres of each other, and a set that
 *  does not look like a set is what taught cashiers none of them were
 *  pressable. */
function FoldHead({
  title,
  open,
  onToggle,
}: {
  title: string;
  open: boolean;
  onToggle: () => void;
}) {
  return (
    <button
      type="button"
      aria-expanded={open}
      /* The same bleed as the menu tile's photograph, and safe here: the
         parent is a block rather than a flex line, so the percentage resolves
         against a box this element's own margins do not change. */
      className="-m-1 flex w-[calc(100%+0.5rem)] items-center justify-between gap-2 rounded-xl p-1 text-left transition active:bg-ink/[0.05]"
      onClick={onToggle}
    >
      <span className="text-sm font-medium text-ink">{title}</span>
      <span className="grid h-7 w-7 shrink-0 place-items-center rounded-full bg-ink/[0.06]">
        <LuChevronDown
          className={`h-4 w-4 text-ink-soft transition-transform ${
            open ? "rotate-180" : ""
          }`}
          aria-hidden
        />
      </span>
    </button>
  );
}

/** A section of the drawer panel that folds away.
 *
 *  ⚠️ **It was a line of text with a ▲ beside it, and cashiers did not know it
 *  could be pressed.** On a monoblock there is no cursor to change shape and no
 *  hover to discover, so an affordance that relies on either is not an
 *  affordance — the row simply read as a heading, and "Kirim/chiqim" and the
 *  closed shifts might as well not have existed. What makes a thing tappable on
 *  a touch screen is that it looks like the other things that are: a filled
 *  row, a border, a chevron in a circle, and something that visibly moves when
 *  pressed.
 *
 *  ⚠️ **One component for all three**, because they are the same control and
 *  the inconsistency was part of the problem: two of them sat on a hairline
 *  border and one did not, so none of them read as a set.
 */
function Fold({
  title,
  hint,
  open,
  onToggle,
  children,
}: {
  title: string;
  /** One line worth seeing without opening it — the expected drawer figure. */
  hint?: React.ReactNode;
  open: boolean;
  onToggle: () => void;
  children: React.ReactNode;
}) {
  return (
    <div className="mt-2 overflow-hidden rounded-xl border border-line">
      <button
        type="button"
        aria-expanded={open}
        className="flex w-full items-center justify-between gap-2 bg-ink/[0.03] px-3 py-2.5 text-left transition active:bg-ink/[0.07]"
        onClick={onToggle}
      >
        <span className="min-w-0">
          <span className="block text-sm font-medium text-ink">{title}</span>
          {hint && !open && (
            <span className="mt-0.5 block text-xs text-ink-muted">{hint}</span>
          )}
        </span>
        {/* ⚠️ A chevron in a filled circle, rotating rather than swapping
            characters. The old ▲/▼ pair changed glyph without changing shape,
            which at arm's length on a counter is not a change at all. */}
        <span className="grid h-7 w-7 shrink-0 place-items-center rounded-full bg-ink/[0.06]">
          <LuChevronDown
            className={`h-4 w-4 text-ink-soft transition-transform ${
              open ? "rotate-180" : ""
            }`}
            aria-hidden
          />
        </span>
      </button>
      {open && <div className="px-3 pb-3 pt-2">{children}</div>}
    </div>
  );
}

/** Where a report actually went.
 *
 * ⚠️ **The branch's printers first, this machine's second.** The till used to
 * print these itself, through a path that reads the *local* printer — and a
 * restaurant whose printers are all on the network has none, so the X report,
 * the Z report and a reprint fell through to a browser dialog that prints on no
 * monoblock anywhere. Kitchen tickets were fine the whole time because they go
 * through the queue; these three were the ones that produced no paper.
 *
 * ⚠️ Falling back is still right: a branch that has configured no printer at
 * all needs the browser's dialog, which is how every restaurant's first evening
 * goes.
 */
async function deliver(res: {
  lines: string[];
  widthMM: number;
  queued?: number;
}): Promise<PrintOutcome> {
  if ((res.queued ?? 0) > 0) return "printed";
  return printReceipt(res.lines, res.widthMM);
}
