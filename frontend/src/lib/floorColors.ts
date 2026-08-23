/** The colours a floor plan may be painted in.
 *
 * ⚠️ **A closed list, and the same one the server keeps.** These names reach an
 * SVG `fill` on the public booking page, so the value stored is a *name* rather
 * than a colour: an arbitrary string there would make the plan editor a way to
 * put something that is not a colour onto every guest's screen. The server
 * clamps to the same set on save (models.FloorColor) — this half is so nobody
 * can pick one that would be thrown away.
 *
 * ⚠️ **Six, because a dining room has about that many areas.** A picker with
 * thirty swatches is a decision nobody wants to make about a wall, and the one
 * thing this feature has to be is faster than not using it.
 *
 * ⚠️ **Tint and line, not one value.** A filled area at full strength swallows
 * the table numbers drawn on top of it — which is the only thing the plan is
 * read for. The colour lives in the outline and a wash of it in the fill, the
 * same division the till's tiles use.
 */
export const FLOOR_COLORS = [
  "slate",
  "amber",
  "green",
  "blue",
  "rose",
  "violet",
] as const;

export type FloorColor = (typeof FLOOR_COLORS)[number];

interface Paint {
  /** Behind everything, so text stays readable on it. */
  fill: string;
  /** The outline, where the colour is allowed to be itself. */
  line: string;
  /** The swatch in the picker. */
  swatch: string;
}

const PAINT: Record<FloorColor, Paint> = {
  slate: { fill: "rgb(100 116 139 / 0.10)", line: "rgb(100 116 139 / 0.55)", swatch: "#64748b" },
  amber: { fill: "rgb(245 158 11 / 0.14)", line: "rgb(217 119 6 / 0.65)", swatch: "#f59e0b" },
  green: { fill: "rgb(34 197 94 / 0.13)", line: "rgb(22 163 74 / 0.6)", swatch: "#22c55e" },
  blue: { fill: "rgb(59 130 246 / 0.13)", line: "rgb(37 99 235 / 0.6)", swatch: "#3b82f6" },
  rose: { fill: "rgb(244 63 94 / 0.12)", line: "rgb(225 29 72 / 0.55)", swatch: "#f43f5e" },
  violet: { fill: "rgb(139 92 246 / 0.13)", line: "rgb(124 58 237 / 0.6)", swatch: "#8b5cf6" },
};

/** What to paint a shape or table with.
 *
 * ⚠️ **An unknown name falls back rather than throwing.** The value arrives
 * from a document that may have been written by an older build or edited by
 * hand, and a floor plan that refuses to draw is worse than one drawn in grey —
 * the room is what the page is for. */
export function floorPaint(color: string | undefined, fallback: Paint): Paint {
  if (color && color in PAINT) return PAINT[color as FloorColor];
  return fallback;
}

export function floorSwatch(color: FloorColor): string {
  return PAINT[color].swatch;
}
