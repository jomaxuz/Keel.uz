"use client";

// Polygon delivery zones, shown when the "zones" pricing mode is selected in
// the delivery settings. A zone needs 3+ points to be usable; if none are
// drawable the backend falls back to the radius model rather than refusing
// every address — see quoteDelivery in backend/internal/handlers/util.go.

import { useState } from "react";
import ZoneMap, { type LatLng } from "@/components/map/ZoneMap";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import type { DeliveryZone } from "@/lib/types";

const inputCls =
  "mt-1 w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand";

export default function DeliveryZonesEditor({
  zones,
  onChange,
  center,
}: {
  zones: DeliveryZone[];
  onChange: (next: DeliveryZone[]) => void;
  // Restaurant location — where the map opens.
  center: LatLng;
}) {
  const [active, setActive] = useState(0);
  const t = useAdminT();

  function update(i: number, patch: Partial<DeliveryZone>) {
    onChange(zones.map((z, zi) => (zi === i ? { ...z, ...patch } : z)));
  }

  function addZone() {
    onChange([
      ...zones,
      {
        name: `${t.zones.zoneName} ${zones.length + 1}`,
        pricing: "fixed",
        fee: 0,
        baseFee: 0,
        perKm: 0,
        polygon: [],
      },
    ]);
    setActive(zones.length);
  }

  function addPoint(p: LatLng) {
    const zone = zones[active];
    if (!zone) return;
    update(active, { polygon: [...(zone.polygon ?? []), [p.lat, p.lng]] });
  }

  const usable = zones.filter((z) => (z.polygon?.length ?? 0) >= 3);

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm text-ink-muted">
          {zones.length === 0 ? t.zones.noZones : t.zones.usable(usable.length)}
        </p>
        <button type="button" className="btn-ghost px-3 py-1.5 text-sm" onClick={addZone}>
          {t.zones.addZone}
        </button>
      </div>

      {zones.length > 0 && usable.length === 0 && (
        <p className="rounded-xl bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
          {t.zones.incomplete}
        </p>
      )}

      <div className="grid gap-4 lg:grid-cols-[320px_1fr]">
        {/* Zone list */}
        <div className="space-y-3">
          {zones.map((zone, i) => {
            const points = zone.polygon?.length ?? 0;
            const isActive = i === active;
            return (
              <div
                key={i}
                onClick={() => setActive(i)}
                className={`cursor-pointer rounded-2xl border p-3 transition-colors ${
                  isActive
                    ? "border-brand bg-brand-tint/40"
                    : "border-line hover:border-brand/50"
                }`}
              >
                <label className="block text-sm">
                  <span className="font-medium">{t.zones.zoneName}</span>
                  <input
                    className={inputCls}
                    value={zone.name}
                    onChange={(e) => update(i, { name: e.target.value })}
                  />
                </label>

                {/* Narxlash usuli — har zona uchun alohida. */}
                <div className="mt-3 flex flex-wrap gap-2">
                  {(
                    [
                      ["fixed", t.zones.pricingFixed],
                      ["perKm", t.zones.pricingPerKm],
                    ] as const
                  ).map(([mode, label]) => (
                    <button
                      key={mode}
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        update(i, { pricing: mode });
                      }}
                      className={`rounded-full border px-3 py-1.5 text-xs font-medium transition-colors ${
                        (zone.pricing ?? "fixed") === mode
                          ? "border-brand bg-brand text-white"
                          : "border-line-strong text-ink-soft hover:border-brand"
                      }`}
                    >
                      {label}
                    </button>
                  ))}
                </div>

                {(zone.pricing ?? "fixed") === "fixed" ? (
                  <label className="mt-3 block text-sm">
                    <span className="font-medium">{t.zones.fee}</span>
                    <input
                      type="number"
                      className={inputCls}
                      value={zone.fee}
                      onChange={(e) =>
                        update(i, { fee: Number(e.target.value) || 0 })
                      }
                    />
                    <span className="mt-1 block text-xs text-ink-muted">
                      {t.zones.feeHint}
                    </span>
                  </label>
                ) : (
                  <div className="mt-3">
                    <div className="grid gap-3 sm:grid-cols-2">
                      <label className="block text-sm">
                        <span className="font-medium">{t.zones.baseFee}</span>
                        <input
                          type="number"
                          className={inputCls}
                          value={zone.baseFee ?? 0}
                          onChange={(e) =>
                            update(i, { baseFee: Number(e.target.value) || 0 })
                          }
                        />
                      </label>
                      <label className="block text-sm">
                        <span className="font-medium">{t.zones.perKm}</span>
                        <input
                          type="number"
                          className={inputCls}
                          value={zone.perKm ?? 0}
                          onChange={(e) =>
                            update(i, { perKm: Number(e.target.value) || 0 })
                          }
                        />
                      </label>
                    </div>
                    <p className="mt-1 text-xs text-ink-muted">
                      {t.zones.perKmHint(
                        formatPrice(zone.baseFee ?? 0),
                        formatPrice(zone.perKm ?? 0),
                        formatPrice(
                          (zone.baseFee ?? 0) + (zone.perKm ?? 0) * 4,
                        ),
                      )}
                    </p>
                  </div>
                )}

                <div className="mt-2 flex flex-wrap items-center gap-3 text-xs">
                  <span
                    className={points >= 3 ? "text-ink-muted" : "text-red-600"}
                  >
                    {t.zones.points(points)}
                    {points < 3 && t.zones.needThree}
                  </span>
                  <span className="text-ink-muted">
                    ·{" "}
                    {(zone.pricing ?? "fixed") === "perKm"
                      ? `${formatPrice(zone.baseFee ?? 0)} + ${formatPrice(zone.perKm ?? 0)}/km`
                      : formatPrice(zone.fee)}
                  </span>
                  <button
                    type="button"
                    disabled={points === 0}
                    className="text-ink-muted hover:text-brand disabled:opacity-40"
                    onClick={(e) => {
                      e.stopPropagation();
                      update(i, { polygon: zone.polygon.slice(0, -1) });
                    }}
                  >
                    {t.zones.undoPoint}
                  </button>
                  <button
                    type="button"
                    disabled={points === 0}
                    className="text-ink-muted hover:text-brand disabled:opacity-40"
                    onClick={(e) => {
                      e.stopPropagation();
                      update(i, { polygon: [] });
                    }}
                  >
                    {t.zones.clearShape}
                  </button>
                  <button
                    type="button"
                    className="ml-auto text-ink-muted hover:text-red-600"
                    onClick={(e) => {
                      e.stopPropagation();
                      if (!confirm(t.zones.confirmDelete(zone.name))) return;
                      onChange(zones.filter((_, zi) => zi !== i));
                      setActive(0);
                    }}
                  >
                    {t.zones.deleteZone}
                  </button>
                </div>
              </div>
            );
          })}
        </div>

        {/* Map */}
        <div>
          <ZoneMap
            zones={zones.map((z) => ({
              name: z.name,
              points: (z.polygon ?? []) as [number, number][],
            }))}
            activeIndex={active}
            onAddPoint={addPoint}
            center={center}
            className="h-[420px] w-full"
          />
          <p className="mt-2 text-xs text-ink-muted">
            {zones.length === 0
              ? t.zones.mapHintEmpty
              : t.zones.mapHint(zones[active]?.name ?? "")}
          </p>
        </div>
      </div>
    </div>
  );
}
