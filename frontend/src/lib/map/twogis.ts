// 2GIS MapGL behind the shared interface.
//
// The default, and the one every existing install runs: best coverage of
// Tashkent and Samarqand, and the library itself is free without a request quota
// to watch. Its one demand is WebGL, which is why `requiresWebGL` exists at all.
//
// ⚠️ **MapGL speaks [lng, lat]**, the reverse of everything stored here (zone
// polygons are [lat, lng], as the delivery quote reads them). The flip happens
// here and nowhere else — a swap that is right in three files and reversed in
// the fourth puts a restaurant in the Aral Sea, and looks like a data problem.

import { load } from "@2gis/mapgl";
import type {
  CreateOptions,
  LatLng,
  MapEngine,
  MapHandle,
  MapPin,
  MapShape,
  PinStyle,
  ShapeStyle,
} from "@/lib/map/engine";

type Mapgl = Awaited<ReturnType<typeof load>>;
type Disposable = { destroy: () => void };

function ring(points: LatLng[]): number[][] {
  return points.map((p) => [p.lng, p.lat]);
}

function labelHtml(label: string, color: string): string {
  // Escaped by construction: labels are names typed in a panel, and this is the
  // one place in the map layer where text becomes markup.
  const safe = label
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
  return `<div style="transform:translate(-50%,-100%);text-align:center;font:600 11px/1.2 system-ui,sans-serif;white-space:nowrap">
    <span style="display:inline-block;background:${color};color:#fff;padding:2px 7px;border-radius:999px;box-shadow:0 1px 4px rgba(0,0,0,.35)">${safe}</span>
    <span style="display:block;width:10px;height:10px;margin:2px auto 0;background:${color};border:2px solid #fff;border-radius:50%;box-shadow:0 1px 3px rgba(0,0,0,.4)"></span>
  </div>`;
}

function safeDestroy(d: Disposable | null) {
  try {
    d?.destroy();
  } catch {
    /* already gone */
  }
}

export const twogisEngine: MapEngine = {
  id: "2gis",
  requiresWebGL: true,
  async create(container: HTMLElement, opts: CreateOptions): Promise<MapHandle> {
    const mapgl: Mapgl = await load();
    const map = new mapgl.Map(container, {
      center: [opts.center.lng, opts.center.lat],
      zoom: opts.zoom,
      key: opts.key,
    });

    return {
      setCenter(p, zoom) {
        map.setCenter([p.lng, p.lat]);
        if (zoom != null) map.setZoom(zoom);
      },
      fitBounds(sw, ne, padding = 24) {
        map.fitBounds(
          {
            southWest: [sw.lng, sw.lat],
            northEast: [ne.lng, ne.lat],
          },
          { padding: { top: padding, right: padding, bottom: padding, left: padding } },
        );
      },
      onClick(cb) {
        map.on("click", (e) => {
          const [lng, lat] = e.lngLat;
          cb({ lat, lng });
        });
      },
      addPin(p: LatLng, style?: PinStyle): MapPin {
        if (style?.label) {
          const marker = new mapgl.HtmlMarker(map, {
            coordinates: [p.lng, p.lat],
            html: labelHtml(style.label, style.color ?? "#e2590d"),
          });
          return {
            setPosition(next) {
              marker.setCoordinates([next.lng, next.lat]);
            },
            remove() {
              safeDestroy(marker as unknown as Disposable);
            },
          };
        }
        const marker = new mapgl.Marker(map, { coordinates: [p.lng, p.lat] });
        return {
          setPosition(next) {
            marker.setCoordinates([next.lng, next.lat]);
          },
          remove() {
            safeDestroy(marker as unknown as Disposable);
          },
        };
      },
      addVertex(p, style): MapShape {
        const marker = new mapgl.CircleMarker(map, {
          coordinates: [p.lng, p.lat],
          radius: style.radius,
          color: style.stroke,
          strokeWidth: style.width,
          strokeColor: "#ffffff",
        });
        return { remove: () => safeDestroy(marker as unknown as Disposable) };
      },
      addPolygon(points, style: ShapeStyle): MapShape {
        const coords = ring(points);
        // MapGL wants the ring closed explicitly.
        const polygon = new mapgl.Polygon(map, {
          coordinates: [[...coords, coords[0]]],
          color: style.fill,
          strokeColor: style.stroke,
          strokeWidth: style.width,
          interactive: false,
        });
        return { remove: () => safeDestroy(polygon as unknown as Disposable) };
      },
      addPolyline(points, style): MapShape {
        const line = new mapgl.Polyline(map, {
          coordinates: ring(points),
          color: style.stroke,
          width: style.width,
        });
        return { remove: () => safeDestroy(line as unknown as Disposable) };
      },
      destroy() {
        safeDestroy(map as unknown as Disposable);
      },
    };
  },
};
