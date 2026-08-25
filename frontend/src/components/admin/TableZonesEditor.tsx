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

/** How a bulk-created hall table is laid out, in plan units.
 *
 *  ⚠️ Sized like the tables the editor draws by hand (see FloorPlanEditor's
 *  MIN_SIZE) so a grid made here and a table drawn there look like the same
 *  room. Eight to a row fits the default 1000-unit plan with margins. */
const PER_ROW = 8;
const CELL_W = 90;
const CELL_H = 70;
const GAP = 24;
const MARGIN = 40;

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

  /** Create tables 1…40 or 100…130 in one go.
   *
   *  ⚠️ **Drawing forty tables one at a time is half an hour of work**, and a
   *  takeaway counter's thirty numbers produce nothing anybody looks at at the
   *  end of it. Both kinds of zone are offered here — see the layout note below
   *  for why they are not made the same way.
   *
   *  ⚠️ **A number already in use is skipped, not duplicated.** A second table
   *  numbered 112 makes "which one is 112" unanswerable on every screen that
   *  shows one.
   */
  function addRange() {
    const a = Number(from);
    const b = Number(to);
    if (!a || !b || b < a || b - a > 200) return;
    // ⚠️ **An empty target is the unnamed room, not a missing answer.** No
    // zones at all is the ordinary state — this screen says so itself — and
    // requiring one here meant the restaurants most likely to want forty
    // numbered tables were the ones that could not make them. It is also how a
    // live branch ended up with thirty-one tables carrying no zone, no
    // coordinates and no seats: created before this was true, and invisible on
    // every screen that groups by zone ever since.
    const zone = zones.find((z) => z.id === target);
    const onMap = (zone?.layout ?? "map") === "map";
    const existing = new Set(tables.map((tb) => tb.number));
    const made: FloorTable[] = [];
    // ⚠️ **A hall's tables are laid out on a grid, a counter's are not.**
    // This used to refuse map zones outright, and the reason was sound: a
    // table with no coordinates sits at 0,0, so forty of them pile into the
    // top-left corner — a room that reads as broken, and the exact thing the
    // "is there a plan?" check exists to catch. Refusing was the wrong half of
    // the answer, though: what the owner wants is forty tables they can then
    // drag into place, and making them one at a time is the work this button
    // exists to remove. So they are placed in rows to start with — visible,
    // separate, and draggable from the first frame.
    //
    // Placed after whatever is already drawn, so a second range does not land
    // on top of the first.
    const startRow = Math.floor(
      tables.filter((tb) => (tb.zoneId ?? "") === target).length / PER_ROW,
    );
    let made_i = 0;
    for (let n = a; n <= b; n++) {
      const number = String(n);
      if (existing.has(number)) continue;
      const row = startRow + Math.floor(made_i / PER_ROW);
      const col = made_i % PER_ROW;
      made_i++;
      made.push({
        id: `t${n}-${Date.now().toString(36)}-${made_i}`,
        number,
        // ⚠️ Four seats on a hall table, none on a counter slot: one is a table
        // somebody sits at and the other is a number an order is called back
        // by. A counter slot showing "0 o'rin" is a fact nobody needed.
        seats: onMap ? 4 : 0,
        shape: "rect",
        x: onMap ? MARGIN + col * (CELL_W + GAP) : 0,
        y: onMap ? MARGIN + row * (CELL_H + GAP) : 0,
        w: onMap ? CELL_W : 0,
        h: onMap ? CELL_H : 0,
        isActive: true,
        note: "",
        zoneId: target,
      });
    }
    onChange(zones, [...tables, ...made]);
    setFrom("");
    setTo("");
  }

  /** Tables that belong on a plan and have never been placed on one.
   *
   *  ⚠️ **They are not missing, they are stacked.** A table with no coordinates
   *  sits at 0,0, so thirty of them are one square in the top-left corner —
   *  present in the data, absent from every screen, and impossible to drag
   *  apart because they are exactly on top of each other. The only way out is
   *  from here. */
  const unplaced = tables.filter((tb) => {
    const zone = zones.find((z) => z.id === (tb.zoneId ?? ""));
    if ((zone?.layout ?? "map") !== "map") return false;
    return !tb.x && !tb.y;
  });

  /** Put them in rows, after whatever is already drawn in their own zone. */
  function placeAll() {
    const rows = new Map<string, number>();
    for (const tb of tables) {
      const key = tb.zoneId ?? "";
      if (tb.x || tb.y) rows.set(key, (rows.get(key) ?? 0) + 1);
    }
    const placed = new Map<string, number>();
    onChange(
      zones,
      tables.map((tb) => {
        if (!unplaced.includes(tb)) return tb;
        const key = tb.zoneId ?? "";
        const i = placed.get(key) ?? Math.ceil((rows.get(key) ?? 0) / PER_ROW) * PER_ROW;
        placed.set(key, i + 1);
        const row = Math.floor(i / PER_ROW);
        const col = i % PER_ROW;
        return {
          ...tb,
          x: MARGIN + col * (CELL_W + GAP),
          y: MARGIN + row * (CELL_H + GAP),
          w: tb.w || CELL_W,
          h: tb.h || CELL_H,
          // ⚠️ Seats are left alone when somebody has set them: this repairs a
          // position, and quietly rewriting how many people fit at a table
          // would be a second change nobody asked for.
          seats: tb.seats || 4,
        };
      }),
    );
  }

  // ⚠️ **Every zone, not only the counters.** The dropdown listed `list` zones
  // alone, so a restaurant that had made a hall and wanted its forty tables
  // numbered found an empty picker and no explanation — the button was there,
  // the zone was there, and the two could not be connected.
  const rangeZones = zones;

  /** The unnamed room, offered whether or not any zone exists — see addRange. */
  const roomCount = count("");

  return (
    <div className="mt-4 rounded-2xl border border-line bg-ink/[0.02] p-4">
      <p className="text-sm font-semibold">{t.tableZones.title}</p>
      <p className="mt-1 text-xs leading-relaxed text-ink-muted">
        {t.tableZones.hint}
      </p>

      {/* ⚠️ Shown only when there is something to repair, and it says how many:
          "some tables are not on the plan" is a sentence somebody ignores, and
          "31 tables are stacked in the corner" is one they press. */}
      {unplaced.length > 0 && (
        <div className="mt-3 flex flex-wrap items-center gap-2 rounded-xl border border-amber-500/40 bg-amber-500/[0.12] p-3">
          <p className="flex-1 text-sm text-ink">
            {t.tableZones.unplaced(unplaced.length)}
          </p>
          <button type="button" className="btn" onClick={placeAll}>
            {t.tableZones.placeAll}
          </button>
        </div>
      )}

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
      {rangeZones.length > 0 && (
        <div className="mt-4 border-t border-line pt-3">
          <p className="text-sm font-medium">{t.tableZones.rangeTitle}</p>
          <p className="mt-1 text-xs text-ink-muted">{t.tableZones.rangeHint}</p>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <select
              className="input h-9 w-40"
              value={target}
              onChange={(e) => setTarget(e.target.value)}
            >
              {/* ⚠️ **The unnamed room is an option, not the absence of one.**
                  It used to read "choose a zone" and do nothing when left
                  alone — so a restaurant with no zones, which this screen calls
                  the ordinary state, had a button that silently did nothing. */}
              <option value="">
                {t.tableZones.wholeRoom}
                {roomCount > 0 ? ` · ${roomCount}` : ""}
              </option>
              {rangeZones.map((z) => (
                <option key={z.id} value={z.id}>
                  {z.name}
                  {/* Which way the tables will be made, said in the option
                      itself: the same range produces a grid in a hall and a
                      bare list at a counter, and that is worth knowing before
                      pressing rather than after. */}
                  {(z.layout ?? "map") === "map"
                    ? ` — ${t.tableZones.layoutMap}`
                    : ` — ${t.tableZones.layoutList}`}
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
