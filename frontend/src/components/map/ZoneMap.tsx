"use client";

// Drawing the delivery zones, on whichever map the restaurant chose.
//
// Provider-agnostic (see lib/map): polygons are [lat, lng] pairs, exactly as
// they are stored in `restaurant.delivery.zones[].polygon`, and the engine deals
// with whatever order its SDK wants.

import { useEffect, useRef, useState } from "react";
import { useI18n } from "@/lib/i18n/client";
import { useMapEngine } from "@/lib/map";
import type { MapShape } from "@/lib/map/engine";

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

const ACTIVE_FILL = "#e2590d55";
const ACTIVE_STROKE = "#e2590d";
const IDLE_FILL = "#6b728033";
const IDLE_STROKE = "#6b7280";

export default function ZoneMap({
  zones,
  activeIndex,
  onAddPoint,
  center,
  className,
}: ZoneMapProps) {
  const { t } = useI18n();
  const containerRef = useRef<HTMLDivElement>(null);
  const [attempt, setAttempt] = useState(0);
  const { handle, failed, hasKey, setFailed } = useMapEngine({
    container: containerRef,
    center,
    zoom: 11,
    attempt,
  });

  const shapesRef = useRef<MapShape[]>([]);
  const onAddPointRef = useRef(onAddPoint);
  onAddPointRef.current = onAddPoint;

  // The click that adds a vertex. Attached once per map: every provider appends
  // listeners rather than replacing them, so re-attaching would add two vertices
  // per click and the zone would double back on itself.
  useEffect(() => {
    if (!handle) return;
    handle.onClick((p) => onAddPointRef.current(p));
  }, [handle]);

  // Redraw all shapes whenever the zones or the selection change.
  useEffect(() => {
    if (!handle) return;
    shapesRef.current.forEach((s) => s.remove());
    shapesRef.current = [];

    zones.forEach((zone, i) => {
      const active = i === activeIndex;
      const points = zone.points.map(([lat, lng]) => ({ lat, lng }));
      if (points.length >= 3) {
        shapesRef.current.push(
          handle.addPolygon(points, {
            fill: active ? ACTIVE_FILL : IDLE_FILL,
            stroke: active ? ACTIVE_STROKE : IDLE_STROKE,
            width: active ? 3 : 2,
          }),
        );
      } else if (points.length === 2) {
        // Two points: show the segment, so a shape in progress is visible.
        shapesRef.current.push(
          handle.addPolyline(points, {
            stroke: active ? ACTIVE_STROKE : IDLE_STROKE,
            width: 3,
          }),
        );
      }

      // Vertex handles for the zone being edited.
      if (active) {
        for (const p of points) {
          shapesRef.current.push(
            handle.addVertex(p, { stroke: ACTIVE_STROKE, width: 2, radius: 6 }),
          );
        }
      }
    });
  }, [zones, activeIndex, handle]);

  if (hasKey === false) {
    return <Fallback className={className}>{t.map.noKeyZones}</Fallback>;
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
