"use client";

// The branch list and the shared map, side by side.
//
// ⚠️ **One map for all of them, not a map per row.** Six small maps is six MapGL
// instances on one page — and the question a guest has ("which of these is near me?") is
// answerable only when the pins are on the same picture. Selecting a row centres that
// pin; selecting a pin is the same thing from the other side.
//
// ⚠️ The map is on the right on a desktop and **above** the list on a phone, where a
// list that pushed it off-screen would leave a map nobody scrolls back up to.

import { useState } from "react";
import LiveMap, { type MapPoint } from "@/components/map/LiveMap";
import RouteButtons from "@/components/map/RouteButtons";
import CallLink from "@/components/site/CallLink";
import { useI18n } from "@/lib/i18n/client";
import { weekdayName } from "@/lib/format";
import type { Branch } from "@/lib/types";
import type { Lang } from "@/lib/i18n/dictionaries";

export default function BranchesView({
  branches,
  lang,
}: {
  branches: Branch[];
  lang: Lang;
}) {
  const { t } = useI18n();
  const [active, setActive] = useState<string>("");

  const mapped = branches.filter((b) => b.address?.lat && b.address?.lng);
  const points: MapPoint[] = mapped.map((b) => ({
    id: b.id,
    lat: b.address.lat,
    lng: b.address.lng,
    label: b.name,
    // The selected branch is drawn as the destination and the rest as idle, so the row
    // and the pin agree about which one is being looked at.
    kind: active === b.id ? "destination" : "restaurant",
  }));

  if (branches.length === 0) {
    return (
      <div className="container-page py-16">
        <p className="text-sm text-ink-muted">{t.branches.empty}</p>
      </div>
    );
  }

  return (
    <div className="container-page grid grid-cols-1 gap-6 py-10 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)]">
      {/* The map first in the DOM so it is above the list on a phone, and moved to the
          right on a desktop — `lg:order-2` rather than a second copy of the markup. */}
      {points.length > 0 && (
        <div className="lg:order-2">
          <div className="card sticky top-24 overflow-hidden p-0">
            <LiveMap
              points={points}
              fallbackCenter={{ lat: points[0].lat, lng: points[0].lng }}
              className="h-72 w-full sm:h-[28rem]"
              // Re-fit whenever the selection changes: the point of tapping a row is to
              // see where that branch is.
              autoFit="always"
            />
          </div>
        </div>
      )}

      <ul className="space-y-4 lg:order-1">
        {branches.map((b) => {
          const hours = (b.workingHours ?? []).filter((h) => !h.isClosed);
          const today = new Date().getDay();
          const todayHours = (b.workingHours ?? []).find((h) => h.day === today);
          return (
            <li
              key={b.id}
              onClick={() => setActive(b.id)}
              className={`card cursor-pointer p-5 transition-colors ${
                active === b.id ? "border-brand" : ""
              }`}
            >
              <h2 className="font-display text-lg font-bold">{b.name}</h2>

              {b.address?.text && (
                <p className="mt-1.5 text-sm text-ink-muted">{b.address.text}</p>
              )}

              {/* Today's hours rather than the whole week: a guest deciding whether to
                  set off needs one line, and the full table is on the about page. */}
              {todayHours && (
                <p className="mt-1 text-xs text-ink-muted">
                  {weekdayName(today, lang)}:{" "}
                  {todayHours.isClosed
                    ? t.common.closed
                    : `${todayHours.open} – ${todayHours.close}`}
                </p>
              )}
              {!todayHours && hours.length > 0 && (
                <p className="mt-1 text-xs text-ink-muted">
                  {hours[0].open} – {hours[0].close}
                </p>
              )}

              {(b.phones ?? []).length > 0 && (
                <ul className="mt-3 space-y-1 text-sm">
                  {b.phones.map((p) => (
                    <li key={p}>
                      <CallLink phone={p} className="font-semibold hover:text-brand">
                        {p}
                      </CallLink>
                    </li>
                  ))}
                </ul>
              )}

              {/* ⚠️ The route buttons hand the journey to the app the phone already has —
                  Yandex, Google or 2GIS — with no starting point, because each of them
                  knows where the guest is and we do not. */}
              {b.address?.lat && b.address?.lng ? (
                <div className="mt-4" onClick={(e) => e.stopPropagation()}>
                  <RouteButtons
                    target={{
                      lat: b.address.lat,
                      lng: b.address.lng,
                      label: b.address.text || b.name,
                    }}
                  />
                </div>
              ) : null}
            </li>
          );
        })}
      </ul>
    </div>
  );
}
