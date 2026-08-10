// Google Maps (JS API) behind the shared interface.
//
// The one an owner usually already has a key for, from whoever built their
// previous site, and the map their foreign guests read without thinking. Also
// the only one of the three that bills per map load, which is worth knowing
// before it is switched on — the settings page says so.
//
// ⚠️ **Google speaks {lat, lng}** objects, which is the same order everything
// here stores. Nothing to flip, and that is exactly why the flip belongs in the
// engines: two of three providers agree, and the third silently disagreeing is
// how a marker ends up in the sea.

import {
  loadScript,
  splitAlpha,
  type CreateOptions,
  type LatLng,
  type MapEngine,
  type MapHandle,
  type MapPin,
  type MapShape,
  type PinStyle,
  type ShapeStyle,
} from "@/lib/map/engine";

// Only what is used, typed by hand rather than pulling in @types/google.maps for
// six calls.
interface GLatLng {
  lat(): number;
  lng(): number;
}
interface GMapMouseEvent {
  latLng: GLatLng | null;
}
interface GRemovable {
  setMap(map: GMap | null): void;
}
interface GMarker extends GRemovable {
  setPosition(p: LatLng): void;
}
interface GBounds {
  extend(p: LatLng): void;
}
interface GMap {
  setCenter(p: LatLng): void;
  setZoom(z: number): void;
  fitBounds(b: GBounds, padding?: number): void;
  addListener(name: string, cb: (e: GMapMouseEvent) => void): void;
}
interface GoogleMaps {
  Map: new (el: HTMLElement, opts: Record<string, unknown>) => GMap;
  Marker: new (opts: Record<string, unknown>) => GMarker;
  Polygon: new (opts: Record<string, unknown>) => GRemovable;
  Polyline: new (opts: Record<string, unknown>) => GRemovable;
  Circle: new (opts: Record<string, unknown>) => GRemovable;
  LatLngBounds: new () => GBounds;
  SymbolPath: { CIRCLE: number };
}

declare global {
  interface Window {
    google?: { maps?: GoogleMaps };
  }
}

async function googleMaps(key: string): Promise<GoogleMaps> {
  await loadScript(
    `https://maps.googleapis.com/maps/api/js?key=${encodeURIComponent(key)}&v=weekly`,
  );
  const maps = window.google?.maps;
  if (!maps) throw new Error("google.maps missing after load");
  return maps;
}

export const googleEngine: MapEngine = {
  id: "google",
  requiresWebGL: false,
  async create(container: HTMLElement, opts: CreateOptions): Promise<MapHandle> {
    const maps = await googleMaps(opts.key);
    const map = new maps.Map(container, {
      center: opts.center,
      zoom: opts.zoom,
      // The default UI puts Street View, map-type and full-screen buttons over a
      // map that is often 170 px tall on a checkout page.
      streetViewControl: false,
      mapTypeControl: false,
      fullscreenControl: false,
      // Google's own clickable POIs open an info window over the map, which on
      // the address picker steals the click that was placing the pin.
      clickableIcons: false,
    });

    const detach = (o: GRemovable) => {
      try {
        o.setMap(null);
      } catch {
        /* already gone */
      }
    };

    return {
      setCenter(p, zoom) {
        map.setCenter(p);
        if (zoom != null) map.setZoom(zoom);
      },
      fitBounds(sw, ne, padding = 24) {
        const bounds = new maps.LatLngBounds();
        bounds.extend(sw);
        bounds.extend(ne);
        map.fitBounds(bounds, padding);
      },
      onClick(cb) {
        map.addListener("click", (e) => {
          if (!e.latLng) return;
          cb({ lat: e.latLng.lat(), lng: e.latLng.lng() });
        });
      },
      addPin(p: LatLng, style?: PinStyle): MapPin {
        const color = style?.color ?? "#e2590d";
        const marker = new maps.Marker({
          position: p,
          map,
          // ⚠️ `label` is set as text, never as HTML: Google renders it as a
          // string, and these labels are names typed in a panel.
          label: style?.label
            ? { text: style.label, color: "#ffffff", fontSize: "11px", fontWeight: "600" }
            : undefined,
          icon: style?.label
            ? {
                path: maps.SymbolPath.CIRCLE,
                scale: 13,
                fillColor: color,
                fillOpacity: 1,
                strokeColor: "#ffffff",
                strokeWeight: 2,
              }
            : undefined,
        });
        return {
          setPosition(next) {
            marker.setPosition(next);
          },
          remove: () => detach(marker),
        };
      },
      addVertex(p, style): MapShape {
        const marker = new maps.Marker({
          position: p,
          map,
          clickable: false,
          icon: {
            path: maps.SymbolPath.CIRCLE,
            scale: style.radius,
            fillColor: style.stroke,
            fillOpacity: 1,
            strokeColor: "#ffffff",
            strokeWeight: style.width,
          },
        });
        return { remove: () => detach(marker) };
      },
      addPolygon(points, style: ShapeStyle): MapShape {
        // Google takes colour and opacity apart; the stored zone colours carry
        // their alpha inside the string.
        const fill = splitAlpha(style.fill ?? "#00000000");
        const stroke = splitAlpha(style.stroke);
        const polygon = new maps.Polygon({
          paths: points,
          map,
          fillColor: fill.hex,
          fillOpacity: fill.opacity,
          strokeColor: stroke.hex,
          strokeOpacity: stroke.opacity,
          strokeWeight: style.width,
          // See MapHandle.addPolygon: a clickable polygon eats the map click and
          // the pin can then only be dropped outside the delivery zone.
          clickable: false,
        });
        return { remove: () => detach(polygon) };
      },
      addPolyline(points, style): MapShape {
        const stroke = splitAlpha(style.stroke);
        const line = new maps.Polyline({
          path: points,
          map,
          strokeColor: stroke.hex,
          strokeOpacity: stroke.opacity,
          strokeWeight: style.width,
          clickable: false,
        });
        return { remove: () => detach(line) };
      },
      destroy() {
        // Google has no map.destroy(): the map dies with its container, which
        // React removes. Listeners and overlays go with it.
        container.innerHTML = "";
      },
    };
  },
};
