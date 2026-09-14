import type { Order, OrderStatus } from "./types";

// The kitchen flow. Every order walks pending → confirmed → preparing →
// on_the_way → delivered; pickup and dine-in orders skip "on the way".
export const FLOW: OrderStatus[] = [
  "pending",
  "confirmed",
  "preparing",
  "on_the_way",
  "delivered",
];

// Whether the order leaves the building with a courier — ours or the
// marketplace's.
function travels(order: Order): boolean {
  return order.type === "delivery" || order.type === "uzum_tezkor";
}

// The status that follows the current one for this order type.
export function nextStatus(order: Order): OrderStatus | null {
  if (order.status === "delivered" || order.status === "cancelled") return null;
  const i = FLOW.indexOf(order.status);
  if (i < 0) return null;
  // Pickup orders never go "on the way". ⚠️ A Uzum Tezkor order does: its
  // courier collecting the food is the step Uzum reads as TAKEN_BY_COURIER, and
  // skipping it would jump the marketplace straight from cooking to delivered.
  if (!travels(order) && FLOW[i + 1] === "on_the_way") {
    return "delivered";
  }
  return FLOW[i + 1] ?? null;
}

// Label for the one-click "advance" button in the admin panel — the whole
// point is that a normal order needs a single tap per stage, not a dropdown.
// The wording comes from the caller's dictionary so the panel can be
// translated; pickup and dine-in orders get their own wording for the
// hand-over step (nobody drives anywhere).
export function nextActionLabel(
  order: Order,
  labels: {
    pending: string;
    confirmed: string;
    preparing: string;
    preparingPickup: string;
    on_the_way: string;
    /** Handing a Uzum Tezkor order to their courier — nobody of ours drives
     *  it. Falls back to the delivery wording when a caller has none. */
    uzumHandover?: string;
  },
): string | null {
  if (!nextStatus(order)) return null;
  if (order.status === "preparing" && order.type === "uzum_tezkor") {
    return labels.uzumHandover ?? labels.preparing;
  }
  if (order.status === "preparing" && !travels(order)) {
    return labels.preparingPickup;
  }
  switch (order.status) {
    case "pending":
      return labels.pending;
    case "confirmed":
      return labels.confirmed;
    case "preparing":
      return labels.preparing;
    case "on_the_way":
      return labels.on_the_way;
    default:
      return null;
  }
}

// "5 daqiqa oldin" — relative time for the orders and customer lists.
//
// ⚠️ **The wording comes from the caller's dictionary**, like nextActionLabel
// above and for the same reason. It used to be hardcoded Uzbek, which meant a
// panel switched to Russian still read "5 daq oldin" beside every order — the
// kind of leftover that tells an owner the translation is a veneer, on the one
// screen they look at all day.
//
// Plural forms belong to the dictionary too, not here: Russian needs three of
// them and English two, and a formatter that tried to serve both would be
// re-implementing each language's grammar next to the arithmetic.
export function timeAgo(
  iso: string,
  labels: {
    now: string;
    min: (n: number) => string;
    hour: (n: number) => string;
    day: (n: number) => string;
  },
): string {
  const diff = Date.now() - new Date(iso).getTime();
  const min = Math.round(diff / 60000);
  if (min < 1) return labels.now;
  if (min < 60) return labels.min(min);
  const h = Math.round(min / 60);
  if (h < 24) return labels.hour(h);
  return labels.day(Math.round(h / 24));
}

// Re-exported from here because a dozen screens already import it from this
// module; the implementation lives in lib/format.ts with the other formatters.
export { formatDateTime } from "./format";
