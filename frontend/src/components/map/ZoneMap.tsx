"use client";

// Swappable map module — 2GIS MapGL implementation for drawing delivery zones.
//
// Same rule as AddressMap: to switch providers later, replace ONLY this file.
// The props contract is provider-agnostic — polygons are [lat, lng] pairs,
// exactly as they are stored in `restaurant.delivery.zones[].polygon`.

import { useEffect, useRef, useState } from "react";
import { load } from "@2gis/mapgl";

export interface LatLng {
  lat: number;
  lng: number;
}

// One drawable zone. `points` are [lat, lng] pairs in drawing order.
export interface ZoneShape {
  name: string;
  points: [number, number][];
}

interface ZoneMapProps {
  zones: ZoneShape[];
  // Index of the zone being edited; clicks append a vertex to it.
  activeIndex: number;
  onAddPoint: (p: LatLng) => void;
  center: LatLng;
  className?: string;
}

const API_KEY = process.env.NEXT_PUBLIC_MAP_API_KEY ?? "";

const ACTIVE_FILL = "#e2590d55";
const ACTIVE_STROKE = "#e2590d";
const IDLE_FILL = "#6b728033";
const IDLE_STROKE = "#6b7280";

type Disposable = { destroy: () => void };

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

export default function ZoneMap({
  zones,
  activeIndex,
  onAddPoint,
  center,
  className,
}: ZoneMapProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<Disposable | null>(null);
  // MapGL has no "update shape" API, so every redraw destroys and recreates.
  const shapesRef = useRef<Disposable[]>([]);
  const mapglRef = useRef<Awaited<ReturnType<typeof load>> | null>(null);
  const onAddPointRef = useRef(onAddPoint);
  onAddPointRef.current = onAddPoint;

  const [ready, setReady] = useState(false);
  const [failed, setFailed] = useState<null | "webgl" | "load">(null);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    if (!API_KEY || !containerRef.current) return;
    if (!webglAvailable()) {
      setFailed("webgl");
      return;
    }

    let destroyed = false;
    const raf = requestAnimationFrame(() => {
      load()
        .then((mapgl) => {
          if (destroyed || !containerRef.current) return;
          try {
            const map = new mapgl.Map(containerRef.current, {
              center: [center.lng, center.lat],
              zoom: 11,
              key: API_KEY,
            });
            mapRef.current = map as unknown as Disposable;
            mapglRef.current = mapgl;
            map.on("click", (e) => {
              const [lng, lat] = e.lngLat;
              onAddPointRef.current({ lat, lng });
            });
            setReady(true);
          } catch (err) {
            console.error("[ZoneMap] map init failed:", err);
            setFailed("webgl");
          }
        })
        .catch((err) => {
          console.error("[ZoneMap] mapgl script load failed:", err);
          setFailed("load");
        });
    });

    return () => {
      destroyed = true;
      cancelAnimationFrame(raf);
      shapesRef.current.forEach((s) => {
        try {
          s.destroy();
        } catch {
          /* already gone */
        }
      });
      shapesRef.current = [];
      try {
        mapRef.current?.destroy();
      } catch {
        /* already gone */
      }
      mapRef.current = null;
      mapglRef.current = null;
      setReady(false);
    };
    // Re-run only on explicit retry.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [attempt]);

  // Redraw all shapes whenever the zones or the selection change.
  useEffect(() => {
    const mapgl = mapglRef.current;
    const map = mapRef.current;
    if (!ready || !mapgl || !map) return;

    shapesRef.current.forEach((s) => {
      try {
        s.destroy();
      } catch {
        /* already gone */
      }
    });
    shapesRef.current = [];

    zones.forEach((zone, i) => {
      const active = i === activeIndex;
      // 2GIS uses [lng, lat] and needs an explicitly closed ring.
      const ring = zone.points.map(([lat, lng]) => [lng, lat]);
      if (ring.length >= 3) {
        const polygon = new mapgl.Polygon(
          map as unknown as ConstructorParameters<typeof mapgl.Polygon>[0],
          {
            coordinates: [[...ring, ring[0]]],
            color: active ? ACTIVE_FILL : IDLE_FILL,
            strokeColor: active ? ACTIVE_STROKE : IDLE_STROKE,
            strokeWidth: active ? 3 : 2,
            // Otherwise the polygon eats the click and no vertex can be added
            // inside an already drawn zone.
            interactive: false,
          },
        );
        shapesRef.current.push(polygon as unknown as Disposable);
      } else if (ring.length === 2) {
        // Two points: show the segment so the shape in progress is visible.
        const line = new mapgl.Polyline(
          map as unknown as ConstructorParameters<typeof mapgl.Polyline>[0],
          {
            coordinates: ring,
            color: active ? ACTIVE_STROKE : IDLE_STROKE,
            width: 3,
          },
        );
        shapesRef.current.push(line as unknown as Disposable);
      }

      // Vertex handles for the zone being edited.
      if (active) {
        ring.forEach((coords) => {
          const marker = new mapgl.CircleMarker(
            map as unknown as ConstructorParameters<
              typeof mapgl.CircleMarker
            >[0],
            {
              coordinates: coords,
              radius: 6,
              color: ACTIVE_STROKE,
              strokeWidth: 2,
              strokeColor: "#ffffff",
            },
          );
          shapesRef.current.push(marker as unknown as Disposable);
        });
      }
    });
  }, [zones, activeIndex, ready]);

  if (!API_KEY) {
    return (
      <Fallback className={className}>
        Xarita uchun 2GIS API key kerak (NEXT_PUBLIC_MAP_API_KEY).
        Zonalarni koordinata bilan qo&apos;lda ham kiritsa bo&apos;ladi.
      </Fallback>
    );
  }

  if (failed) {
    return (
      <Fallback className={className}>
        {failed === "webgl"
          ? "Brauzer xaritani ko'rsata olmadi (WebGL yo'q)."
          : "Xarita yuklanmadi."}
        <button
          type="button"
          onClick={() => {
            setFailed(null);
            setAttempt((a) => a + 1);
          }}
          className="mt-3 block rounded-xl border border-line-strong px-3 py-1.5 text-xs font-medium hover:bg-ink/5"
        >
          Qayta urinish
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
