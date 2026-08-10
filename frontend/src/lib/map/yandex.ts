// Yandex Maps (JS API 2.1) behind the shared interface.
//
// For the restaurant that already has a Yandex business account, or whose guests
// navigate with Yandex Navigator and expect the same map on the site. No WebGL
// requirement, which also makes it the working answer on the old Android phones
// where 2GIS shows a grey box.
//
// ⚠️ **Yandex speaks [lat, lng]** — the same order everything here stores, and
// the reverse of 2GIS. That is the whole reason the flip lives in the engines
// rather than in the components.

import {
  loadScript,
  type CreateOptions,
  type LatLng,
  type MapEngine,
  type MapHandle,
  type MapPin,
  type MapShape,
  type PinStyle,
  type ShapeStyle,
} from "@/lib/map/engine";

// Only what is used, typed by hand: the published @types package covers a much
// larger surface and would have to be kept in the bundle for four calls.
interface YMapsGeoObject {
  geometry: { setCoordinates(c: number[]): void };
  events: { add(name: string, cb: (e: YMapsEvent) => void): void };
}
interface YMapsEvent {
  get(name: string): number[];
}
interface YMapsCollection {
  add(o: YMapsGeoObject): void;
  remove(o: YMapsGeoObject): void;
}
interface YMapsMap {
  geoObjects: YMapsCollection;
  events: { add(name: string, cb: (e: YMapsEvent) => void): void };
  setCenter(c: number[], zoom?: number): void;
  setBounds(b: number[][], o?: { checkZoomRange?: boolean; zoomMargin?: number }): void;
  destroy(): void;
}
interface YMaps {
  ready(cb: () => void): void;
  Map: new (
    el: HTMLElement,
    state: { center: number[]; zoom: number; controls: string[] },
    opts?: Record<string, unknown>,
  ) => YMapsMap;
  Placemark: new (
    coords: number[],
    props: Record<string, unknown>,
    opts: Record<string, unknown>,
  ) => YMapsGeoObject;
  Polygon: new (
    geometry: number[][][],
    props: Record<string, unknown>,
    opts: Record<string, unknown>,
  ) => YMapsGeoObject;
  Polyline: new (
    geometry: number[][],
    props: Record<string, unknown>,
    opts: Record<string, unknown>,
  ) => YMapsGeoObject;
  Circle: new (
    geometry: [number[], number],
    props: Record<string, unknown>,
    opts: Record<string, unknown>,
  ) => YMapsGeoObject;
}

declare global {
  interface Window {
    ymaps?: YMaps;
  }
}

function pair(p: LatLng): number[] {
  return [p.lat, p.lng];
}

async function ymapsReady(key: string): Promise<YMaps> {
  // `lang` decides the language of the map's own labels. Russian is the one
  // Yandex actually renders Uzbek cities in; `uz_UZ` is not offered at all.
  await loadScript(
    `https://api-maps.yandex.ru/2.1/?apikey=${encodeURIComponent(key)}&lang=ru_RU`,
  );
  const ymaps = window.ymaps;
  if (!ymaps) throw new Error("ymaps missing after load");
  // ⚠️ The script tag firing is not the API being usable: modules load after it,
  // and constructing a Map before `ready` throws.
  await new Promise<void>((resolve) => ymaps.ready(() => resolve()));
  return ymaps;
}

export const yandexEngine: MapEngine = {
  id: "yandex",
  requiresWebGL: false,
  async create(container: HTMLElement, opts: CreateOptions): Promise<MapHandle> {
    const ymaps = await ymapsReady(opts.key);
    const map = new ymaps.Map(
      container,
      {
        center: pair(opts.center),
        zoom: opts.zoom,
        // The default set includes a search box and a traffic panel, which on a
        // 170 px checkout map cover the thing being picked.
        controls: ["zoomControl", "geolocationControl"],
      },
      { suppressMapOpenBlock: true },
    );

    const detach = (o: YMapsGeoObject) => {
      try {
        map.geoObjects.remove(o);
      } catch {
        /* already gone */
      }
    };

    return {
      setCenter(p, zoom) {
        map.setCenter(pair(p), zoom);
      },
      fitBounds(sw, ne, padding = 24) {
        map.setBounds([pair(sw), pair(ne)], {
          checkZoomRange: true,
          zoomMargin: padding,
        });
      },
      onClick(cb) {
        map.events.add("click", (e) => {
          const [lat, lng] = e.get("coords");
          cb({ lat, lng });
        });
      },
      addPin(p: LatLng, style?: PinStyle): MapPin {
        const color = style?.color ?? "#e2590d";
        const placemark = new ymaps.Placemark(
          pair(p),
          // ⚠️ Plain text, and a caption rather than markup: `iconContent` is
          // rendered as HTML by Yandex, and these labels are names typed in a
          // panel. Nothing in the map layer is worth an injection.
          style?.label ? { iconCaption: style.label } : {},
          {
            preset: style?.label
              ? "islands#circleDotIconWithCaption"
              : "islands#circleDotIcon",
            iconColor: color,
          },
        );
        map.geoObjects.add(placemark);
        return {
          setPosition(next) {
            placemark.geometry.setCoordinates(pair(next));
          },
          remove: () => detach(placemark),
        };
      },
      addVertex(p, style): MapShape {
        const circle = new ymaps.Circle(
          // Yandex circles are metres on the ground, not pixels: a fixed radius
          // would be invisible at city zoom and cover a district when zoomed in.
          // 12 m reads as a handle at the zoom the zone editor works at.
          [pair(p), style.radius * 2],
          {},
          {
            fillColor: style.stroke,
            strokeColor: "#ffffff",
            strokeWidth: style.width,
            // Otherwise the handle eats the click that would add the next vertex.
            interactivityModel: "default#transparent",
          },
        );
        map.geoObjects.add(circle);
        return { remove: () => detach(circle) };
      },
      addPolygon(points, style: ShapeStyle): MapShape {
        const polygon = new ymaps.Polygon(
          [points.map(pair)],
          {},
          {
            fillColor: style.fill,
            strokeColor: style.stroke,
            strokeWidth: style.width,
            // See the note on MapHandle.addPolygon: an interactive polygon
            // swallows the click and the pin can only land outside the zone.
            interactivityModel: "default#transparent",
          },
        );
        map.geoObjects.add(polygon);
        return { remove: () => detach(polygon) };
      },
      addPolyline(points, style): MapShape {
        const line = new ymaps.Polyline(
          points.map(pair),
          {},
          {
            strokeColor: style.stroke,
            strokeWidth: style.width,
            interactivityModel: "default#transparent",
          },
        );
        map.geoObjects.add(line);
        return { remove: () => detach(line) };
      },
      destroy() {
        try {
          map.destroy();
        } catch {
          /* already gone */
        }
      },
    };
  },
};
