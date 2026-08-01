"use client";

// One-tap route to a point in the map app the customer already uses. On a phone
// these URLs open the installed app (Yandex Maps / Google Maps / 2GIS) and fall
// back to the website in a desktop browser — no SDK, no API key, nothing to
// break. The origin is deliberately left empty in every link so each app uses
// the phone's own location, which it knows better than we do.

import { useI18n } from "@/lib/i18n/client";

export interface RouteTarget {
  lat: number;
  lng: number;
  // Only used by Google Maps, which accepts a readable destination label.
  label?: string;
}

// Note the coordinate order: Yandex and Google take lat,lng — 2GIS takes lng,lat.
//
// 2GIS: the old `/routeSearch/rsType/car/to/...` path is dead — it 301s to the
// city page, which is what "the route is not built" looked like. The live format
// is `/directions/points/<id>|lon,lat` with an empty id for a bare coordinate
// (the pipe must stay percent-encoded or the path is truncated).
function links({ lat, lng }: RouteTarget) {
  const ll = `${lat},${lng}`;
  return {
    yandex: `https://yandex.uz/maps/?rtext=~${ll}&rtt=auto&z=16`,
    google: `https://www.google.com/maps/dir/?api=1&destination=${ll}&travelmode=driving`,
    gis: `https://2gis.uz/directions/points/%7C${lng}%2C${lat}`,
  };
}

export default function RouteButtons({
  target,
  className = "",
  title,
}: {
  target: RouteTarget;
  className?: string;
  /** Where the route leads. Defaults to the restaurant, because that is who
   *  the guest is being sent to; the courier app passes the customer instead —
   *  the same three links, the opposite direction. */
  title?: string;
}) {
  const { t } = useI18n();
  if (!target.lat || !target.lng) return null;
  const url = links(target);

  const items: [string, string][] = [
    [url.yandex, t.order.routeYandex],
    [url.google, t.order.routeGoogle],
    [url.gis, t.order.route2gis],
  ];

  return (
    <div className={className}>
      <p className="text-xs font-medium text-ink-muted">
        {title ?? t.order.routeTitle}
      </p>
      <div className="mt-2 flex flex-wrap gap-2">
        {items.map(([href, label]) => (
          <a
            key={label}
            href={href}
            target="_blank"
            rel="noreferrer"
            className="rounded-full border border-line-strong px-3.5 py-1.5 text-sm font-semibold text-ink-soft transition-colors hover:border-brand hover:text-brand"
          >
            ➤ {label}
          </a>
        ))}
      </div>
    </div>
  );
}
