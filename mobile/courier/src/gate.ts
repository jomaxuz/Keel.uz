import type { Order } from "@/lib/types";

import type { Dict } from "./i18n";
import { metersBetween, type Fix } from "./tracking";

// May this order be marked delivered?
//
// ⚠️ **The server decides; this only decides what the button looks like.**
// `CourierAdvanceOrder` runs the same check with the courier's last *reported*
// position (`arrivalBlocked` in `internal/handlers/courier.go`), and it is the
// authority — a phone can lie about where it is, and the whole point of the
// rule is that "delivered" means the food arrived.
//
// ⚠️ **So the two have to agree, or the app is the thing that looks broken.**
// A courier shown an open button and then refused by the server is standing in
// front of a customer reading a failure they cannot act on. Hence: the same
// haversine, the same radius, and the same ten-minute staleness window — and
// when this one is *stricter* than the server (it uses the fix the phone has
// now, which is newer than the one the server has), the button opens a moment
// later rather than a moment too early. That is the safe direction.

/** How old a fix may be before it stops counting.
 *
 *  ⚠️ Matches `maxLocationAge` on the server. A phone that has been in a
 *  pocket for twenty minutes knows where it was, not where it is, and a
 *  delivery closed from that position could be a street away. */
export const MAX_FIX_AGE_MS = 10 * 60 * 1000;

export interface Gate {
  /** True when the check applies to this order at all. */
  applies: boolean;
  /** Metres to the customer, or null when there is no usable fix. */
  meters: number | null;
  /** Why the button is shut, in the courier's language, or null when it is
   *  open. */
  reason: string | null;
}

export function arrivalGate(
  order: Order,
  fix: Fix | null,
  radiusM: number,
  t: Dict,
  now = Date.now(),
): Gate {
  // ⚠️ The same three escapes the server allows, in the same order. A pickup
  // order has no address to arrive at; an order placed without a map point
  // cannot be measured; and a restaurant that sets the radius to 0 has switched
  // the rule off — usually a village where "the address" is a landmark.
  const off =
    order.type !== "delivery" ||
    !order.address?.lat ||
    !order.address?.lng ||
    radiusM <= 0;
  if (off) return { applies: false, meters: null, reason: null };

  if (!fix) return { applies: true, meters: null, reason: t.gate.noFix };
  if (now - fix.at > MAX_FIX_AGE_MS) {
    return { applies: true, meters: null, reason: t.gate.stale };
  }

  const meters = Math.round(
    metersBetween(fix, { lat: order.address.lat, lng: order.address.lng }),
  );
  return {
    applies: true,
    meters,
    reason: meters > radiusM ? t.gate.tooFar(meters, radiusM) : null,
  };
}
