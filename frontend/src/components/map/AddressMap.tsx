"use client";

// Picking a point: the delivery address, or the restaurant shown for pickup.
//
// Provider-agnostic — 2GIS, Yandex or Google, whichever the restaurant chose in
// its settings (see lib/map). Everything here is lat/lng and nothing knows which
// map is underneath, which is what makes the setting possible at all.
//
// Cost note (CLAUDE.md §7): the map is loaded only where a point is picked, and
// the text address comes from a separate input — we place a pin rather than pay
// for geocoding.

import { useEffect, useRef, useState } from "react";
import { useI18n } from "@/lib/i18n/client";
import { useMapEngine } from "@/lib/map";
import type { MapPin, MapShape } from "@/lib/map/engine";
import { boundsOf } from "@/lib/map/engine";
import type { DeliveryZone } from "@/lib/types";

export interface LatLng {
  lat: number;
  lng: number;
}

interface AddressMapProps {
  value: LatLng | null;
  onChange?: (p: LatLng) => void;
  center: LatLng;
  className?: string;
  // Read-only: show a fixed marker (e.g. restaurant location for pickup),
  // no click-to-place.
  readOnly?: boolean;
  // Delivery zones drawn under the marker so the customer can see where we
  // deliver before dropping a pin. Purely informational — the fee still comes
  // from POST /delivery/quote.
  zones?: DeliveryZone[] | null;
}

const ZONE_FILL = "#e2590d33";
const ZONE_STROKE = "#e2590d";

export default function AddressMap({
  value,
  onChange,
  center,
  className,
  readOnly = false,
  zones,
}: AddressMapProps) {
  const { t } = useI18n();
  const containerRef = useRef<HTMLDivElement>(null);
  const [attempt, setAttempt] = useState(0);
  const { handle, failed, hasKey, setFailed } = useMapEngine({
    container: containerRef,
    center: value ?? center,
    zoom: 14,
    attempt,
  });

  const pinRef = useRef<MapPin | null>(null);
  const shapesRef = useRef<MapShape[]>([]);
  // The zone fit runs once per mounted map, never fighting the user's panning.
  const fittedRef = useRef(false);
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  // The pin, and the click that moves it.
  useEffect(() => {
    if (!handle) return;
    const pin = handle.addPin(value ?? center);
    pinRef.current = pin;
    if (!readOnly) {
      handle.onClick((p) => {
        pin.setPosition(p);
        onChangeRef.current?.(p);
      });
    }
    return () => {
      pin.remove();
      pinRef.current = null;
      fittedRef.current = false;
    };
    // Only when the map itself is (re)created: the pin follows `value` in its
    // own effect below, and rebuilding it per change would drop the click
    // listener with it.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [handle, readOnly]);

  // Zones, redrawn whenever they change.
  useEffect(() => {
    if (!handle) return;
    shapesRef.current.forEach((s) => s.remove());
    shapesRef.current = [];

    for (const zone of zones ?? []) {
      const points = (zone.polygon ?? []).map(([lat, lng]) => ({ lat, lng }));
      if (points.length < 3) continue;
      shapesRef.current.push(
        handle.addPolygon(points, {
          fill: ZONE_FILL,
          stroke: ZONE_STROKE,
          width: 2,
        }),
      );
    }

    // Zones can be far larger than the default viewport, so a customer who has
    // not dropped a pin yet would just see the middle of a polygon. Fit the
    // whole delivery area once, so "where do you deliver" is answered at a
    // glance; picking a point takes over the viewport from there.
    if (!fittedRef.current && !value) {
      const all = (zones ?? []).flatMap((z) =>
        (z.polygon ?? []).length >= 3
          ? z.polygon.map(([lat, lng]) => ({ lat, lng }))
          : [],
      );
      const box = boundsOf(all);
      if (box) {
        handle.fitBounds(box.sw, box.ne, 24);
        fittedRef.current = true;
      }
    }
    // `value` is intentionally not a dependency: fitting happens once, and a
    // picked point must not re-trigger it.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [zones, handle]);

  // Reflect external value changes onto the marker and recenter the map (e.g.
  // when the user picks an address suggestion).
  useEffect(() => {
    if (!value) return;
    pinRef.current?.setPosition(value);
    handle?.setCenter(value);
  }, [value, handle]);

  if (hasKey === false) {
    return <Fallback className={className}>{t.map.noKey}</Fallback>;
  }

  if (failed) {
    return (
      <Fallback className={className}>
        {failed === "webgl" ? <>{t.map.webgl}</> : <>{t.map.loadFailed}</>}
        <button
          type="button"
          onClick={() => {
            setFailed(null);
            setAttempt((a) => a + 1);
          }}
          className="mt-3 block rounded-xl border border-line-strong px-3 py-1.5 text-xs font-medium hover:bg-ink/5"
        >
          {t.map.retry}
        </button>
      </Fallback>
    );
  }

  return (
    <div
      ref={containerRef}
      className={`overflow-hidden rounded-xl border border-line ${className ?? ""}`}
    />
  );
}

function Fallback({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={`flex flex-col items-center justify-center rounded-xl border border-dashed border-line-strong bg-cream p-6 text-center text-sm text-ink-muted ${className ?? ""}`}
    >
      <p>{children}</p>
    </div>
  );
}
