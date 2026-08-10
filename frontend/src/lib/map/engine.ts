// One map interface, three maps behind it.
//
// The site draws maps in four places — picking a delivery address, drawing the
// delivery zones, showing a courier, showing where an order goes — and until now
// each of those files was written against 2GIS directly, with a comment
// promising that switching providers meant "replace only this file". Four files
// is not one file, and the promise was never tested.
//
// ⚠️ **Restaurants do not all want the same map.** One already pays for a Yandex
// business account, one has a Google key from a previous site, one wants 2GIS
// because its Tashkent coverage is the best. That choice belongs to the
// restaurant, not to us — so it is a setting, and the setting can only exist if
// something below it can actually swap.
//
// So: this file is the contract, `twogis.ts` / `yandex.ts` / `google.ts` are the
// implementations, and the four components above know nothing except lat/lng.
// Everything provider-shaped lives behind `MapHandle` — including the two
// coordinate orders, which is exactly the kind of detail that is correct in
// three files and reversed in the fourth.

import type { MapProviderId } from "@/lib/map/config";

export interface LatLng {
  lat: number;
  lng: number;
}

/** A movable point. `remove` is safe to call twice — a React cleanup will. */
export interface MapPin {
  setPosition(p: LatLng): void;
  remove(): void;
}

export interface MapShape {
  remove(): void;
}

export interface ShapeStyle {
  /** Fill, as #rrggbbaa. Polygons only. */
  fill?: string;
  stroke: string;
  width: number;
}

export interface PinStyle {
  color?: string;
  /** Drawn beside the pin. Plain text — never markup, whatever the provider
   *  would allow: these are names typed in a panel. */
  label?: string;
}

export interface MapHandle {
  setCenter(p: LatLng, zoom?: number): void;
  /** Frame a box. `padding` is in pixels, on every side. */
  fitBounds(sw: LatLng, ne: LatLng, padding?: number): void;
  /** ⚠️ Called once per map. Every provider appends listeners rather than
   *  replacing them, so a second call would fire the handler twice — which on
   *  the zone editor means two vertices per click. */
  onClick(cb: (p: LatLng) => void): void;
  addPin(p: LatLng, style?: PinStyle): MapPin;
  /** A small filled circle: the draggable-looking handles on a zone's corners. */
  addVertex(p: LatLng, style: ShapeStyle & { radius: number }): MapShape;
  /** ⚠️ Never interactive. An interactive polygon swallows the map click, and
   *  the guest can then only drop a pin *outside* the delivery zone — which is
   *  the one place it is useless. Cost us a bug report the first time. */
  addPolygon(points: LatLng[], style: ShapeStyle): MapShape;
  addPolyline(points: LatLng[], style: ShapeStyle): MapShape;
  destroy(): void;
}

export interface CreateOptions {
  center: LatLng;
  zoom: number;
  key: string;
}

export interface MapEngine {
  id: MapProviderId;
  /** 2GIS MapGL is WebGL-only; the other two fall back to canvas. Asking the
   *  browser for a context it cannot give is worth a clear message rather than
   *  an empty grey box, but only where it is actually required. */
  requiresWebGL: boolean;
  create(container: HTMLElement, opts: CreateOptions): Promise<MapHandle>;
}

/** Loads a provider's <script> once per page, however many maps ask for it. */
const scripts = new Map<string, Promise<void>>();

export function loadScript(src: string): Promise<void> {
  const cached = scripts.get(src);
  if (cached) return cached;
  const p = new Promise<void>((resolve, reject) => {
    const el = document.createElement("script");
    el.src = src;
    el.async = true;
    el.onload = () => resolve();
    el.onerror = () => {
      // Forget it, so a retry button can actually retry rather than resolving
      // the same rejected promise for the rest of the session.
      scripts.delete(src);
      reject(new Error(`script failed: ${src}`));
    };
    document.head.appendChild(el);
  });
  scripts.set(src, p);
  return p;
}

export function boundsOf(points: LatLng[]): { sw: LatLng; ne: LatLng } | null {
  if (points.length === 0) return null;
  const lats = points.map((p) => p.lat);
  const lngs = points.map((p) => p.lng);
  return {
    sw: { lat: Math.min(...lats), lng: Math.min(...lngs) },
    ne: { lat: Math.max(...lats), lng: Math.max(...lngs) },
  };
}

/** #rrggbbaa → { color, opacity }, for the providers that take them apart.
 *  The stored zone colours carry their alpha in the string, and dropping it
 *  would paint the whole delivery area solid over the streets. */
export function splitAlpha(color: string): { hex: string; opacity: number } {
  if (/^#[0-9a-f]{8}$/i.test(color)) {
    return {
      hex: color.slice(0, 7),
      opacity: parseInt(color.slice(7), 16) / 255,
    };
  }
  return { hex: color, opacity: 1 };
}
