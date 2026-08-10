"use client";

// Read-only map showing a set of labelled points — couriers on the admin
// dashboard, or "restaurant + courier + destination" on the customer's order
// tracking page.
//
// Provider-agnostic (see lib/map): points are {lat, lng} and the engine deals
// with the SDK underneath.

import { useEffect, useRef, useState } from "react";
import { useI18n } from "@/lib/i18n/client";
import { useMapEngine } from "@/lib/map";
import { boundsOf, type MapPin } from "@/lib/map/engine";

export interface MapPoint {
  id: string;
  lat: number;
  lng: number;
  label: string;
  // Visual role — drives the marker colour.
  kind?: "courier" | "restaurant" | "destination" | "idle";
}

const COLORS: Record<string, string> = {
  courier: "#e2590d",
  restaurant: "#111827",
  destination: "#2563eb",
  idle: "#6b7280",
};

export default function LiveMap({
  points,
  className,
  fallbackCenter,
  // Re-fit the viewport whenever the points change (off by default so the
  // admin can pan around while positions refresh underneath).
  autoFit = "once",
}: {
  points: MapPoint[];
  className?: string;
  fallbackCenter: { lat: number; lng: number };
  autoFit?: "once" | "always" | "never";
}) {
  const { t } = useI18n();
  const containerRef = useRef<HTMLDivElement>(null);
  const { handle, failed, hasKey } = useMapEngine({
    container: containerRef,
    center: fallbackCenter,
    zoom: 12,
    attempt: 0,
  });
  const pinsRef = useRef<MapPin[]>([]);
  const fittedRef = useRef(false);

  // Redrawn on every update. A courier's position changes as often as the poll,
  // and a label that has to move with it is cheaper to rebuild than to reconcile
  // across three SDKs with three different ideas of a mutable marker.
  useEffect(() => {
    if (!handle) return;
    pinsRef.current.forEach((p) => p.remove());
    pinsRef.current = points.map((p) =>
      handle.addPin(p, { color: COLORS[p.kind ?? "courier"], label: p.label }),
    );

    const shouldFit =
      points.length > 0 &&
      (autoFit === "always" || (autoFit === "once" && !fittedRef.current));
    if (shouldFit) {
      fittedRef.current = true;
      if (points.length === 1) {
        handle.setCenter(points[0], 14);
      } else {
        const box = boundsOf(points);
        if (box) handle.fitBounds(box.sw, box.ne, 40);
      }
    }
  }, [points, handle, autoFit]);

  useEffect(() => {
    return () => {
      pinsRef.current = [];
      fittedRef.current = false;
    };
  }, []);

  if (hasKey === false || failed) {
    return (
      <div
        className={`flex items-center justify-center rounded-xl border border-dashed border-line-strong bg-cream p-6 text-center text-sm text-ink-muted ${className ?? ""}`}
      >
        <p>
          {hasKey === false
            ? t.map.noKey
            : failed === "webgl"
              ? t.map.webgl
              : t.map.loadFailed}
        </p>
      </div>
    );
  }

  return (
    <div
      ref={containerRef}
      className={`overflow-hidden rounded-xl border border-line ${className ?? ""}`}
    />
  );
}
