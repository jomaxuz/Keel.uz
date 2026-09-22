// The dashboard's tiles, as data.
//
// They were twenty JSX elements in four hard-coded blocks, which is the right
// shape right up until somebody wants to turn one off. A registry keyed by a
// stable id means the front page and the screen that arranges it are reading
// the same list — the alternative is two lists that agree until one of them is
// edited, and then a tile nobody can find a switch for.
//
// ⚠️ **The ids are stable and match the server's `dashboardTileIDs`.** They are
// stored on the admin's account, so renaming one silently un-hides whatever the
// old id named, on every account that had hidden it.

import type { PanelWords } from "@/lib/panelWords";
import type { AdminStats } from "@/lib/types";
import type { useAdminT } from "@/lib/i18n/admin";

type T = ReturnType<typeof useAdminT>;

/** Which heading a tile lives under. A tile keeps its group whatever the
 *  admin's order says: "Pul" with an order count under it is a heading that
 *  lies, and headings are the only thing making twenty figures readable. */
export type TileGroup = "orders" | "money" | "people" | "menu";

export const TILE_GROUPS: TileGroup[] = ["orders", "money", "people", "menu"];

export interface TileSpec {
  id: string;
  group: TileGroup;
  /** ⚠️ **The words, as well as the dictionary.** One tile counts the thing
   *  this business sells, and a chemist's front page headed "Taomlar" is the
   *  sidebar's old complaint on the first screen anybody opens. Every other
   *  tile ignores the second argument. See lib/panelWords.ts. */
  label: (t: T, w: PanelWords) => string;
  /** The figure. `null`/`undefined` means "not loaded", which the caller
   *  renders as "…" rather than as 0 — a dashboard that shows 0 while loading
   *  tells the reader something false for as long as the request takes. */
  value: (s: AdminStats) => number | undefined;
  /** Money is formatted differently and it is a property of the figure, not of
   *  the caller's mood. */
  money?: boolean;
  hint?: (s: AdminStats, t: T) => string | undefined;
  accent?: boolean;
}

export const DASHBOARD_TILES: TileSpec[] = [
  // ---- Orders ----
  {
    id: "orders.total",
    group: "orders",
    label: (t) => t.dashboard.ordersTotal,
    value: (s) => s.period.orders,
  },
  {
    id: "orders.delivered",
    group: "orders",
    label: (t) => t.dashboard.ordersDelivered,
    value: (s) => s.period.delivered,
  },
  {
    id: "orders.delivery",
    group: "orders",
    label: (t) => t.dashboard.ordersDelivery,
    value: (s) => s.period.delivery,
  },
  {
    id: "orders.pickup",
    group: "orders",
    label: (t) => t.dashboard.ordersPickup,
    value: (s) => s.period.pickup,
  },
  {
    id: "orders.dineIn",
    group: "orders",
    label: (t) => t.dashboard.ordersDineIn,
    value: (s) => s.period.dineIn,
  },
  {
    id: "orders.cancelled",
    group: "orders",
    label: (t) => t.dashboard.ordersCancelled,
    value: (s) => s.period.cancelled,
  },

  // ---- Money ----
  {
    id: "money.revenue",
    group: "money",
    label: (t) => t.dashboard.revenue,
    value: (s) => s.period.revenue,
    money: true,
    accent: true,
  },
  {
    // Beside the takings, never folded into them. An owner does want to know
    // what today is still going to bring in — they just must not be told they
    // already have it.
    id: "money.pending",
    group: "money",
    label: (t) => t.dashboard.pending,
    value: (s) => s.period.pending,
    money: true,
  },
  {
    // ⚠️ **Beside `pending`, not inside it.** The rest of what is owed arrives
    // by itself within the hour; this part arrives when somebody rings the
    // guest, and a slate that keeps growing is invisible while it is folded
    // into "still to come".
    id: "money.debt",
    group: "money",
    label: (t) => t.dashboard.debt,
    value: (s) => s.period.debt,
    money: true,
  },
  {
    id: "money.avgOrder",
    group: "money",
    label: (t) => t.dashboard.avgOrder,
    value: (s) => s.period.avgOrder,
    money: true,
  },
  {
    id: "money.deliveryFee",
    group: "money",
    label: (t) => t.dashboard.deliveryFees,
    value: (s) => s.period.deliveryFee,
    money: true,
  },
  {
    id: "money.cash",
    group: "money",
    label: (t) => t.dashboard.cashTotal,
    value: (s) => s.period.cashTotal,
    money: true,
  },

  // ---- People ----
  {
    id: "people.usersTotal",
    group: "people",
    label: (t) => t.dashboard.usersTotal,
    value: (s) => s.users.total,
  },
  {
    id: "people.usersNew",
    group: "people",
    label: (t) => t.dashboard.usersNew,
    value: (s) => s.users.new,
  },
  {
    id: "people.usersActive",
    group: "people",
    label: (t) => t.dashboard.usersActive,
    value: (s) => s.users.active,
    hint: (_s, t) => t.dashboard.usersActiveHint,
  },
  {
    id: "people.couriers",
    group: "people",
    label: (t) => t.dashboard.couriersTotal,
    value: (s) => s.couriers.total,
    hint: (s, t) => `${t.dashboard.couriersOnline}: ${s.couriers.online}`,
  },
  {
    id: "people.admins",
    group: "people",
    label: (t) => t.dashboard.adminsTotal,
    value: (s) => s.admins.total,
    hint: (s, t) => t.dashboard.adminsSplit(s.admins.owners, s.admins.managers),
  },
  {
    id: "people.withAddress",
    group: "people",
    label: (t) => t.dashboard.usersWithAddress,
    value: (s) => s.users.withAddress,
  },

  // ---- Menu ----
  {
    id: "menu.dishes",
    group: "menu",
    label: (_t, w) => w.items,
    value: (s) => s.menu.dishes,
  },
  {
    id: "menu.available",
    group: "menu",
    label: (t) => t.dashboard.menuAvailable,
    value: (s) => s.menu.available,
  },
  {
    id: "menu.categories",
    group: "menu",
    label: (t) => t.dashboard.menuCategories,
    value: (s) => s.menu.categories,
  },
];

const BY_ID = new Map(DASHBOARD_TILES.map((tile) => [tile.id, tile]));

/** Looks a tile up by id.
 *
 *  Returns undefined for an id the panel does not know, which happens for real:
 *  an account can carry a preference naming a tile a later version removed.
 *  Callers skip those rather than crashing — a stored setting must never be
 *  able to take the front page down. */
export function tileById(id: string): TileSpec | undefined {
  return BY_ID.get(id);
}

/** The tiles to draw, in order, for a group.
 *
 *  `visible` comes from the server with hiding and ordering already applied;
 *  when it has not loaded yet the default order stands in, so the dashboard
 *  draws its usual self while the preference is on its way rather than
 *  flickering from empty to full. */
export function tilesForGroup(group: TileGroup, visible: string[] | null): TileSpec[] {
  const ids = visible ?? DASHBOARD_TILES.map((tile) => tile.id);
  return ids
    .map(tileById)
    .filter((tile): tile is TileSpec => !!tile && tile.group === group);
}
