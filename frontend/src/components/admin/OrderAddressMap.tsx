"use client";

// The delivery point of one order, on a map, correctable from the panel.
//
// Customers drop their own pin at checkout and sometimes drop it on the wrong
// building. The pin is not decoration: the courier app refuses "Yetkazdim"
// until the courier stands within `arrivalRadiusM` of it, so a wrong pin blocks
// closing the order. Tapping the map moves it, and the address text follows
// from the same reverse geocoder the customer's checkout uses.
//
// The order's money is never touched here — see the backend handler. When the
// corrected point falls in a differently priced zone we say so instead, and the
// operator decides what to do about it.

import { useEffect, useState } from "react";
import { api, ApiError } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { reverseGeocode } from "@/lib/geocode";
import AddressMap, { type LatLng } from "@/components/map/AddressMap";
import type { Order, Restaurant } from "@/lib/types";

// Fallback centre (Tashkent) for an order that has no point at all yet.
const DEFAULT_CENTER: LatLng = { lat: 41.311081, lng: 69.279737 };

export default function OrderAddressMap({
  order,
  editable = false,
  onSaved,
}: {
  order: Order;
  // The orders section corrects addresses; the customer card only shows them.
  editable?: boolean;
  onSaved?: () => void;
}) {
  const t = useAdminT();
  const [restaurant, setRestaurant] = useState<Restaurant | null>(null);
  const [point, setPoint] = useState<LatLng | null>(
    order.address?.lat && order.address?.lng
      ? { lat: order.address.lat, lng: order.address.lng }
      : null,
  );
  const [text, setText] = useState(order.address?.text ?? "");
  const [dirty, setDirty] = useState(false);
  const [busy, setBusy] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [quote, setQuote] = useState<{
    fee: number;
    available: boolean;
  } | null>(null);

  useEffect(() => {
    api
      .getRestaurant()
      .then((r) => setRestaurant(r.restaurant))
      .catch(() => setRestaurant(null));
  }, []);

  // A fresh order replaces whatever was being edited (the list polls every 20s
  // and the operator may switch receipts).
  useEffect(() => {
    setPoint(
      order.address?.lat && order.address?.lng
        ? { lat: order.address.lat, lng: order.address.lng }
        : null,
    );
    setText(order.address?.text ?? "");
    setDirty(false);
    setSaved(false);
    setQuote(null);
    setError(null);
  }, [order.id, order.address?.lat, order.address?.lng, order.address?.text]);

  const zones = (restaurant?.delivery.zones ?? []).filter(
    (z) => (z.polygon?.length ?? 0) >= 3,
  );
  const currency = restaurant?.currency ?? "UZS";
  const center =
    point ??
    (restaurant?.address?.lat
      ? { lat: restaurant.address.lat, lng: restaurant.address.lng }
      : DEFAULT_CENTER);

  async function pick(p: LatLng) {
    setPoint(p);
    setDirty(true);
    setSaved(false);
    try {
      const found = await reverseGeocode(p.lat, p.lng);
      if (found) setText(found);
    } catch {
      /* reverse geocode is best-effort — the coordinates are what matter */
    }
  }

  async function save() {
    if (!point) return;
    setBusy(true);
    setError(null);
    try {
      const res = await api.updateOrderAddress(order.id, {
        lat: point.lat,
        lng: point.lng,
        text,
        comment: order.address?.comment ?? "",
      });
      setQuote({ fee: res.quotedFee, available: res.available });
      setDirty(false);
      setSaved(true);
      onSaved?.();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  function revert() {
    setPoint(
      order.address?.lat && order.address?.lng
        ? { lat: order.address.lat, lng: order.address.lng }
        : null,
    );
    setText(order.address?.text ?? "");
    setDirty(false);
    setError(null);
  }

  return (
    <div>
      <h3 className="font-semibold">{t.receipt.addressMapTitle}</h3>

      {editable && (
        <p className="mt-1 text-xs text-ink-muted">{t.receipt.addressEditHint}</p>
      )}

      <div className="mt-2">
        <AddressMap
          value={point}
          onChange={editable ? pick : undefined}
          center={center}
          className="h-64 w-full"
          readOnly={!editable}
          zones={zones}
        />
      </div>

      {!point && (
        <p className="mt-2 text-xs text-amber-700 dark:text-amber-300">
          {t.receipt.addressNoPoint}
        </p>
      )}

      {point && (
        <p className="mt-2 text-xs text-ink-muted">
          {t.receipt.addressPicked}: {point.lat.toFixed(5)},{" "}
          {point.lng.toFixed(5)}
        </p>
      )}

      {editable && (
        <>
          <input
            className="mt-2 w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand"
            value={text}
            onChange={(e) => {
              setText(e.target.value);
              setDirty(true);
              setSaved(false);
            }}
          />

          {error && <p className="mt-2 text-xs text-red-600">{error}</p>}

          {saved && (
            <p className="mt-2 text-xs font-medium text-emerald-600">
              ✓ {t.receipt.addressSaved}
            </p>
          )}

          {/* Said only after a save, when the corrected point prices
              differently from what the customer already agreed. */}
          {saved && quote && !quote.available && (
            <p className="mt-1 text-xs text-amber-700 dark:text-amber-300">
              {t.receipt.addressOutside}
            </p>
          )}
          {saved && quote && quote.available && quote.fee !== order.deliveryFee && (
            <p className="mt-1 text-xs text-amber-700 dark:text-amber-300">
              {t.receipt.addressFeeNote(
                formatPrice(quote.fee, currency),
                formatPrice(order.deliveryFee, currency),
              )}
            </p>
          )}

          {dirty && (
            <div className="mt-3 flex flex-wrap gap-2">
              <button
                type="button"
                disabled={busy || !point}
                onClick={save}
                className="btn-primary px-4 py-2 text-sm disabled:opacity-60"
              >
                {busy ? "..." : t.receipt.addressSave}
              </button>
              <button
                type="button"
                disabled={busy}
                onClick={revert}
                className="btn-ghost px-4 py-2 text-sm"
              >
                {t.receipt.addressCancel}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
