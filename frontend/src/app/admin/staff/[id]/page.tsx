"use client";

// One employee's card: the roster, the calendar those punches produced, the
// comparison against the previous week and month, and the money.
//
// It is also where attendance gets corrected. A phone dies, somebody forgets to
// clock out — and a time clock that cannot be corrected is a time clock the
// restaurant stops using by the second week. Every correction is signed and
// logged.

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { ApiError, api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { formatPrice, formatUzPhone, weekdayName } from "@/lib/format";
import { formatDateTime } from "@/lib/orderFlow";
import Modal from "@/components/admin/Modal";
import AttendanceCalendar from "@/components/staff/AttendanceCalendar";
import { ListScroll, Pager, usePaged } from "@/components/admin/PagedList";
import {
  MONDAY_ORDER,
  formatDuration,
  payModeLabel,
  payPeriodLabel,
  payUnitLabel,
  shortDate,
  toDayKey,
} from "@/lib/attendance";
import type { AdminStaffDetail, StaffDay, StaffTrend } from "@/lib/types";

interface ShiftDraft {
  id: string;
  date: string;
  in: string;
  out: string;
  note: string;
}

const inputCls =
  "mt-1 w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand";

export default function AdminStaffCardPage() {
  const params = useParams<{ id: string }>();
  const id = params.id;
  const t = useAdminT();
  const { lang } = useI18n();

  const [data, setData] = useState<AdminStaffDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [range, setRange] = useState<{ from: string; to: string } | null>(null);
  const [shiftDraft, setShiftDraft] = useState<ShiftDraft | null>(null);
  const [payOpen, setPayOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const dur = useCallback(
    (m: number) => formatDuration(m, t.staff.hoursShort, t.staff.minutesShort),
    [t],
  );

  const load = useCallback(() => {
    api
      .adminStaffMember(id, range?.from, range?.to)
      .then(setData)
      .catch(() => setData(null))
      .finally(() => setLoading(false));
  }, [id, range]);

  useEffect(load, [load]);

  const payments = usePaged(data?.payments ?? [], 10);

  if (loading) {
    return <p className="text-sm text-ink-muted">{t.common.loading}</p>;
  }
  if (!data) {
    return (
      <div className="space-y-3">
        <p className="text-sm text-ink-muted">{t.staff.notFound}</p>
        <Link href="/admin/staff" className="text-sm text-brand hover:underline">
          {t.staff.backToList}
        </Link>
      </div>
    );
  }

  const { staff, report } = data;

  async function saveShift() {
    if (!shiftDraft) return;
    setSaving(true);
    setError(null);
    const body = {
      date: shiftDraft.date,
      in: shiftDraft.in,
      out: shiftDraft.out,
      note: shiftDraft.note,
    };
    try {
      if (shiftDraft.id) {
        await api.updateShift(id, shiftDraft.id, body);
      } else {
        await api.createShift(id, body);
      }
      setShiftDraft(null);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  async function removeShift(shiftId: string) {
    if (!window.confirm(t.staff.confirmDeleteShift)) return;
    try {
      await api.deleteShift(id, shiftId);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    }
  }

  async function removePayment(paymentId: string) {
    if (!window.confirm(t.staff.confirmDeletePayment)) return;
    try {
      await api.deleteStaffPayment(id, paymentId);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    }
  }

  return (
    <div className="space-y-5">
      <Link href="/admin/staff" className="text-sm text-brand hover:underline">
        {t.staff.backToList}
      </Link>

      <header className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="font-display text-2xl font-bold">
            {staff.name}
            {!staff.isActive && (
              <span className="ml-2 text-sm font-normal text-ink-muted">
                {t.staff.disabled}
              </span>
            )}
          </h1>
          <p className="mt-0.5 text-sm text-ink-muted">
            {staff.position || `@${staff.username}`}
            {staff.phone && ` · ${formatUzPhone(staff.phone)}`}
            {data.branchName && ` · ${data.branchName}`}
          </p>
        </div>
        <button
          type="button"
          onClick={() => setPayOpen(true)}
          className="btn-primary px-4 py-2 text-sm"
        >
          {t.staff.payButton}
        </button>
      </header>

      {error && <p className="text-sm text-red-600">{error}</p>}

      {/* Roster — what the calendar below is measured against. */}
      <section className="rounded-2xl border border-line bg-surface p-4 shadow-card">
        <p className="text-sm font-semibold">{t.staff.scheduleTitle}</p>
        <div className="mt-2 flex flex-wrap gap-x-5 gap-y-1 text-sm">
          {MONDAY_ORDER.map((day) => {
            const row = staff.schedule?.find((s) => s.day === day);
            const off = !row || row.isOff || !row.start || !row.end;
            return (
              <span key={day} className={off ? "text-ink-muted" : ""}>
                <span className="capitalize">{weekdayName(day, lang)}</span>:{" "}
                {off ? t.staff.dayOff : `${row!.start}–${row!.end}`}
              </span>
            );
          })}
        </div>
      </section>

      {/* Pay rule and the current period's balance. */}
      <section className="grid gap-3 sm:grid-cols-3">
        <Card
          label={t.staff.payMode}
          value={payModeLabel(report.payMode, t.staff)}
          hint={`${formatPrice(report.rate, "UZS", lang)} ${payUnitLabel(report.payMode, t.staff)} · ${payPeriodLabel(report.payPeriod, t.staff)}`}
        />
        <Card
          label={t.staff.periodEarned}
          value={formatPrice(report.periodPay, "UZS", lang)}
          hint={`${shortDate(report.periodFrom)} — ${shortDate(report.periodTo)}`}
        />
        <Card
          label={report.periodDue < 0 ? t.staff.overpaid : t.staff.periodDue}
          value={formatPrice(Math.abs(report.periodDue), "UZS", lang)}
          hint={`${t.staff.periodPaid}: ${formatPrice(report.periodPaid, "UZS", lang)}`}
          accent
        />
      </section>

      {/* Trends: each against its own previous window, independent of the
          range being browsed below. */}
      <section className="rounded-2xl border border-line bg-surface p-4 shadow-card">
        <p className="text-sm font-semibold">{t.staff.trendsTitle}</p>
        <div className="mt-3 grid gap-2 sm:grid-cols-3">
          {(
            [
              [t.staff.trendToday, report.today],
              [t.staff.trendWeek, report.week],
              [t.staff.trendMonth, report.month],
            ] as const
          ).map(([label, trend]) => (
            <Trend key={label} label={label} trend={trend} dur={dur} t={t} />
          ))}
        </div>
      </section>

      {/* Range + totals + calendar */}
      <section className="space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 className="font-display text-lg font-bold">
            {t.staff.calendarTitle}
          </h2>
          <button
            type="button"
            onClick={() =>
              setShiftDraft({
                id: "",
                date: toDayKey(new Date()),
                in: "",
                out: "",
                note: "",
              })
            }
            className="btn-ghost px-3 py-1.5 text-sm"
          >
            {t.staff.addShift}
          </button>
        </div>

        <RangePicker range={range} setRange={setRange} t={t} />

        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
          <Card label={t.staff.totalWorked} value={dur(report.totals.worked)} />
          <Card
            label={t.staff.totalExpected}
            value={dur(report.totals.expected)}
          />
          <Card label={t.staff.daysWorked} value={String(report.totals.days)} />
          <Card
            label={t.staff.daysAbsent}
            value={String(report.totals.absent)}
          />
          <Card label={t.staff.overtime} value={dur(report.totals.overtime)} />
          <Card
            label={t.staff.periodEarned}
            value={formatPrice(report.totals.pay, "UZS", lang)}
            accent
          />
        </div>

        <AttendanceCalendar
          days={report.days}
          renderDayExtra={(day) => (
            <DayActions
              day={day}
              t={t}
              onAdd={() =>
                setShiftDraft({
                  id: "",
                  date: day.date,
                  in: day.planStart || "",
                  out: day.planEnd || "",
                  note: "",
                })
              }
              onEdit={(session) =>
                setShiftDraft({
                  id: session.id,
                  date: day.date,
                  in: new Date(session.in).toTimeString().slice(0, 5),
                  out: session.out
                    ? new Date(session.out).toTimeString().slice(0, 5)
                    : "",
                  note: session.note ?? "",
                })
              }
              onDelete={removeShift}
            />
          )}
        />

        <p className="text-xs leading-relaxed text-ink-muted">
          {t.staff.manualHint}
        </p>
      </section>

      {/* Payment ledger */}
      <section className="rounded-2xl border border-line bg-surface shadow-card">
        <div className="flex items-center justify-between border-b border-line px-4 py-3">
          <h2 className="text-sm font-semibold">{t.staff.paymentsTitle}</h2>
          <span className="text-xs text-ink-muted">
            {t.staff.totalPaid}: {formatPrice(data.paidTotal, "UZS", lang)}
          </span>
        </div>
        {data.payments.length === 0 ? (
          <p className="p-6 text-sm text-ink-muted">{t.staff.paymentsEmpty}</p>
        ) : (
          <>
            <ListScroll max="max-h-96">
              <table className="w-full text-sm">
                <tbody>
                  {payments.pageItems.map((p) => (
                    <tr key={p.id} className="border-b border-line last:border-0">
                      <td className="px-4 py-2.5">
                        <span className="font-semibold tabular-nums">
                          {formatPrice(p.amount, "UZS", lang)}
                        </span>
                        <p className="text-xs text-ink-muted">
                          {shortDate(p.from)} — {shortDate(p.to)}
                          {p.paidBy && ` · ${t.staff.paidBy(p.paidBy)}`}
                          {p.note && ` · ${p.note}`}
                        </p>
                      </td>
                      <td className="px-4 py-2.5 text-right text-xs text-ink-muted whitespace-nowrap">
                        {formatDateTime(p.at)}
                        <button
                          type="button"
                          onClick={() => removePayment(p.id)}
                          className="ml-3 text-red-600 hover:underline"
                        >
                          {t.common.delete}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </ListScroll>
            <Pager {...payments} onPage={payments.setPage} />
          </>
        )}
      </section>

      {shiftDraft && (
        <Modal onClose={() => setShiftDraft(null)}>
          <h2 className="font-display text-lg font-bold">
            {shiftDraft.id ? t.staff.editShift : t.staff.addShift}
          </h2>
          <label className="mt-4 block text-sm">
            <span className="font-medium">{t.staff.shiftDate}</span>
            <input
              type="date"
              className={inputCls}
              value={shiftDraft.date}
              onChange={(e) =>
                setShiftDraft({ ...shiftDraft, date: e.target.value })
              }
            />
          </label>
          <div className="mt-3 grid grid-cols-2 gap-3">
            <label className="block text-sm">
              <span className="font-medium">{t.staff.shiftIn}</span>
              <input
                type="time"
                className={inputCls}
                value={shiftDraft.in}
                onChange={(e) =>
                  setShiftDraft({ ...shiftDraft, in: e.target.value })
                }
              />
            </label>
            <label className="block text-sm">
              <span className="font-medium">{t.staff.shiftOut}</span>
              <input
                type="time"
                className={inputCls}
                value={shiftDraft.out}
                onChange={(e) =>
                  setShiftDraft({ ...shiftDraft, out: e.target.value })
                }
              />
              <span className="mt-1 block text-[11px] text-ink-muted">
                {t.staff.shiftOutHint}
              </span>
            </label>
          </div>
          <label className="mt-3 block text-sm">
            <span className="font-medium">{t.staff.shiftNote}</span>
            <input
              className={inputCls}
              value={shiftDraft.note}
              onChange={(e) =>
                setShiftDraft({ ...shiftDraft, note: e.target.value })
              }
            />
          </label>

          {error && <p className="mt-3 text-sm text-red-600">{error}</p>}

          <div className="mt-5 flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setShiftDraft(null)}
              className="btn-ghost px-4 py-2 text-sm"
            >
              {t.common.cancel}
            </button>
            <button
              type="button"
              disabled={saving || !shiftDraft.in}
              onClick={saveShift}
              className="btn-primary px-4 py-2 text-sm disabled:opacity-50"
            >
              {saving ? t.common.saving : t.common.save}
            </button>
          </div>
        </Modal>
      )}

      {payOpen && (
        <PayModal
          name={staff.name}
          defaultAmount={Math.max(0, report.periodDue)}
          from={report.periodFrom}
          to={report.periodTo}
          onClose={() => setPayOpen(false)}
          onDone={() => {
            setPayOpen(false);
            load();
          }}
          staffId={id}
        />
      )}
    </div>
  );
}

// ---- pieces ----

function Card({
  label,
  value,
  hint,
  accent,
}: {
  label: string;
  value: string;
  hint?: string;
  accent?: boolean;
}) {
  return (
    <div className="rounded-2xl border border-line bg-surface p-3 shadow-card">
      <p className="text-[11px] uppercase tracking-wider text-ink-muted">
        {label}
      </p>
      <p
        className={`mt-0.5 font-display text-lg font-bold tabular-nums ${
          accent ? "text-brand" : ""
        }`}
      >
        {value}
      </p>
      {hint && <p className="text-[11px] text-ink-muted">{hint}</p>}
    </div>
  );
}

function Trend({
  label,
  trend,
  dur,
  t,
}: {
  label: string;
  trend: StaffTrend;
  dur: (m: number) => string;
  t: ReturnType<typeof useAdminT>;
}) {
  const up = trend.percent > 0;
  return (
    <div className="rounded-2xl bg-ink/[0.03] px-3 py-2">
      <p className="text-[11px] uppercase tracking-wider text-ink-muted">
        {label}
      </p>
      <p className="mt-0.5 font-semibold tabular-nums">{dur(trend.current)}</p>
      <p
        className={`text-xs ${
          !trend.hasPrev
            ? "text-ink-muted/70"
            : trend.percent === 0
              ? "text-ink-muted"
              : up
                ? "text-sky-600"
                : "text-amber-600"
        }`}
      >
        {!trend.hasPrev
          ? t.staff.trendNoPrev
          : trend.percent === 0
            ? t.staff.trendSame
            : up
              ? t.staff.trendUp(String(trend.percent))
              : t.staff.trendDown(String(Math.abs(trend.percent)))}
      </p>
    </div>
  );
}

/** Correction buttons hung under the selected day in the calendar. */
function DayActions({
  day,
  t,
  onAdd,
  onEdit,
  onDelete,
}: {
  day: StaffDay;
  t: ReturnType<typeof useAdminT>;
  onAdd: () => void;
  onEdit: (session: StaffDay["sessions"][number]) => void;
  onDelete: (shiftId: string) => void;
}) {
  return (
    <div className="mt-3 flex flex-wrap gap-2 border-t border-line pt-3">
      <button type="button" onClick={onAdd} className="btn-ghost px-3 py-1.5 text-xs">
        {t.staff.addShift}
      </button>
      {day.sessions.map((s) => (
        <span key={s.id} className="flex items-center gap-1">
          <button
            type="button"
            onClick={() => onEdit(s)}
            className="btn-ghost px-3 py-1.5 text-xs"
          >
            {t.common.edit}
          </button>
          <button
            type="button"
            onClick={() => onDelete(s.id)}
            className="px-2 py-1.5 text-xs text-red-600 hover:underline"
          >
            {t.staff.deleteShift}
          </button>
        </span>
      ))}
    </div>
  );
}

function PayModal({
  staffId,
  name,
  defaultAmount,
  from,
  to,
  onClose,
  onDone,
}: {
  staffId: string;
  name: string;
  defaultAmount: number;
  from: string;
  to: string;
  onClose: () => void;
  onDone: () => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [amount, setAmount] = useState(String(defaultAmount));
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submit() {
    const value = Number(amount) || 0;
    if (value <= 0) return;
    setBusy(true);
    setError(null);
    try {
      await api.payStaff(staffId, { amount: value, from, to, note });
      onDone();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal onClose={onClose}>
      <h2 className="font-display text-lg font-bold">
        {t.staff.payModalTitle(name)}
      </h2>
      <p className="mt-1 text-sm text-ink-muted">
        {t.staff.periodRange(shortDate(from), shortDate(to))}
      </p>

      <label className="mt-4 block text-sm">
        <span className="font-medium">{t.staff.payAmount}</span>
        <input
          className={inputCls}
          inputMode="numeric"
          value={amount}
          onChange={(e) => setAmount(e.target.value.replace(/\D/g, ""))}
        />
      </label>
      <label className="mt-3 block text-sm">
        <span className="font-medium">{t.staff.payNote}</span>
        <input
          className={inputCls}
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
      </label>

      <p className="mt-3 text-xs text-ink-muted">{t.staff.payHint}</p>
      {error && <p className="mt-3 text-sm text-red-600">{error}</p>}

      <div className="mt-5 flex justify-end gap-2">
        <button type="button" onClick={onClose} className="btn-ghost px-4 py-2 text-sm">
          {t.common.cancel}
        </button>
        <button
          type="button"
          disabled={busy || !Number(amount)}
          onClick={submit}
          className="btn-primary px-4 py-2 text-sm disabled:opacity-50"
        >
          {busy ? t.common.saving : t.staff.payConfirm}
        </button>
      </div>
    </Modal>
  );
}

/** Presets plus two date inputs. `null` means "let the server decide", which is
 *  this calendar month. */
function RangePicker({
  range,
  setRange,
  t,
}: {
  range: { from: string; to: string } | null;
  setRange: (r: { from: string; to: string } | null) => void;
  t: ReturnType<typeof useAdminT>;
}) {
  function preset(kind: "month" | "last" | "d7" | "d30") {
    const now = new Date();
    if (kind === "month") return setRange(null);
    if (kind === "last") {
      const first = new Date(now.getFullYear(), now.getMonth() - 1, 1);
      const last = new Date(now.getFullYear(), now.getMonth(), 0);
      return setRange({ from: toDayKey(first), to: toDayKey(last) });
    }
    const days = kind === "d7" ? 6 : 29;
    const first = new Date(now);
    first.setDate(first.getDate() - days);
    setRange({ from: toDayKey(first), to: toDayKey(now) });
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      {(
        [
          ["month", t.staff.rangeThisMonth],
          ["last", t.staff.rangeLastMonth],
          ["d7", t.staff.range7],
          ["d30", t.staff.range30],
        ] as const
      ).map(([kind, label]) => (
        <button
          key={kind}
          type="button"
          onClick={() => preset(kind)}
          className="chip"
        >
          {label}
        </button>
      ))}
      <input
        type="date"
        className="rounded-xl border border-line-strong bg-surface px-2 py-1 text-xs outline-none focus:border-brand"
        value={range?.from ?? ""}
        onChange={(e) =>
          setRange({ from: e.target.value, to: range?.to || e.target.value })
        }
      />
      <span className="text-xs text-ink-muted">—</span>
      <input
        type="date"
        className="rounded-xl border border-line-strong bg-surface px-2 py-1 text-xs outline-none focus:border-brand"
        value={range?.to ?? ""}
        onChange={(e) =>
          setRange({ from: range?.from || e.target.value, to: e.target.value })
        }
      />
    </div>
  );
}
