"use client";

// Zones: the parts of the business tables belong to.
//
// ⚠️ **The reason this exists is that "table 112" is not a table.** A takeaway
// counter numbers its orders 100–130, and those numbers are not seats a guest
// can reserve — but the till needs them, because a takeaway order has to be
// opened against something. Before zones there was one list, and it could
// either put those numbers on the booking page or keep them off the till.
//
// ⚠️ **A restaurant that has not split its room sees nothing new.** With no
// zones every table is bookable and the till draws one grid — which is what it
// did before this existed, and what most restaurants will keep doing.

import { useState } from "react";

import { useAdminT } from "@/lib/i18n/admin";
import type { FloorTable, TableZone } from "@/lib/types";

export default function ZonesEditor({
  zones,
  tables,
  onChange,
}: {
  zones: TableZone[];
  tables: FloorTable[];
  onChange: (zones: TableZone[], tables: FloorTable[]) => void;
}) {
  const t = useAdminT();
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [target, setTarget] = useState("");

  const count = (zoneID: string) =>
    tables.filter((tb) => (tb.zoneId ?? "") === zoneID).length;

  function addZone(layout: "map" | "list") {
    const id = `z${Date.now().toString(36)}`;
    onChange(
      [
        ...zones,
        {
          id,
          name: layout === "list" ? t.tableZones.takeawayName : t.tableZones.hallName,
          // ⚠️ A list zone defaults to **not bookable**: the only reason to
          // make one is a takeaway counter, and a takeaway number that lands on
          // the booking page is the mistake this whole feature prevents.
          bookable: layout === "map",
          layout,
          sort: zones.length,
        },
      ],
      tables,
    );
  }

  function patchZone(id: string, p: Partial<TableZone>) {
    onChange(
      zones.map((z) => (z.id === id ? { ...z, ...p } : z)),
      tables,
    );
  }

  function removeZone(id: string) {
    // ⚠️ The tables are kept and released to the default zone, never deleted
    // with it. A zone removed by mistake must not take a night's worth of open
    // checks and every reservation pointing at those tables with it.
    onChange(
      zones.filter((z) => z.id !== id),
      tables.map((tb) => (tb.zoneId === id ? { ...tb, zoneId: "" } : tb)),
    );
  }

  /** Create tables 100…130 in one go.
   *
   *  ⚠️ A takeaway counter has thirty numbers and no shape, and drawing them
   *  one at a time on a floor plan is half an hour of work that produces
   *  nothing anybody looks at. They are made here, with no coordinates, and the
   *  till renders that zone as a list. */
  function addRange() {
    const a = Number(from);
    const b = Number(to);
    if (!target || !a || !b || b < a || b - a > 200) return;
    const existing = new Set(tables.map((tb) => tb.number));
    const made: FloorTable[] = [];
    for (let n = a; n <= b; n++) {
      const number = String(n);
      // Skipped rather than duplicated: a second table numbered 112 makes
      // "which one is 112" unanswerable on every screen.
      if (existing.has(number)) continue;
      made.push({
        id: `t${n}-${Date.now().toString(36)}`,
        number,
        seats: 0,
        shape: "rect",
        x: 0,
        y: 0,
        w: 0,
        h: 0,
        isActive: true,
        note: "",
        zoneId: target,
      });
    }
    onChange(zones, [...tables, ...made]);
    setFrom("");
    setTo("");
  }

  const listZones = zones.filter((z) => z.layout === "list");

  return (
    <div className="mt-4 rounded-2xl border border-line bg-ink/[0.02] p-4">
      <p className="text-sm font-semibold">{t.tableZones.title}</p>
      <p className="mt-1 text-xs leading-relaxed text-ink-muted">
        {t.tableZones.hint}
      </p>

      {zones.length === 0 && (
        // ⚠️ Said rather than shown as an empty list: no zones is the ordinary
        // state and it is not a missing setting, it is one bookable room.
        <p className="mt-3 text-xs text-ink-soft">{t.tableZones.none}</p>
      )}

      <ul className="mt-3 space-y-2">
        {zones.map((z) => (
          <li
            key={z.id}
            className="flex flex-wrap items-center gap-2 rounded-xl border border-line bg-surface p-2"
          >
            <input
              className="input h-9 w-40"
              value={z.name}
              onChange={(e) => patchZone(z.id, { name: e.target.value })}
            />
            <select
              className="input h-9 w-32"
              value={z.layout ?? "map"}
              onChange={(e) =>
                patchZone(z.id, { layout: e.target.value as "map" | "list" })
              }
            >
              <option value="map">{t.tableZones.layoutMap}</option>
              <option value="list">{t.tableZones.layoutList}</option>
            </select>
            <label className="flex items-center gap-1 text-xs">
              <input
                type="checkbox"
                checked={z.bookable}
                onChange={(e) => patchZone(z.id, { bookable: e.target.checked })}
              />
              {t.tableZones.bookable}
            </label>
            <span className="text-xs text-ink-muted">
              {t.tableZones.tableCount(count(z.id))}
            </span>
            <button
              type="button"
              className="btn-ghost ml-auto px-2 text-xs text-danger"
              onClick={() => removeZone(z.id)}
            >
              {t.common.delete}
            </button>
          </li>
        ))}
      </ul>

      <div className="mt-3 flex flex-wrap gap-2">
        <button type="button" className="btn" onClick={() => addZone("map")}>
          + {t.tableZones.addMap}
        </button>
        <button type="button" className="btn" onClick={() => addZone("list")}>
          + {t.tableZones.addList}
        </button>
      </div>

      {/* ---- Bulk numbering ---- */}
      {listZones.length > 0 && (
        <div className="mt-4 border-t border-line pt-3">
          <p className="text-sm font-medium">{t.tableZones.rangeTitle}</p>
          <p className="mt-1 text-xs text-ink-muted">{t.tableZones.rangeHint}</p>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <select
              className="input h-9 w-40"
              value={target}
              onChange={(e) => setTarget(e.target.value)}
            >
              <option value="">{t.tableZones.pickZone}</option>
              {listZones.map((z) => (
                <option key={z.id} value={z.id}>
                  {z.name}
                </option>
              ))}
            </select>
            <input
              className="input h-9 w-24"
              inputMode="numeric"
              placeholder="100"
              value={from}
              onChange={(e) => setFrom(e.target.value.replace(/\D/g, ""))}
            />
            <span className="text-ink-muted">—</span>
            <input
              className="input h-9 w-24"
              inputMode="numeric"
              placeholder="130"
              value={to}
              onChange={(e) => setTo(e.target.value.replace(/\D/g, ""))}
            />
            <button
              type="button"
              className="btn"
              disabled={!target || !from || !to}
              onClick={addRange}
            >
              {t.tableZones.addRange}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
