"use client";

// Booking a table from the site: pick the moment, tap a free table on the
// restaurant's own floor plan, leave a name and a phone.
//
// The shading tells the guest what is free *at the moment they chose*, and it
// re-fetches whenever that moment changes — but the answer that counts is the
// server's: it re-checks on submit and answers 409 if someone else got there
// first, which this page shows in place rather than as a dead end.
//
// Booking needs a phone confirmed by SMS, exactly as ordering does: a table
// held for a number nobody answers costs the restaurant a whole evening's
// seating. So the page sends guests through the same login, and the booking
// carries the number they proved they own.

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";
import { useI18n } from "@/lib/i18n/client";
import { useUser } from "@/lib/user";
import { formatDateTime, formatUzPhone } from "@/lib/format";
import FloorPlanView from "@/components/booking/FloorPlanView";
import { readBranchCookie, readBrandCookie } from "@/lib/siteBrand";
import type {
  BookingPlan,
  Branch,
  FloorTable,
  Reservation,
} from "@/lib/types";
import { useSiteWords } from "@/lib/business";

/** `yyyy-mm-dd` / `hh:mm` in the visitor's own timezone. */
function todayISO(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

function nextHalfHour(): string {
  const d = new Date(Date.now() + 30 * 60000);
  d.setMinutes(d.getMinutes() > 30 ? 60 : 30, 0, 0);
  return `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
}

export default function BookingPage() {
  const { t } = useI18n();
  const { user, loading: userLoading } = useUser();
  // The catalogue's own address — see lib/siteWords.ts.
  const w = useSiteWords();
  const router = useRouter();

  const [date, setDate] = useState(todayISO);
  const [time, setTime] = useState(nextHalfHour);
  const [guests, setGuests] = useState(2);
  const [plan, setPlan] = useState<BookingPlan | null>(null);
  const [loading, setLoading] = useState(true);
  const [picked, setPicked] = useState<FloorTable | null>(null);
  const [name, setName] = useState("");
  const [phone, setPhone] = useState("");
  const [comment, setComment] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [done, setDone] = useState<Reservation | null>(null);

  // The chosen moment as an ISO instant the API understands.
  const atISO = useMemo(() => {
    const d = new Date(`${date}T${time}`);
    return isNaN(d.getTime()) ? null : d.toISOString();
  }, [date, time]);

  useEffect(() => {
    if (user) {
      const full = [user.firstName, user.lastName].filter(Boolean).join(" ");
      if (full) setName((n) => n || full);
      // Display only: the server books against the confirmed number.
      if (user.phone) setPhone(formatUzPhone(user.phone));
    }
  }, [user]);

  // Which room is being booked. A company with several branches has several
  // floor plans, and table 7 in one is not table 7 in another; the guest picks
  // a branch first, or is already pinned to one by the QR they scanned.
  const [branchId, setBranchId] = useState(() => readBranchCookie());
  const [branches, setBranches] = useState<Branch[]>([]);
  useEffect(() => {
    api
      .getBrands()
      .then((d) => {
        const brand = readBrandCookie();
        const usable = d.branches.filter((b) => {
          if (!brand) return true;
          const owner = d.brands.find((x) => x.id === b.brandId);
          return owner?.id === brand || owner?.slug === brand;
        });
        setBranches(usable);
        setBranchId((cur) => cur || usable[0]?.id || "");
      })
      .catch(() => {
        /* one branch is the usual case — the picker stays hidden */
      });
  }, []);

  const load = useCallback(() => {
    if (!atISO) return;
    setLoading(true);
    api
      .bookingPlan(atISO, { branchId: branchId || undefined })
      .then(setPlan)
      .catch(() => setPlan(null))
      .finally(() => setLoading(false));
  }, [atISO, branchId]);

  useEffect(load, [load]);

  const booking = plan?.booking;
  // A branch whose room has never been drawn used to hand back `tables: null`
  // (Go marshals a nil slice as null, not []). The server no longer does, but
  // reading `.length` off whatever arrives is not something this page should
  // depend on being right.
  const tables = booking?.tables ?? [];
  const busyIds = useMemo(
    () => new Set((plan?.busy ?? []).map((b) => b.tableId)),
    [plan],
  );
  // When the picked table becomes busy (someone booked it while this page was
  // open, or the guest moved the time), say so instead of failing on submit.
  const pickedBusy = !!picked && busyIds.has(picked.id);

  async function submit() {
    if (!user) {
      router.push("/login?next=/bron&reason=booking");
      return;
    }
    // With the plan hidden there is nothing to pick and the server chooses,
    // so the guard applies only where the guest was actually asked.
    if (!picked && !booking?.hidePlan) {
      setError(t.booking.needTable);
      return;
    }
    if (!atISO) return;
    setBusy(true);
    setError(null);
    try {
      const res = await api.createReservation({
        // Empty when the restaurant hides its plan: the server assigns the
        // smallest free table that fits, and refuses with 409 if the room is
        // full at that moment — the same answer as tapping a table somebody
        // took a second earlier.
        tableId: picked?.id ?? "",
        branchId: branchId || undefined,
        at: atISO,
        guests,
        customer: { name, phone },
        comment,
      });
      setDone(res);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.booking.closed);
      // Someone else may have taken it — refresh the shading either way.
      load();
    } finally {
      setBusy(false);
    }
  }

  if (done) {
    return (
      <main className="container-page max-w-xl py-16 text-center">
        <div className="rounded-3xl border border-line bg-surface p-8 shadow-card">
          <p className="text-5xl" aria-hidden>
            ✅
          </p>
          <h1 className="mt-4 font-display text-2xl font-bold">
            {t.booking.successTitle}
          </h1>
          <p className="mt-3 text-ink-muted">
            {t.booking.successText(done.number)}
          </p>
          <p className="mt-2 text-sm text-ink-muted">
            {t.booking.tableLabel(done.tableNumber)} ·{" "}
            {formatDateTime(done.at)}
          </p>
          <div className="mt-6 flex flex-wrap justify-center gap-3">
            <button
              type="button"
              onClick={() => {
                setDone(null);
                setPicked(null);
                setComment("");
                load();
              }}
              className="btn-ghost px-5 py-2.5"
            >
              {t.booking.another}
            </button>
            <Link href={w.href} className="btn-primary px-5 py-2.5">
              {t.common.goToMenu}
            </Link>
          </div>
        </div>
      </main>
    );
  }

  return (
    <main className="container-page py-10">
      <h1 className="font-display text-3xl font-bold tracking-tight">
        {t.booking.title}
      </h1>
      <p className="mt-2 max-w-xl text-ink-muted">{t.booking.subtitle}</p>

      {!loading && booking && !booking.enabled && (
        <p className="mt-6 rounded-2xl bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
          {t.booking.closed}
        </p>
      )}

      <div className="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_360px]">
        {/* ---- when + plan ---- */}
        <section className="rounded-3xl border border-line bg-surface p-5 shadow-card sm:p-6">
          <h2 className="font-display text-lg font-bold">{t.booking.when}</h2>
          {/* Which room. Only drawn when there is more than one to choose from,
              so a single restaurant sees the page exactly as before. */}
          {branches.length > 1 && (
            <div className="mt-3 flex flex-wrap gap-2">
              {branches.map((b) => (
                <button
                  key={b.id}
                  type="button"
                  onClick={() => {
                    setBranchId(b.id);
                    // The plan changes, so a table picked in the other room is
                    // meaningless now.
                    setPicked(null);
                  }}
                  className={`rounded-full border px-4 py-2 text-sm font-semibold transition-colors ${
                    branchId === b.id
                      ? "border-brand bg-brand-tint text-brand-dark"
                      : "border-line-strong text-ink-soft hover:border-brand"
                  }`}
                >
                  {b.name}
                </button>
              ))}
            </div>
          )}
          <div className="mt-3 flex flex-wrap gap-3">
            <label className="block text-sm">
              <span className="font-medium">{t.booking.date}</span>
              <input
                type="date"
                className="input mt-1"
                value={date}
                min={todayISO()}
                onChange={(e) => setDate(e.target.value)}
              />
            </label>
            <label className="block text-sm">
              <span className="font-medium">{t.booking.time}</span>
              <input
                type="time"
                className="input mt-1"
                value={time}
                step={900}
                onChange={(e) => setTime(e.target.value)}
              />
            </label>
            <label className="block text-sm">
              <span className="font-medium">{t.booking.guests}</span>
              <input
                type="number"
                min={1}
                max={booking?.maxGuests ?? 20}
                className="input mt-1 w-28"
                value={guests}
                onChange={(e) => setGuests(Number(e.target.value) || 1)}
              />
            </label>
          </div>
          {booking && (
            <p className="mt-2 text-xs text-ink-muted">
              {t.booking.slotNote(booking.slotMinutes)}
            </p>
          )}

          {/* ⚠️ The whole table half disappears when the restaurant hides its
              plan — heading, map and legend together. Leaving the heading with
              nothing under it, or a disabled map, would tell the guest that
              something is broken; the honest reading of the setting is that
              choosing a table is not part of booking here. The server picks one
              (the smallest free table that fits), so the booking is as real as
              any other and the restaurant can move them. */}
          {!booking?.hidePlan && (
            <>
          <h2 className="mt-6 font-display text-lg font-bold">
            {t.booking.pickTable}
          </h2>
          {loading ? (
            <p className="mt-3 text-sm text-ink-muted/70">{t.common.loading}</p>
          ) : !booking || tables.length === 0 ? (
            <p className="mt-3 rounded-2xl border border-dashed border-line-strong p-6 text-center text-sm text-ink-muted/70">
              {t.booking.noPlan}
            </p>
          ) : (
            <>
              <div className="mt-3">
                <FloorPlanView
                  width={booking.width}
                  height={booking.height}
                  shapes={booking.shapes ?? []}
                  tables={tables}
                  busyIds={busyIds}
                  selectedId={picked?.id ?? null}
                  onSelect={(tb) => {
                    if (!tb.isActive) return;
                    setPicked(tb);
                    setError(null);
                  }}
                />
              </div>

              <div className="mt-3 flex flex-wrap gap-3 text-xs text-ink-muted">
                <Legend className="bg-emerald-500/20 ring-emerald-600/60">
                  {t.booking.legendFree}
                </Legend>
                <Legend className="bg-rose-500/25 ring-rose-500/70">
                  {t.booking.legendBusy}
                </Legend>
                <Legend className="bg-brand ring-brand">
                  {t.booking.legendPicked}
                </Legend>
                <Legend className="bg-ink/10 ring-ink/20">
                  {t.booking.legendOff}
                </Legend>
              </div>
            </>
          )}
            </>
          )}
        </section>

        {/* ---- guest details ---- */}
        <aside className="h-fit rounded-3xl border border-line bg-surface p-5 shadow-card sm:p-6">
          <h2 className="font-display text-lg font-bold">
            {t.booking.yourDetails}
          </h2>

          <p className="mt-3 rounded-2xl bg-ink/[0.03] px-4 py-3 text-sm">
            {booking?.hidePlan ? (
              // Says what will happen instead of naming a table, so the guest is
              // not left wondering which one they got. "We will seat you" is the
              // truthful version of a booking made without a plan.
              <span className="text-ink-muted">{t.booking.weWillSeat}</span>
            ) : picked ? (
              <>
                <span className="font-semibold">
                  {t.booking.tableLabel(picked.number)}
                </span>
                {picked.seats > 0 && ` · ${t.booking.seats(picked.seats)}`}
              </>
            ) : (
              <span className="text-ink-muted">{t.booking.needTable}</span>
            )}
          </p>

          {pickedBusy && (
            <p className="mt-2 rounded-2xl bg-rose-50 px-4 py-2.5 text-sm text-rose-700 dark:bg-rose-500/10 dark:text-rose-300">
              {t.booking.busyPick}
            </p>
          )}

          <label className="mt-4 block text-sm">
            <span className="font-medium">{t.booking.name}</span>
            <input
              className="input mt-1"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </label>
          <label className="mt-3 block text-sm">
            <span className="font-medium">{t.booking.phone}</span>
            <input
              className="input mt-1 disabled:opacity-70"
              placeholder="+998 90 123 45 67"
              value={phone}
              disabled={!!user}
              onChange={(e) => setPhone(e.target.value)}
            />
            {user && (
              <span className="mt-1 block text-xs text-ink-muted">
                {t.booking.phoneVerified}
              </span>
            )}
          </label>
          <label className="mt-3 block text-sm">
            <span className="font-medium">{t.booking.comment}</span>
            <input
              className="input mt-1"
              placeholder={t.booking.commentPh}
              maxLength={200}
              value={comment}
              onChange={(e) => setComment(e.target.value)}
            />
          </label>

          {error && (
            <p className="mt-3 rounded-xl bg-rose-50 px-3 py-2 text-xs text-rose-700 dark:bg-rose-500/10 dark:text-rose-300">
              {error}
            </p>
          )}

          {!userLoading && !user && (
            <p className="mt-4 rounded-2xl bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
              {t.login.bookingRequired}
            </p>
          )}

          <button
            type="button"
            disabled={busy || pickedBusy || (!!user && !picked && !booking?.hidePlan) || userLoading}
            onClick={submit}
            className="btn-primary mt-5 w-full px-6 py-3 disabled:opacity-60"
          >
            {busy
              ? t.booking.submitting
              : user
                ? t.booking.submit
                : t.booking.loginAndBook}
          </button>
        </aside>
      </div>
    </main>
  );
}

function Legend({
  className,
  children,
}: {
  className: string;
  children: React.ReactNode;
}) {
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className={`h-3 w-3 rounded ring-2 ${className}`} aria-hidden />
      {children}
    </span>
  );
}
