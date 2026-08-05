"use client";

// Swappable map module — 2GIS MapGL implementation.
//
// To switch providers (Yandex / Leaflet+OSM) later, replace ONLY this file:
// keep the same props contract { value, onChange, center, readOnly }. The rest
// of the app depends on lat/lng and never on the provider.
//
// Cost note (CLAUDE.md §7): the map is loaded only on checkout/settings; we pick
// lat/lng on the map and take the text address from a manual input — 2GIS
// geocoding is billed separately and we deliberately avoid it.

import { useEffect, useRef, useState } from "react";
import { useI18n } from "@/lib/i18n/client";
import { useMapKey } from "@/lib/mapKey";
import { load } from "@2gis/mapgl";
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

type FailKind = null | "webgl" | "load";

// Probe whether the browser can give us a WebGL context at all.
function webglAvailable(): boolean {
  try {
    const canvas = document.createElement("canvas");
    return !!(
      canvas.getContext("webgl") || canvas.getContext("experimental-webgl")
    );
  } catch {
    return false;
  }
}

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
  const mapRef = useRef<{
    destroy: () => void;
    setCenter: (c: number[]) => void;
    fitBounds: (
      b: { southWest: number[]; northEast: number[] },
      o?: { padding?: Record<string, number> },
    ) => void;
  } | null>(null);
  const markerRef = useRef<{ setCoordinates: (c: number[]) => void } | null>(
    null,
  );
  // MapGL has no "update shape" API, so zone polygons are recreated on change.
  const mapglRef = useRef<Awaited<ReturnType<typeof load>> | null>(null);
  const zoneShapesRef = useRef<{ destroy: () => void }[]>([]);
  // The zone fit runs once per mounted map, never fighting the user's panning.
  const fittedRef = useRef(false);
  // Fetched at run time from the restaurant profile: each restaurant
  // brings its own 2GIS key, so it cannot be baked into a build that
  // serves every tenant.
  const API_KEY = useMapKey();
  const [ready, setReady] = useState(false);
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  const [failed, setFailed] = useState<FailKind>(null);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    if (!API_KEY || !containerRef.current) return;

    if (!webglAvailable()) {
      setFailed("webgl");
      return;
    }

    let destroyed = false;
    let raf = 0;

    // 2GIS uses [lng, lat] coordinate order.
    const start: [number, number] = value
      ? [value.lng, value.lat]
      : [center.lng, center.lat];

    // Defer one frame so the container is laid out before MapGL sizes its canvas.
    raf = requestAnimationFrame(() => {
      load()
        .then((mapgl) => {
          if (destroyed || !containerRef.current) return;
          try {
            const map = new mapgl.Map(containerRef.current, {
              center: start,
              zoom: 14,
              key: API_KEY,
            });
            mapRef.current = map as unknown as typeof mapRef.current;
            mapglRef.current = mapgl;

            const marker = new mapgl.Marker(map, { coordinates: start });
            markerRef.current = marker as unknown as {
              setCoordinates: (c: number[]) => void;
            };

            if (!readOnly) {
              map.on("click", (e) => {
                const [lng, lat] = e.lngLat;
                marker.setCoordinates([lng, lat]);
                onChangeRef.current?.({ lat, lng });
              });
            }
            setReady(true);
          } catch (err) {
            console.error("[AddressMap] map init failed:", err);
            setFailed("webgl");
          }
        })
        .catch((err) => {
          console.error("[AddressMap] mapgl script load failed:", err);
          setFailed("load");
        });
    });

    return () => {
      destroyed = true;
      cancelAnimationFrame(raf);
      zoneShapesRef.current.forEach((z) => {
        try {
          z.destroy();
        } catch {
          /* already gone */
        }
      });
      zoneShapesRef.current = [];
      try {
        mapRef.current?.destroy();
      } catch {
        /* already gone */
      }
      mapRef.current = null;
      markerRef.current = null;
      mapglRef.current = null;
      fittedRef.current = false;
      setReady(false);
    };
    // Re-run only on explicit retry.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [attempt, API_KEY]);

  // Draw the delivery zones. Polygons are stored as [lat, lng]; 2GIS wants
  // [lng, lat] and an explicitly closed ring.
  useEffect(() => {
    const mapgl = mapglRef.current;
    const map = mapRef.current;
    if (!ready || !mapgl || !map) return;

    zoneShapesRef.current.forEach((z) => {
      try {
        z.destroy();
      } catch {
        /* already gone */
      }
    });
    zoneShapesRef.current = [];

    (zones ?? []).forEach((zone) => {
      const ring = (zone.polygon ?? []).map(([lat, lng]) => [lng, lat]);
      if (ring.length < 3) return;
      const polygon = new mapgl.Polygon(
        map as unknown as ConstructorParameters<typeof mapgl.Polygon>[0],
        {
          coordinates: [[...ring, ring[0]]],
          color: ZONE_FILL,
          strokeColor: ZONE_STROKE,
          strokeWidth: 2,
          // MapGL polygons are interactive by default and would swallow the
          // map click — the customer could then only drop a pin OUTSIDE the
          // delivery zone.
          interactive: false,
        },
      );
      zoneShapesRef.current.push(polygon as unknown as { destroy: () => void });
    });

    // Zones can be far larger than the default viewport, so a customer who has
    // not dropped a pin yet would just see the middle of a polygon. Fit the
    // whole delivery area once, so "where do you deliver" is answered at a
    // glance; picking a point takes over the viewport from there.
    if (!fittedRef.current && !value) {
      const all = (zones ?? []).flatMap((z) =>
        (z.polygon ?? []).length >= 3 ? z.polygon : [],
      );
      if (all.length >= 3) {
        const lats = all.map(([lat]) => lat);
        const lngs = all.map(([, lng]) => lng);
        mapRef.current?.fitBounds(
          {
            southWest: [Math.min(...lngs), Math.min(...lats)],
            northEast: [Math.max(...lngs), Math.max(...lats)],
          },
          { padding: { top: 24, right: 24, bottom: 24, left: 24 } },
        );
        fittedRef.current = true;
      }
    }
    // `value` is intentionally not a dependency: fitting happens once, and a
    // picked point must not re-trigger it.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [zones, ready]);

  // Reflect external value changes onto the marker and recenter the map (e.g.
  // when the user picks an address suggestion).
  useEffect(() => {
    if (value) {
      markerRef.current?.setCoordinates([value.lng, value.lat]);
      mapRef.current?.setCenter([value.lng, value.lat]);
    }
  }, [value]);

  if (API_KEY === "") {
    return (
      <Fallback className={className}>{t.map.noKey}</Fallback>
    );
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
