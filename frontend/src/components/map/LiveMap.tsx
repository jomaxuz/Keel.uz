"use client";

// Read-only map showing a set of labelled points — couriers on the admin
// dashboard, or "restaurant + courier + destination" on the customer's order
// tracking page. Same swappable-module rule as AddressMap: to change provider,
// replace only this file. Points are {lat, lng}; 2GIS wants [lng, lat].

import { useEffect, useRef, useState } from "react";
import { load } from "@2gis/mapgl";

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

const API_KEY = process.env.NEXT_PUBLIC_MAP_API_KEY ?? "";

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

function markerHtml(p: MapPoint): string {
  const color = COLORS[p.kind ?? "courier"];
  // Escaped by construction: labels are names typed in the admin panel.
  const label = p.label
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
  return `<div style="transform:translate(-50%,-100%);text-align:center;font:600 11px/1.2 system-ui,sans-serif;white-space:nowrap">
    <span style="display:inline-block;background:${color};color:#fff;padding:2px 7px;border-radius:999px;box-shadow:0 1px 4px rgba(0,0,0,.35)">${label}</span>
    <span style="display:block;width:10px;height:10px;margin:2px auto 0;background:${color};border:2px solid #fff;border-radius:50%;box-shadow:0 1px 3px rgba(0,0,0,.4)"></span>
  </div>`;
}

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
  const containerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<{
    destroy: () => void;
    setCenter: (c: number[]) => void;
    setZoom: (z: number) => void;
    fitBounds: (
      b: { southWest: number[]; northEast: number[] },
      o?: { padding?: Record<string, number> },
    ) => void;
  } | null>(null);
  const mapglRef = useRef<Awaited<ReturnType<typeof load>> | null>(null);
  const markersRef = useRef<Disposable[]>([]);
  const fittedRef = useRef(false);

  const [ready, setReady] = useState(false);
  const [failed, setFailed] = useState<null | "webgl" | "load">(null);

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
              center: [fallbackCenter.lng, fallbackCenter.lat],
              zoom: 12,
              key: API_KEY,
            });
            mapRef.current = map as unknown as typeof mapRef.current;
            mapglRef.current = mapgl;
            setReady(true);
          } catch {
            setFailed("webgl");
          }
        })
        .catch(() => setFailed("load"));
    });
    return () => {
      destroyed = true;
      cancelAnimationFrame(raf);
      markersRef.current.forEach((m) => {
        try {
          m.destroy();
        } catch {
          /* already gone */
        }
      });
      markersRef.current = [];
      try {
        mapRef.current?.destroy();
      } catch {
        /* already gone */
      }
      mapRef.current = null;
      mapglRef.current = null;
      fittedRef.current = false;
      setReady(false);
    };
    // Mount once; the map is not re-created when points change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Redraw markers on every update — MapGL has no "move marker html" API.
  useEffect(() => {
    const mapgl = mapglRef.current;
    const map = mapRef.current;
    if (!ready || !mapgl || !map) return;

    markersRef.current.forEach((m) => {
      try {
        m.destroy();
      } catch {
        /* already gone */
      }
    });
    markersRef.current = [];

    points.forEach((p) => {
      const marker = new mapgl.HtmlMarker(
        map as unknown as ConstructorParameters<typeof mapgl.HtmlMarker>[0],
        { coordinates: [p.lng, p.lat], html: markerHtml(p) },
      );
      markersRef.current.push(marker as unknown as Disposable);
    });

    const shouldFit =
      points.length > 0 &&
      (autoFit === "always" || (autoFit === "once" && !fittedRef.current));
    if (shouldFit) {
      fittedRef.current = true;
      if (points.length === 1) {
        map.setCenter([points[0].lng, points[0].lat]);
        map.setZoom(14);
      } else {
        const lats = points.map((p) => p.lat);
        const lngs = points.map((p) => p.lng);
        map.fitBounds(
          {
            southWest: [Math.min(...lngs), Math.min(...lats)],
            northEast: [Math.max(...lngs), Math.max(...lats)],
          },
          { padding: { top: 40, right: 40, bottom: 40, left: 40 } },
        );
      }
    }
  }, [points, ready, autoFit]);

  if (!API_KEY || failed) {
    return (
      <div
        className={`flex items-center justify-center rounded-xl border border-dashed border-line-strong bg-cream p-6 text-center text-sm text-ink-muted ${className ?? ""}`}
      >
        <p>
          {!API_KEY
            ? "Xarita uchun NEXT_PUBLIC_MAP_API_KEY sozlanmagan."
            : failed === "webgl"
              ? "Brauzer xaritani ko'rsata olmadi (WebGL yo'q)."
              : "Xarita yuklanmadi."}
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
