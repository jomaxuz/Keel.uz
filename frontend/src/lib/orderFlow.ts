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

// The status that follows the current one for this order type.
export function nextStatus(order: Order): OrderStatus | null {
  if (order.status === "delivered" || order.status === "cancelled") return null;
  const i = FLOW.indexOf(order.status);
  if (i < 0) return null;
  // Pickup orders never go "on the way".
  if (order.type !== "delivery" && FLOW[i + 1] === "on_the_way") {
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
  },
): string | null {
  if (!nextStatus(order)) return null;
  if (order.status === "preparing" && order.type !== "delivery") {
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

// "5 daqiqa oldin" — relative time for the orders list.
export function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const min = Math.round(diff / 60000);
  if (min < 1) return "hozir";
  if (min < 60) return `${min} daq oldin`;
  const h = Math.round(min / 60);
  if (h < 24) return `${h} soat oldin`;
  return `${Math.round(h / 24)} kun oldin`;
}

// Re-exported from here because a dozen screens already import it from this
// module; the implementation lives in lib/format.ts with the other formatters.
export { formatDateTime } from "./format";
