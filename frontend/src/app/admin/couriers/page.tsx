"use client";

// Courier accounts are created by hand here — there is no self-signup. The
// list doubles as a live dispatch board: status, last seen, and a map with
// every courier that has reported a position.

import { useCallback, useEffect, useMemo, useState } from "react";
import { api, ApiError } from "@/lib/api";
import Link from "next/link";
import Modal from "@/components/admin/Modal";
import LiveMap, { type MapPoint } from "@/components/map/LiveMap";
import { timeAgo } from "@/lib/orderFlow";
import { COURIER_BADGE, COURIER_ROW } from "@/lib/orderStatus";
import { ListScroll, Pager, usePaged } from "@/components/admin/PagedList";
import { useAdminT } from "@/lib/i18n/admin";
import { useAsk } from "@/components/ui/Ask";
import type {
  Courier,
  CourierPayoutMode,
  CourierStatus,
  Restaurant,
  StaffPayPeriod,
} from "@/lib/types";

const VEHICLE_KEYS = ["", "moto", "car", "bike", "foot"] as const;
type VehicleKey = (typeof VEHICLE_KEYS)[number];

interface Draft {
  id: string;
  name: string;
  phone: string;
  username: string;
  password: string;
  vehicle: string;
  isActive: boolean;
  payoutMode: CourierPayoutMode;
  payoutPerOrder: string;
  payoutPercent: string;
  monthlyRate: string;
  payPeriod: StaffPayPeriod;
}

const emptyDraft = (): Draft => ({
  id: "",
  name: "",
  phone: "",
  username: "",
  password: "",
  vehicle: "moto",
  isActive: true,
  payoutMode: "deliveryFee",
  payoutPerOrder: "0",
  payoutPercent: "0",
  monthlyRate: "0",
  payPeriod: "monthly",
});

const inputCls =
  "mt-1 w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand";

