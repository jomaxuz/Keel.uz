"use client";

// Table bookings: what is coming, who is sitting where, and the floor plan for
// the moment being looked at — the question a host actually asks is "is table 7
// free at eight?", which a list alone cannot answer.

import { useCallback, useEffect, useMemo, useState } from "react";
import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { formatTime, formatUzPhone } from "@/lib/format";
import { formatDateTime } from "@/lib/orderFlow";
import { RESERVATION_BADGE, RESERVATION_ROW } from "@/lib/orderStatus";
import { ListScroll, Pager, usePaged } from "@/components/admin/PagedList";
import FloorPlanView from "@/components/booking/FloorPlanView";
import type {
  BookingSettings,
  Reservation,
  ReservationStatus,
} from "@/lib/types";

type Scope = "upcoming" | "today" | "past" | "all";

const REFRESH_MS = 20000;

// 24-hour, from the shared formatter: `toLocaleTimeString` follows the device
// locale and would print "7:30 PM" on an English phone.
const hhmm = formatTime;

export default function AdminReservationsPage() {
  const t = useAdminT();
  const [rows, setRows] = useState<Reservation[]>([]);
  const [scope, setScope] = useState<Scope>("upcoming");
  const [q, setQ] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [booking, setBooking] = useState<BookingSettings | null>(null);
  // Which moment the plan is drawn for; defaults to now.
  const [planAt, setPlanAt] = useState<string | null>(null);

  const paged = usePaged(rows, 12);

  const load = useCallback(
    (opts?: { silent?: boolean }) => {
      if (!opts?.silent) setLoading(true);
      api
        .adminReservations({ scope, q: q.trim() || undefined })
        .then(setRows)
        .catch(() => setRows([]))
        .finally(() => setLoading(false));
    },
    [scope, q],
  );

  useEffect(() => {
    const id = setTimeout(() => load(), 250);
    return () => clearTimeout(id);
  }, [load]);

  useEffect(() => {
    const id = setInterval(() => load({ silent: true }), REFRESH_MS);
    return () => clearInterval(id);
  }, [load]);

  useEffect(() => {
    api
      .getRestaurant()
      .then((r) => setBooking(r.restaurant.booking ?? null))
      .catch(() => setBooking(null));
  }, []);

  // Tables held at the moment the plan is drawn for. Computed from the rows we
  // already have, so moving the time needs no extra request.
  const planMoment = planAt ? new Date(planAt) : new Date();
  const busyIds = useMemo(() => {
    const at = planMoment.getTime();
    const held = new Set<string>();
    for (const r of rows) {
      if (r.status === "cancelled" || r.status === "done") continue;
      if (new Date(r.at).getTime() <= at && new Date(r.endsAt).getTime() > at) {
        held.add(r.tableId);
      }
    }
    return held;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rows, planAt]);

  async function setStatus(res: Reservation, status: ReservationStatus) {
    let reason = "";
    if (status === "cancelled") {
      reason = window.prompt(t.booking.cancelPrompt) ?? "";
      if (!reason.trim()) return;
    }
    setBusyId(res.id);
    setError(null);
    try {
      await api.updateReservationStatus(res.id, status, reason);
      load({ silent: true });
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusyId(null);
    }
  }

  async function remove(res: Reservation) {
    if (!confirm(t.booking.removeConfirm)) return;
    setBusyId(res.id);
    try {
      await api.deleteReservation(res.id);
      load({ silent: true });
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.deleteFailed);
    } finally {
      setBusyId(null);
    }
  }

  const scopes: { key: Scope; label: string }[] = [
    { key: "upcoming", label: t.booking.scopeUpcoming },
    { key: "today", label: t.booking.scopeToday },
    { key: "past", label: t.booking.scopePast },
    { key: "all", label: t.booking.scopeAll },
  ];

  const statusLabel: Record<ReservationStatus, string> = {
    pending: t.booking.statusPending,
    confirmed: t.booking.statusConfirmed,
    seated: t.booking.statusSeated,
    done: t.booking.statusDone,
    cancelled: t.booking.statusCancelled,
  };

  return (
    <div>
      <h1 className="text-2xl font-bold">{t.booking.title}</h1>

      <div className="mt-4 flex flex-wrap items-center gap-2">
        {scopes.map((s) => (
          <button
            key={s.key}
            type="button"
            onClick={() => setScope(s.key)}
            className={`rounded-full px-3.5 py-1.5 text-sm font-semibold transition-colors ${
              scope === s.key
                ? "bg-brand text-white"
                : "border border-line bg-surface text-ink-soft hover:border-brand hover:text-brand"
            }`}
          >
            {s.label}
          </button>
        ))}
        <input
          className="input ml-auto max-w-xs"
          placeholder={t.booking.searchPh}
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </div>

      {error && <p className="mt-3 text-sm text-red-600">{error}</p>}

      <div className="mt-5 grid grid-cols-1 gap-6 xl:grid-cols-[minmax(0,1fr)_420px]">
        {/* ---- the bookings themselves ---- */}
        <div className="rounded-3xl border border-line bg-surface shadow-card">
          <ListScroll className="space-y-3 p-3" max="max-h-[70vh]">
            {loading ? (
              <p className="py-10 text-center text-ink-muted/70">
                {t.common.loading}
              </p>
            ) : rows.length === 0 ? (
              <p className="py-10 text-center text-ink-muted/70">
                {t.booking.empty}
              </p>
            ) : (
              paged.pageItems.map((r) => (
                // The booking wears its status, in the same vocabulary the
                // orders board uses — see RESERVATION_ROW.
                <article
                  key={r.id}
                  className={`rounded-2xl border p-4 ${RESERVATION_ROW[r.status]}`}
                >
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-semibold">#{r.number}</span>
                    <span className={`badge ${RESERVATION_BADGE[r.status]}`}>
                      {statusLabel[r.status]}
                    </span>
                    <span className="badge bg-ink/10 text-ink-soft">
                      {t.booking.tableLabel(r.tableNumber)}
                    </span>
                    <span className="ml-auto text-sm font-semibold tabular-nums">
                      {formatDateTime(r.at)}
                    </span>
                  </div>

                  <p className="mt-2 text-sm">
                    {r.customer.name} ·{" "}
                    <a
                      href={`tel:+${r.customer.phone.replace(/\D/g, "")}`}
                      className="text-brand hover:underline"
                    >
                      {formatUzPhone(r.customer.phone)}
                    </a>{" "}
                    · {t.booking.guestsCount(r.guests)}
                  </p>
                  <p className="text-xs text-ink-muted">
                    {t.booking.slotRange(hhmm(r.at), hhmm(r.endsAt))}
                  </p>
                  {r.comment && (
                    <p className="mt-1 text-xs italic text-amber-700 dark:text-amber-300">
                      “{r.comment}”
                    </p>
                  )}
                  {r.cancelReason && (
                    <p className="mt-1 text-xs text-rose-700 dark:text-rose-300">
                      {t.booking.cancel}: {r.cancelReason}
                    </p>
                  )}

                  <div className="mt-3 flex flex-wrap gap-2 text-xs">
                    {r.status === "pending" && (
                      <button
                        type="button"
                        disabled={busyId === r.id}
                        onClick={() => setStatus(r, "confirmed")}
                        className="btn-primary px-3 py-1.5"
                      >
                        {t.booking.confirm}
                      </button>
                    )}
                    {(r.status === "confirmed" || r.status === "pending") && (
                      <button
                        type="button"
                        disabled={busyId === r.id}
                        onClick={() => setStatus(r, "seated")}
                        className="btn-ghost px-3 py-1.5"
                      >
                        {t.booking.seat}
                      </button>
                    )}
                    {r.status === "seated" && (
                      <button
                        type="button"
                        disabled={busyId === r.id}
                        onClick={() => setStatus(r, "done")}
                        className="btn-primary px-3 py-1.5"
                      >
                        {t.booking.finish}
                      </button>
                    )}
                    {r.status !== "cancelled" && (
                      <button
                        type="button"
                        disabled={busyId === r.id}
                        onClick={() => setStatus(r, "cancelled")}
                        className="px-3 py-1.5 text-ink-muted hover:text-red-600"
                      >
                        {t.booking.cancel}
                      </button>
                    )}
                    <button
                      type="button"
                      disabled={busyId === r.id}
                      onClick={() => remove(r)}
                      className="ml-auto px-3 py-1.5 text-ink-muted/70 hover:text-red-600"
                    >
                      {t.booking.remove}
                    </button>
                  </div>
                </article>
              ))
            )}
          </ListScroll>
          <Pager
            page={paged.page}
            pageCount={paged.pageCount}
            from={paged.from}
            to={paged.to}
            total={paged.total}
            onPage={paged.setPage}
          />
        </div>

        {/* ---- the room, at the chosen moment ---- */}
        <aside className="h-fit rounded-3xl border border-line bg-surface p-4 shadow-card">
          <h2 className="text-sm font-semibold">{t.booking.planTitle}</h2>
          <input
            type="datetime-local"
            className="mt-2 w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand"
            value={
              planAt ??
              new Date(Date.now() - new Date().getTimezoneOffset() * 60000)
                .toISOString()
                .slice(0, 16)
            }
            onChange={(e) => setPlanAt(e.target.value)}
          />
          {booking && booking.tables?.length ? (
            <div className="mt-3">
              <FloorPlanView
                width={booking.width}
                height={booking.height}
                shapes={booking.shapes ?? []}
                tables={booking.tables}
                busyIds={busyIds}
              />
            </div>
          ) : (
            <p className="mt-3 rounded-2xl border border-dashed border-line-strong p-4 text-center text-sm text-ink-muted/70">
              {t.booking.planEmptyNotice}
            </p>
          )}
        </aside>
      </div>
    </div>
  );
}
