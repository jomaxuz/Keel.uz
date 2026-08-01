"use client";

// Address input + 2GIS map kept in sync, used both at checkout and in the
// customer profile:
//
//   • typing → debounced suggestions; picking one moves the marker
//   • clicking/dragging on the map → reverse-geocode fills the text field
//
// The caller owns the value ({ text, lat, lng }) so it can be stored either in
// the checkout form or on the user profile.

import { useI18n } from "@/lib/i18n/client";
import { reverseGeocode } from "@/lib/geocode";
import { formatPrice } from "@/lib/format";
import AddressAutocomplete from "@/components/map/AddressAutocomplete";
import AddressMap, { type LatLng } from "@/components/map/AddressMap";
import type { DeliveryZone } from "@/lib/types";

export interface PickedAddress {
  text: string;
  lat: number;
  lng: number;
}

export default function AddressPicker({
  value,
  onChange,
  center,
  mapClassName = "h-64 w-full",
  inputClassName = "input mt-1",
  placeholder,
  zones,
  currency = "UZS",
}: {
  value: PickedAddress;
  onChange: (next: PickedAddress) => void;
  center: LatLng;
  mapClassName?: string;
  inputClassName?: string;
  placeholder?: string;
  // Delivery zones to draw on the map (and list under it) when configured.
  zones?: DeliveryZone[] | null;
  currency?: string;
}) {
  const { lang, t } = useI18n();
  // Only zones that are actually drawable count — same rule as the backend.
  const shownZones = (zones ?? []).filter((z) => (z.polygon?.length ?? 0) >= 3);
  const point: LatLng | null =
    value.lat && value.lng ? { lat: value.lat, lng: value.lng } : null;

  async function pickOnMap(p: LatLng) {
    // Show the coordinates immediately; the text follows once geocoded.
    onChange({ ...value, lat: p.lat, lng: p.lng });
    try {
      const text = await reverseGeocode(p.lat, p.lng);
      if (text) onChange({ text, lat: p.lat, lng: p.lng });
    } catch {
      /* reverse geocode is best-effort */
    }
  }

  return (
    <div className="space-y-3">
      <label className="block text-sm">
        <span className="font-medium">{t.checkout.address}</span>
        <AddressAutocomplete
          value={value.text}
          onTextChange={(text) => onChange({ ...value, text })}
          onSelect={(p) => onChange({ text: p.text, lat: p.lat, lng: p.lng })}
          placeholder={placeholder ?? t.checkout.addressPh}
          className={inputClassName}
        />
      </label>

      <div>
        <p className="mb-2 text-sm font-medium">{t.checkout.mapHint}</p>
        <AddressMap
          value={point}
          onChange={pickOnMap}
          center={center}
          className={mapClassName}
          zones={shownZones}
        />

        {shownZones.length > 0 && (
          <div className="mt-2 rounded-xl border border-line bg-ink/[0.02] p-3">
            <p className="text-xs font-semibold">{t.map.zonesTitle}</p>
            <ul className="mt-1.5 space-y-1">
              {shownZones.map((z, i) => (
                <li
                  key={i}
                  className="flex items-center gap-2 text-xs text-ink-muted"
                >
                  <span
                    aria-hidden
                    className="h-2.5 w-2.5 shrink-0 rounded-sm border border-brand bg-brand/25"
                  />
                  <span className="flex-1">{z.name}</span>
                  <span className="font-medium text-ink">
                    {z.pricing === "perKm"
                      ? `${formatPrice(z.baseFee ?? 0, currency, lang)} + ${formatPrice(z.perKm ?? 0, currency, lang)}/km`
                      : formatPrice(z.fee, currency, lang)}
                  </span>
                </li>
              ))}
            </ul>
            <p className="mt-2 text-xs text-ink-muted/70">{t.map.zonesHint}</p>
          </div>
        )}
        {point && (
          <p className="mt-2 text-xs text-ink-muted">
            {t.checkout.picked}: {point.lat.toFixed(5)}, {point.lng.toFixed(5)}
          </p>
        )}
      </div>
    </div>
  );
}