export default function AdminCouriersPage() {
  const [couriers, setCouriers] = useState<Courier[]>([]);
  const { ask, tell } = useAsk();
  const [restaurant, setRestaurant] = useState<Restaurant | null>(null);
  const [loading, setLoading] = useState(true);
  const paged = usePaged(couriers, 20);
  const [draft, setDraft] = useState<Draft | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const t = useAdminT();

  const vehicleLabel = (v: string) =>
    v === "moto"
      ? t.couriers.vehicles.moto
      : v === "car"
        ? t.couriers.vehicles.car
        : v === "bike"
          ? t.couriers.vehicles.bike
          : v === "foot"
            ? t.couriers.vehicles.foot
            : t.couriers.vehicles.none;

  const load = useCallback(() => {
    api
      .adminCouriers()
      .then(setCouriers)
      .catch(() => setCouriers([]))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    load();
    api
      .getRestaurant()
      .then((r) => setRestaurant(r.restaurant))
      .catch(() => setRestaurant(null));
  }, [load]);

  // Positions move while the dispatcher watches — refresh them quietly.
  useEffect(() => {
    const t = setInterval(load, 15000);
    return () => clearInterval(t);
  }, [load]);

  const points = useMemo<MapPoint[]>(() => {
    const list: MapPoint[] = couriers
      .filter((c) => c.location && c.location.lat !== 0)
      .map((c) => ({
        id: c.id,
        lat: c.location!.lat,
        lng: c.location!.lng,
        label: c.name,
        kind: c.status === "off" ? "idle" : "courier",
      }));
    if (restaurant?.address?.lat) {
      list.push({
        id: "restaurant",
        lat: restaurant.address.lat,
        lng: restaurant.address.lng,
        label: restaurant.name || t.nav.panel,
        kind: "restaurant",
      });
    }
    return list;
  }, [couriers, restaurant, t.nav.panel]);

  async function save() {
    if (!draft) return;
    if (!draft.name.trim() || !draft.username.trim()) {
      setError(t.couriers.nameRequired);
      return;
    }
    if (!draft.id && draft.password.length < 5) {
      setError(t.couriers.passwordShort);
      return;
    }
    setSaving(true);
    setError(null);
    try {
      const body = {
        name: draft.name.trim(),
        phone: draft.phone.trim(),
        username: draft.username.trim().toLowerCase(),
        vehicle: draft.vehicle,
        isActive: draft.isActive,
        password: draft.password,
        payoutMode: draft.payoutMode,
        payoutPerOrder: Number(draft.payoutPerOrder) || 0,
        payoutPercent: Number(draft.payoutPercent) || 0,
        monthlyRate: Number(draft.monthlyRate) || 0,
        payPeriod: draft.payPeriod,
      };
      if (draft.id) {
        await api.updateCourier(draft.id, body);
      } else {
        await api.createCourier(body);
      }
      setDraft(null);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  async function remove(c: Courier) {
    if (!(await ask({ title: t.couriers.confirmDelete(c.name), danger: true })))
      return;
    try {
      await api.deleteCourier(c.id);
      load();
    } catch (e) {
      void tell({
        title: e instanceof ApiError ? e.message : t.common.deleteFailed,
      });
    }
  }

  const online = couriers.filter((c) => c.status !== "off").length;
  const center = restaurant?.address?.lat
    ? { lat: restaurant.address.lat, lng: restaurant.address.lng }
    : { lat: 41.311081, lng: 69.279737 };

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="font-display text-2xl font-bold">
            {t.couriers.title}
          </h1>
          <p className="mt-1 text-sm text-ink-muted">
            {loading
              ? t.common.loading
              : t.couriers.summary(couriers.length, online)}
          </p>
        </div>
        <button
          type="button"
          className="btn-primary px-4 py-2 text-sm"
          onClick={() => {
            setError(null);
            setDraft(emptyDraft());
          }}
        >
          {t.couriers.add}
        </button>
      </div>

      {/* Dispatch map */}
      <div className="mt-6">
        <LiveMap
          points={points}
          fallbackCenter={center}
          autoFit="once"
          className="h-[360px] w-full"
        />
        <p className="mt-2 text-xs text-ink-muted">{t.couriers.mapHint}</p>
      </div>

      {/* List */}
      <div className="mt-6 overflow-hidden rounded-3xl border border-line bg-surface shadow-card">
        {loading ? (
          <p className="py-10 text-center text-ink-muted/70">
            {t.common.loading}
          </p>
        ) : couriers.length === 0 ? (
          <p className="py-10 text-center text-ink-muted/70">
            {t.couriers.empty}
          </p>
        ) : (
          <ListScroll className="overflow-x-auto" max="max-h-[60vh]">
            <table className="w-full min-w-[760px] text-sm">
              <thead className="sticky top-0 z-10 border-b border-line bg-surface text-left text-xs uppercase tracking-wider text-ink-muted">
                <tr>
                  <th className="px-4 py-3 font-semibold">
                    {t.couriers.colCourier}
                  </th>
                  <th className="px-4 py-3 font-semibold">
                    {t.couriers.colLogin}
                  </th>
                  <th className="px-4 py-3 font-semibold">
                    {t.couriers.colStatus}
                  </th>
                  <th className="px-4 py-3 font-semibold">
                    {t.couriers.colLocation}
                  </th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody className="divide-y divide-line">
                {paged.pageItems.map((c) => (
                  // The row wears the rider's state, in the same vocabulary
                  // the orders board and the bookings use — see COURIER_ROW.
                  // A deactivated account keeps its fade on top of it.
                  <tr
                    key={c.id}
                    className={`${COURIER_ROW[c.status]} ${
                      c.isActive ? "" : "opacity-50"
                    }`}
                  >
                    <td className="px-4 py-3">
                      <Link
                        href={`/admin/couriers/${c.id}`}
                        className="font-medium hover:text-brand"
                      >
                        {c.name}
                      </Link>
                      <p className="text-xs text-ink-muted">
                        {c.phone ? (
                          <a
                            href={`tel:${c.phone}`}
                            className="hover:text-brand"
                          >
                            {c.phone}
                          </a>
                        ) : (
                          t.couriers.noPhone
                        )}
                        {c.vehicle && ` · ${vehicleLabel(c.vehicle)}`}
                      </p>
                    </td>
                    <td className="px-4 py-3 font-mono text-xs">
                      {c.username}
                    </td>
                    <td className="px-4 py-3">
                      <span className={`badge ${COURIER_BADGE[c.status]}`}>
                        {t.couriers.status[c.status]}
                      </span>
                      {!c.isActive && (
                        <span className="ml-2 text-xs text-ink-muted">
                          {t.couriers.disabled}
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-xs text-ink-muted">
                      {c.location?.at ? (
                        <>
                          {timeAgo(c.location.at, t.common.timeAgo)}
                          <span className="block font-mono">
                            {c.location.lat.toFixed(4)},{" "}
                            {c.location.lng.toFixed(4)}
                          </span>
                        </>
                      ) : (
                        t.couriers.noLocation
                      )}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <button
                        type="button"
                        className="text-xs text-ink-muted hover:text-brand"
                        onClick={() => {
                          setError(null);
                          setDraft({
                            id: c.id,
                            name: c.name,
                            phone: c.phone,
                            username: c.username,
                            password: "",
                            vehicle: c.vehicle,
                            isActive: c.isActive,
                            payoutMode: c.payoutMode ?? "deliveryFee",
                            payoutPerOrder: String(c.payoutPerOrder ?? 0),
                            payoutPercent: String(c.payoutPercent ?? 0),
                            monthlyRate: String(c.monthlyRate ?? 0),
                            payPeriod: c.payPeriod ?? "monthly",
                          });
                        }}
                      >
                        {t.common.edit}
                      </button>
                      <button
                        type="button"
                        className="ml-3 text-xs text-ink-muted hover:text-red-600"
                        onClick={() => remove(c)}
                      >
                        {t.common.delete}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </ListScroll>
        )}
        <Pager
          page={paged.page}
          pageCount={paged.pageCount}
          from={paged.from}
          to={paged.to}
          total={paged.total}
          onPage={paged.setPage}
        />
      </div>

      <p className="mt-4 text-xs text-ink-muted">{t.couriers.installHint}</p>

      {draft && (
        <Modal onClose={() => setDraft(null)}>
          <h3 className="mb-4 font-display text-lg font-bold">
            {draft.id ? t.couriers.editTitle : t.couriers.newTitle}
          </h3>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block text-sm">
              <span className="font-medium">{t.couriers.name}</span>
              <input
                className={inputCls}
                value={draft.name}
                onChange={(e) => setDraft({ ...draft, name: e.target.value })}
                autoFocus
              />
            </label>
            <label className="block text-sm">
              <span className="font-medium">{t.couriers.phone}</span>
              <input
                className={inputCls}
                value={draft.phone}
                placeholder="998901234567"
                onChange={(e) => setDraft({ ...draft, phone: e.target.value })}
              />
            </label>
            <label className="block text-sm">
              <span className="font-medium">{t.couriers.username}</span>
              <input
                className={inputCls}
                value={draft.username}
                onChange={(e) =>
                  setDraft({ ...draft, username: e.target.value })
                }
              />
            </label>
            <label className="block text-sm">
              <span className="font-medium">
                {t.couriers.password}
                {draft.id && t.couriers.passwordKeep}
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
            <label className="block text-sm">
              <span className="font-medium">{t.couriers.vehicle}</span>
              <select
                className={inputCls}
                value={draft.vehicle}
                onChange={(e) =>
                  setDraft({ ...draft, vehicle: e.target.value })
                }
              >
                {VEHICLE_KEYS.map((v: VehicleKey) => (
                  <option key={v} value={v}>
                    {vehicleLabel(v)}
                  </option>
                ))}
              </select>
            </label>
            {/* Payout rule — what the courier earns per delivery. */}
            <div className="sm:col-span-2">
              <span className="text-sm font-medium">{t.couriers.payout}</span>
              <div className="mt-2 grid gap-2 sm:grid-cols-3">
                {(
                  [
                    [
                      "deliveryFee",
                      t.couriers.payoutDeliveryFee,
                      t.couriers.payoutDeliveryFeeHint,
                    ],
                    [
                      "perOrder",
                      t.couriers.payoutPerOrder,
                      t.couriers.payoutPerOrderHint,
                    ],
                    [
                      "percent",
                      t.couriers.payoutPercent,
                      t.couriers.payoutPercentHint,
                    ],
                    // ⚠️ A salaried courier: the deliveries earn nothing on
                    // their own, because the wage is the wage. Paying both
                    // would pay twice, and the doubled figure would look
                    // exactly like a busy month.
                    [
                      "monthly",
                      t.couriers.payoutMonthly,
                      t.couriers.payoutMonthlyHint,
                    ],
                  ] as const
                ).map(([mode, label, hint]) => (
                  <button
                    key={mode}
                    type="button"
                    onClick={() => setDraft({ ...draft, payoutMode: mode })}
                    className={`rounded-2xl border px-3 py-2 text-left transition-colors ${
                      draft.payoutMode === mode
                        ? "border-brand bg-brand-tint/40"
                        : "border-line-strong hover:border-brand/60"
                    }`}
                  >
                    <span className="block text-sm font-semibold">{label}</span>
                    <span className="mt-0.5 block text-[11px] text-ink-muted">
                      {hint}
                    </span>
                  </button>
                ))}
              </div>

              {draft.payoutMode === "monthly" && (
                <label className="mt-3 block text-sm">
                  <span className="font-medium">{t.couriers.monthlyRate}</span>
                  <input
                    type="number"
                    className={inputCls}
                    value={draft.monthlyRate}
                    onChange={(e) =>
                      setDraft({ ...draft, monthlyRate: e.target.value })
                    }
                  />
                </label>
              )}

              {/* ⚠️ **Couriers were the only paid people with no pay period.**
                  Their money was thought of as per-delivery and therefore
                  continuous, so the screens could say what one had ever earned
                  — a figure that only grows — and never what was owed now. */}
              <label className="mt-3 block text-sm">
                <span className="font-medium">{t.couriers.payPeriod}</span>
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
                  <option value="monthly">{t.staff.periodMonthly}</option>
                  <option value="15days">{t.staff.period15}</option>
                  <option value="10days">{t.staff.period10}</option>
                  <option value="daily">{t.staff.periodDaily}</option>
                </select>
              </label>

              {draft.payoutMode === "perOrder" && (
                <label className="mt-3 block text-sm">
                  <span className="font-medium">{t.couriers.payoutAmount}</span>
                  <input
                    type="number"
                    className={inputCls}
                    value={draft.payoutPerOrder}
                    onChange={(e) =>
                      setDraft({ ...draft, payoutPerOrder: e.target.value })
                    }
                  />
                </label>
              )}
              {draft.payoutMode === "percent" && (
                <label className="mt-3 block text-sm">
                  <span className="font-medium">
                    {t.couriers.payoutPercentAmount}
                  </span>
                  <input
                    type="number"
                    className={inputCls}
                    value={draft.payoutPercent}
                    onChange={(e) =>
                      setDraft({ ...draft, payoutPercent: e.target.value })
                    }
                  />
                </label>
              )}
            </div>

            <label className="mt-7 flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={draft.isActive}
                onChange={(e) =>
                  setDraft({ ...draft, isActive: e.target.checked })
                }
              />
              <span>{t.couriers.isActive}</span>
            </label>
          </div>

          {error && <p className="mt-3 text-sm text-red-600">{error}</p>}

          <div className="mt-6 flex justify-end gap-3">
            <button
              type="button"
              className="btn-ghost px-4 py-2"
              onClick={() => setDraft(null)}
            >
              {t.common.cancel}
            </button>
            <button
              type="button"
              className="btn-primary px-5 py-2"
              disabled={saving}
              onClick={save}
            >
              {saving ? t.common.saving : t.common.save}
            </button>
          </div>
        </Modal>
      )}
    </div>
  );
}
