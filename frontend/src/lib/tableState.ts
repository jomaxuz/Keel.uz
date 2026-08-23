/** What a table is doing, as one word.
 *
 * ⚠️ **One definition, three renderers.** The grid, the floor plan and the
 * waiter cards all colour a table, and until this existed each worked the
 * threshold and the precedence out for itself — so a table could be amber in
 * the grid and red on the plan, on the same screen, at the same minute. A room
 * that disagrees with itself about which table is late is a room nobody
 * believes about any of them.
 *
 * ⚠️ **The bill outranks the age.** A table that has asked to pay is waiting
 * for a person, not for food, and forty minutes of that is a different problem
 * from forty minutes of eating — it goes to the waiter, not to the kitchen.
 */

import type { Check } from "@/lib/types";

export type TableState = "free" | "open" | "late" | "billed";

/** How long a table can sit before the tile stops being ordinary.
 *
 * ⚠️ **Not a rule about service, a rule about attention.** Forty-five minutes
 * is a normal lunch and a long wait for a bill, so the colour does not accuse
 * anybody — it answers the only question this screen is scanned for during a
 * rush: which table has nobody looking at it. A shorter threshold turns the
 * whole room red at eight o'clock, and a room that is always red says nothing.
 */
export const LATE_MIN = 45;

export function tableState(check?: Check | null): TableState {
  if (!check) return "free";
  if (check.precheckAt) return "billed";
  if (check.openMin >= LATE_MIN) return "late";
  return "open";
}

/** The colour each state carries, as the till's own tokens.
 *
 * ⚠️ Returned as strings rather than class names because two of the three
 * renderers are SVG, where Tailwind's `bg-*` does nothing. */
export function stateColor(s: TableState): string {
  switch (s) {
    case "billed":
      return "rgb(var(--till-info))";
    case "late":
      return "rgb(var(--till-late))";
    case "open":
      return "rgb(var(--till-accent))";
    default:
      return "rgb(var(--till-ok))";
  }
}

/** The wash under an occupied table. `free` is left as the surface: a room
 *  where every table is tinted has no free tables to find. */
export function stateTint(s: TableState): string {
  switch (s) {
    case "billed":
      return "#F2F7FD";
    case "late":
      return "#FDF3F3";
    case "open":
      return "#FFFBF2";
    default:
      return "rgb(var(--surface))";
  }
}

export function stateLine(s: TableState): string {
  switch (s) {
    case "billed":
      return "rgb(var(--till-info) / 0.55)";
    case "late":
      return "rgb(var(--till-late) / 0.5)";
    case "open":
      return "rgb(var(--till-accent-line))";
    default:
      return "var(--line-strong)";
  }
}
