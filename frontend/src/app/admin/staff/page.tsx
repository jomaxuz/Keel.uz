"use client";

// Staff accounts and the attendance board.
//
// The list is not a phonebook: each row carries what today looks like and what
// the current pay period owes, because those are the two questions that make
// anybody open this screen. The card behind each row has the calendar.

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { ApiError, api } from "@/lib/api";
import { useAdminScope } from "@/lib/adminScope";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { formatPrice, formatUzPhone } from "@/lib/format";
import Modal from "@/components/admin/Modal";
import ScheduleEditor from "@/components/admin/ScheduleEditor";
import { ListScroll, Pager, usePaged } from "@/components/admin/PagedList";
import {
  STATUS_DOT,
  STATUS_ROW,
  formatDuration,
  payModeLabel,
  payPeriodLabel,
  payUnitLabel,
  shortDate,
} from "@/lib/attendance";
import type {
  StaffPayMode,
  StaffPayPeriod,
  StaffRow,
  StaffSchedule,
  StaffRole,
} from "@/lib/types";

interface Draft {
  id: string;
  name: string;
  phone: string;
  username: string;
  password: string;
  position: string;
  roleId: string;
  branchId: string;
  isActive: boolean;
  canKitchen: boolean;
  canWaiter: boolean;
  canCashier: boolean;
  hasPin?: boolean;
  schedule: StaffSchedule[];
  payMode: StaffPayMode;
  hourlyRate: string;
  shiftRate: string;
  monthlyRate: string;
  payPeriod: StaffPayPeriod;
}

// A new employee starts on a plausible roster rather than an empty one: an
// empty roster means every day is a day off, which makes the attendance screen
// say nothing until somebody notices the warning. The times are meant to be
// changed — that is the point of the form — but the shape is right.
const defaultSchedule = (): StaffSchedule[] =>
  [0, 1, 2, 3, 4, 5, 6].map((day) => ({
    day,
    start: day === 0 ? "" : "09:00",
    end: day === 0 ? "" : "18:00",
    isOff: day === 0,
  }));

const emptyDraft = (branchId: string): Draft => ({
  id: "",
  name: "",
  phone: "",
  username: "",
  password: "",
  position: "",
  roleId: "",
  branchId,
  isActive: true,
  // ⚠️ Off for a new employee. Most staff are not cooks, and a permission
  // handed out by default is not a permission — which is the whole reason
  // this field exists.
  canKitchen: false,
  canWaiter: false,
  canCashier: false,
  schedule: defaultSchedule(),
  payMode: "monthly",
  hourlyRate: "0",
  shiftRate: "0",
  monthlyRate: "0",
  payPeriod: "monthly",
});

const inputCls =
  "mt-1 w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand";

export default function AdminStaffPage() {
  const t = useAdminT();
  const { lang } = useI18n();
  const { branch, brandBranches, multi } = useAdminScope();

  const [rows, setRows] = useState<StaffRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState("");
  const [draft, setDraft] = useState<Draft | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const dur = useCallback(
    (m: number) => formatDuration(m, t.staff.hoursShort, t.staff.minutesShort),
    [t],
  );

  const [roles, setRoles] = useState<StaffRole[]>([]);

  useEffect(() => {
    api
      .adminRoles()
      .then((d) => setRoles(d.roles))
      // Silent: a staff screen whose role list did not load still edits names
      // and rotas, and the select simply shows the current value.
      .catch(() => {});
  }, []);

  const load = useCallback(() => {
    api
      .adminStaff()
      .then(setRows)
      .catch(() => setRows([]))
      .finally(() => setLoading(false));
  }, []);

  useEffect(load, [load]);

  // Somebody clocks in while the manager is looking at the board.
  useEffect(() => {
    const timer = setInterval(load, 30000);
    return () => clearInterval(timer);
  }, [load]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return rows;
    return rows.filter((r) =>
      [r.name, r.username, r.position, r.phone]
        .join(" ")
        .toLowerCase()
        .includes(q),
    );
  }, [rows, query]);

  const paged = usePaged(filtered, 15);
  const onShift = rows.filter((r) => r.onShift).length;

  function startCreate() {
    setError(null);
    setDraft(emptyDraft(branch?.id ?? ""));
  }

  function startEdit(row: StaffRow) {
    setError(null);
    setDraft({
      id: row.id,
      name: row.name,
      phone: row.phone,
      username: row.username,
      password: "",
      position: row.position,
      roleId: row.roleId ?? "",
      branchId: row.branchId ?? "",
      isActive: row.isActive,
      canKitchen: row.canKitchen ?? false,
      canWaiter: row.canWaiter ?? false,
      canCashier: row.canCashier ?? false,
      hasPin: row.hasPin ?? false,
      schedule: row.schedule ?? [],
      payMode: row.payMode || "monthly",
      hourlyRate: String(row.hourlyRate ?? 0),
      shiftRate: String(row.shiftRate ?? 0),
      monthlyRate: String(row.monthlyRate ?? 0),
      payPeriod: row.payPeriod || "monthly",
    });
  }

  async function save() {
    if (!draft) return;
    if (!draft.name.trim() || !draft.username.trim()) {
      setError(t.staff.nameRequired);
      return;
    }
    if (!draft.id && draft.password.length < 5) {
      setError(t.staff.passwordShort);
      return;
    }
    setSaving(true);
    setError(null);
    const body = {
      name: draft.name.trim(),
      phone: draft.phone.trim(),
      username: draft.username.trim().toLowerCase(),
      position: draft.position.trim(),
      roleId: draft.roleId,
      branchId: draft.branchId,
      isActive: draft.isActive,
      canKitchen: draft.canKitchen,
      canWaiter: draft.canWaiter,
      canCashier: draft.canCashier,
      schedule: draft.schedule,
      payMode: draft.payMode,
      hourlyRate: Number(draft.hourlyRate) || 0,
      shiftRate: Number(draft.shiftRate) || 0,
      monthlyRate: Number(draft.monthlyRate) || 0,
      payPeriod: draft.payPeriod,
    };
    try {
      if (draft.id) {
        await api.updateStaff(draft.id, {
          ...body,
          ...(draft.password ? { password: draft.password } : {}),
        });
      } else {
        await api.createStaff({ ...body, password: draft.password });
      }
      setDraft(null);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  async function remove(row: StaffRow) {
    if (!window.confirm(t.staff.confirmDelete(row.name))) return;
    try {
      await api.deleteStaff(row.id);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    }
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="font-display text-2xl font-bold">{t.staff.title}</h1>
          <p className="mt-0.5 text-sm text-ink-muted">
            {t.staff.summary(rows.length, onShift)}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Link href="/admin/payroll" className="btn-ghost px-4 py-2 text-sm">
            {t.staff.payrollTitle}
          </Link>
          <button
            type="button"
            onClick={startCreate}
            className="btn-primary px-4 py-2 text-sm"
            data-help="add"
          >
            {t.staff.add}
          </button>
        </div>
      </div>

      <input
        className="w-full max-w-sm rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand"
        placeholder={t.staff.searchPh}
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />

      {error && <p className="text-sm text-red-600">{error}</p>}

      <div className="rounded-2xl border border-line bg-surface shadow-card">
        {loading ? (
          <p className="p-6 text-sm text-ink-muted">{t.common.loading}</p>
        ) : filtered.length === 0 ? (
          <p className="p-6 text-sm text-ink-muted">{t.staff.empty}</p>
        ) : (
          <>
            <ListScroll>
              <table className="w-full text-sm">
                <thead className="sticky top-0 bg-surface text-left text-xs uppercase tracking-wide text-ink-muted">
                  <tr className="border-b border-line">
                    <th className="px-4 py-3">{t.staff.colStaff}</th>
                    <th className="px-4 py-3">{t.staff.colToday}</th>
                    <th className="px-4 py-3">{t.staff.colPeriod}</th>
                    <th className="px-4 py-3 text-right">{t.staff.colDue}</th>
                    <th className="px-4 py-3" />
                  </tr>
                </thead>
                <tbody>
                  {paged.pageItems.map((row) => (
                    // The row wears today's state, in the calendar's own
                    // colours — see STATUS_ROW in lib/attendance.ts.
                    <tr
                      key={row.id}
                      className={`border-b border-line last:border-0 ${
                        STATUS_ROW[row.todayStatus]
                      }`}
                    >
                      <td className="px-4 py-3">
                        <Link
                          href={`/admin/staff/${row.id}`}
                          className="font-medium hover:text-brand"
                        >
                          {row.name}
                        </Link>
                        {!row.isActive && (
                          <span className="ml-1.5 text-xs text-ink-muted">
                            {t.staff.disabled}
                          </span>
                        )}
                        <p className="text-xs text-ink-muted">
                          {row.position || `@${row.username}`}
                          {row.phone && ` · ${formatUzPhone(row.phone)}`}
                          {multi && row.branchName && ` · ${row.branchName}`}
                        </p>
                      </td>

                      <td className="px-4 py-3">
                        <span className="inline-flex items-center gap-1.5">
                          <span
                            className={`h-2 w-2 shrink-0 rounded-full ${STATUS_DOT[row.todayStatus]}`}
                          />
                          <span className="tabular-nums">
                            {row.todayIn || "—"}
                            {row.todayIn && " → "}
                            {row.todayIn && (row.todayOut || t.staff.stillOpen)}
                          </span>
                        </span>
                        <p className="text-xs text-ink-muted">
                          {row.onShift
                            ? t.staff.onShiftNow
                            : t.staff.statuses[row.todayStatus]}
                          {row.todayWorked > 0 && ` · ${dur(row.todayWorked)}`}
                          {row.todayExpected > 0 &&
                            ` / ${dur(row.todayExpected)}`}
                        </p>
                      </td>

                      <td className="px-4 py-3">
                        <span className="tabular-nums">
                          {dur(row.totals.worked)}
                        </span>
                        <p className="text-xs text-ink-muted">
                          {t.staff.daysWorked}: {row.totals.days}
                          {row.totals.absent > 0 && (
                            <span className="text-red-600">
                              {" · "}
                              {t.staff.daysAbsent}: {row.totals.absent}
                            </span>
                          )}
                        </p>
                      </td>

                      <td className="px-4 py-3 text-right">
                        <span className="font-semibold tabular-nums">
                          {formatPrice(row.periodDue, "UZS", lang)}
                        </span>
                        <p className="text-xs text-ink-muted">
                          {shortDate(row.periodFrom)} —{" "}
                          {shortDate(row.periodTo)}
                        </p>
                      </td>

                      <td className="px-4 py-3 text-right whitespace-nowrap">
                        <button
                          type="button"
                          onClick={() => startEdit(row)}
                          className="text-xs text-brand hover:underline"
                        >
                          {t.common.edit}
                        </button>
                        <button
                          type="button"
                          onClick={() => remove(row)}
                          className="ml-3 text-xs text-red-600 hover:underline"
                        >
                          {t.common.delete}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </ListScroll>
            <Pager {...paged} onPage={paged.setPage} />
          </>
        )}
      </div>

      <p className="text-xs leading-relaxed text-ink-muted">
        {t.staff.installHint}
      </p>

      {draft && (
        <Modal onClose={() => setDraft(null)} wide>
          <h2 className="font-display text-lg font-bold">
            {draft.id ? t.staff.editTitle : t.staff.newTitle}
          </h2>

          <div className="mt-4 grid gap-3 sm:grid-cols-2">
            <label className="block text-sm">
              <span className="font-medium">{t.staff.name}</span>
              <input
                className={inputCls}
                value={draft.name}
                onChange={(e) => setDraft({ ...draft, name: e.target.value })}
              />
            </label>
            <label className="block text-sm">
              <span className="font-medium">{t.staff.role}</span>
              <select
                className={inputCls}
                value={draft.roleId}
                onChange={(e) => setDraft({ ...draft, roleId: e.target.value })}
              >
                <option value="">{t.staff.roleNone}</option>
                {roles.map((r) => (
                  <option key={r.id} value={r.id}>
                    {r.name}
                  </option>
                ))}
              </select>
              {/* ⚠️ The role is what the system reads; the free-text position
                  below it is a note and nothing checks it. Said here because
                  the two fields look alike and only one of them opens a till. */}
              <span className="mt-1 block text-xs text-ink-muted">
                {t.staff.roleHint}
              </span>
            </label>

            <label className="block text-sm">
              <span className="font-medium">{t.staff.position}</span>
              <input
                className={inputCls}
                placeholder={t.staff.positionPh}
                value={draft.position}
                onChange={(e) =>
                  setDraft({ ...draft, position: e.target.value })
                }
              />
            </label>
            <label className="block text-sm">
              <span className="font-medium">{t.staff.phone}</span>
              <input
                className={inputCls}
                value={draft.phone}
                onChange={(e) => setDraft({ ...draft, phone: e.target.value })}
              />
            </label>
            {multi && (
              <label className="block text-sm">
                <span className="font-medium">{t.staff.branch}</span>
                <select
                  className={inputCls}
                  value={draft.branchId}
                  onChange={(e) =>
                    setDraft({ ...draft, branchId: e.target.value })
                  }
                >
                  {brandBranches.map((b) => (
                    <option key={b.id} value={b.id}>
                      {b.name}
                    </option>
                  ))}
                </select>
              </label>
            )}
            <label className="block text-sm">
              <span className="font-medium">{t.staff.username}</span>
              <input
                className={inputCls}
                autoCapitalize="none"
                value={draft.username}
                onChange={(e) =>
                  setDraft({ ...draft, username: e.target.value })
                }
              />
            </label>
            <label className="block text-sm">
              <span className="font-medium">
                {t.staff.password}
                {draft.id && (
                  <span className="text-ink-muted">{t.staff.passwordKeep}</span>
                )}
              </span>
              <input
                className={inputCls}
                type="text"
                value={draft.password}
                onChange={(e) =>
                  setDraft({ ...draft, password: e.target.value })
                }
              />
            </label>
          </div>

          <div className="mt-4 border-t border-line pt-4">
            <ScheduleEditor
              value={draft.schedule}
              onChange={(schedule) => setDraft({ ...draft, schedule })}
            />
          </div>

          <div className="mt-4 border-t border-line pt-4">
            <p className="text-sm font-medium">{t.staff.payMode}</p>
            <div className="mt-2 grid gap-2 sm:grid-cols-3">
              {(
                [
                  ["hourly", t.staff.payHourly, t.staff.payHourlyHint],
                  ["shift", t.staff.payShift, t.staff.payShiftHint],
                  ["monthly", t.staff.payMonthly, t.staff.payMonthlyHint],
                ] as const
              ).map(([mode, label, hint]) => (
                <button
                  key={mode}
                  type="button"
                  onClick={() => setDraft({ ...draft, payMode: mode })}
                  className={`rounded-2xl border px-3 py-2 text-left transition-colors ${
                    draft.payMode === mode
                      ? "border-brand bg-brand/10"
                      : "border-line-strong hover:border-brand"
                  }`}
                >
                  <span className="block text-sm font-semibold">{label}</span>
                  <span className="mt-0.5 block text-[11px] text-ink-muted">
                    {hint}
                  </span>
                </button>
              ))}
            </div>

            <label className="mt-3 block text-sm">
              <span className="font-medium">
                {t.staff.rate} ({payUnitLabel(draft.payMode, t.staff)})
              </span>
              <input
                className={inputCls}
                inputMode="numeric"
                value={
                  draft.payMode === "hourly"
                    ? draft.hourlyRate
                    : draft.payMode === "shift"
                      ? draft.shiftRate
                      : draft.monthlyRate
                }
                onChange={(e) => {
                  const v = e.target.value.replace(/\D/g, "");
                  setDraft({
                    ...draft,
                    ...(draft.payMode === "hourly"
                      ? { hourlyRate: v }
                      : draft.payMode === "shift"
                        ? { shiftRate: v }
                        : { monthlyRate: v }),
                  });
                }}
              />
            </label>

            <label className="mt-3 block text-sm">
              <span className="font-medium">{t.staff.payPeriod}</span>
              <select
                className={inputCls}
                value={draft.payPeriod}
                onChange={(e) =>
                  setDraft({
                    ...draft,
                    payPeriod: e.target.value as StaffPayPeriod,
                  })
                }
              >
                <option value="daily">{t.staff.periodDaily}</option>
                <option value="10days">{t.staff.period10}</option>
                <option value="15days">{t.staff.period15}</option>
                <option value="monthly">{t.staff.periodMonthly}</option>
              </select>
            </label>
          </div>

          <label className="mt-4 flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={draft.isActive}
              onChange={(e) =>
                setDraft({ ...draft, isActive: e.target.checked })
              }
            />
            {t.staff.isActive}
          </label>

          {/* Who runs the pass. Separate from the job title above, which is
              free text this system never reads: "oshpaz" typed into a box
              cannot be a permission, and treating it as one would give access
              to whoever spelled it that way. */}
          <label className="mt-3 flex items-start gap-2 text-sm">
            <input
              type="checkbox"
              className="mt-1"
              checked={draft.canKitchen}
              onChange={(e) =>
                setDraft({ ...draft, canKitchen: e.target.checked })
              }
            />
            <span>
              {t.staff.canKitchen}
              <span className="mt-0.5 block text-xs text-ink-muted">
                {t.staff.canKitchenHint}
              </span>
            </span>
          </label>

          {/* The till, and why it is two boxes rather than one.
              The line is drawn where the money is: a waiter builds the check,
              a cashier settles it. Collapsing them would hand everybody who can
              carry a plate the ability to close a table as "discount 100%".

              ⚠️ Ticking cashier disables the floor box rather than hiding it —
              cashier implies waiter on the server, and a box that silently
              means something else is worse than one that explains itself. */}
          <label className="mt-3 flex items-start gap-2 text-sm">
            <input
              type="checkbox"
              className="mt-1"
              checked={draft.canWaiter || draft.canCashier}
              disabled={draft.canCashier}
              onChange={(e) =>
                setDraft({ ...draft, canWaiter: e.target.checked })
              }
            />
            <span>
              {t.staff.canWaiter}
              <span className="mt-0.5 block text-xs text-ink-muted">
                {t.staff.canWaiterHint}
              </span>
            </span>
          </label>

          <label className="mt-3 flex items-start gap-2 text-sm">
            <input
              type="checkbox"
              className="mt-1"
              checked={draft.canCashier}
              onChange={(e) =>
                setDraft({ ...draft, canCashier: e.target.checked })
              }
            />
            <span>
              {t.staff.canCashier}
              <span className="mt-0.5 block text-xs text-ink-muted">
                {t.staff.canCashierHint}
              </span>
            </span>
          </label>

          {/* ⚠️ Only for people who can actually reach a till, and only on an
              account that already exists — the PIN is saved by its own call,
              not with this form, so there is no record to attach it to until
              the employee has been created. */}
          {draft.id && (draft.canWaiter || draft.canCashier) && (
            <PinField
              staffId={draft.id}
              hasPin={draft.hasPin ?? false}
              onSaved={(has) =>
                setDraft((d) => (d ? { ...d, hasPin: has } : d))
              }
            />
          )}

          {error && <p className="mt-3 text-sm text-red-600">{error}</p>}

          <div className="mt-5 flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setDraft(null)}
              className="btn-ghost px-4 py-2 text-sm"
            >
              {t.common.cancel}
            </button>
            <button
              type="button"
              disabled={saving}
              onClick={save}
              className="btn-primary px-4 py-2 text-sm"
            >
              {saving ? t.common.saving : t.common.save}
            </button>
          </div>
        </Modal>
      )}
    </div>
  );
}

/** Setting an employee's till code.
 *
 * ⚠️ **Saved by its own call, deliberately not with the rest of the form.**
 * A form that does not display the PIN would send it empty on every save, so
 * editing a phone number would silently lock that person out of the till — the
 * same trap `branch.soldOut` and `kioskSecret` are kept out of their forms for.
 *
 * ⚠️ **The existing code is never shown, only whether one is set.** It is a
 * live credential: an admin screen that displayed it would be a place to read
 * other people's PINs, and the journal those PINs sign would stop meaning
 * anything.
 */
function PinField({
  staffId,
  hasPin,
  onSaved,
}: {
  staffId: string;
  hasPin: boolean;
  onSaved: (hasPin: boolean) => void;
}) {
  const t = useAdminT();
  const [pin, setPin] = useState("");
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");

  async function save(code: string) {
    setBusy(true);
    setErr("");
    setMsg("");
    try {
      const res = await api.setStaffPin(staffId, code);
      onSaved(res.hasPin);
      setPin("");
      setMsg(res.hasPin ? t.staff.pinSaved : t.staff.pinCleared);
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mt-4 rounded-xl border border-line p-3">
      <div className="text-sm font-medium">{t.staff.pin}</div>
      <p className="mt-1 text-xs text-ink-muted">{t.staff.pinHint}</p>
      <div className="mt-2 flex gap-2">
        <input
          className="input"
          inputMode="numeric"
          autoComplete="off"
          placeholder={hasPin ? t.staff.pinSetPh : t.staff.pinNewPh}
          value={pin}
          onChange={(e) =>
            setPin(e.target.value.replace(/\D/g, "").slice(0, 6))
          }
        />
        <button
          type="button"
          className="btn shrink-0"
          disabled={busy || pin.length !== 4}
          onClick={() => void save(pin)}
        >
          {t.common.save}
        </button>
        {hasPin && (
          <button
            type="button"
            className="btn-ghost shrink-0 px-3 text-sm"
            disabled={busy}
            onClick={() => void save("")}
          >
            {t.common.delete}
          </button>
        )}
      </div>
      {msg && <p className="mt-2 text-xs text-success">{msg}</p>}
      {err && <p className="mt-2 text-xs text-danger">{err}</p>}
    </div>
  );
}
