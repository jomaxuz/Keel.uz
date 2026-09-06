package uz.keel.courier.location

import uz.keel.courier.data.Order
import uz.keel.courier.i18n.Dict

// May this order be marked delivered?
//
// ⚠️ **The server decides; this only decides what the button looks like.**
// `CourierAdvanceOrder` runs the same check with the courier's last *reported*
// position (`arrivalBlocked` in `internal/handlers/courier.go`), and it is the
// authority — a phone can lie about where it is, and the whole point of the rule
// is that "delivered" means the food arrived.
//
// ⚠️ **So the two have to agree, or the app is the thing that looks broken.** A
// courier shown an open button and then refused by the server is standing in
// front of a customer reading a failure they cannot act on. Hence: the same
// haversine, the same radius — read from the server, never invented — and the
// same ten-minute staleness window. Where this one is *stricter* than the server
// (it uses the fix the phone has now, which is newer than the one the server
// has) the button opens a moment later rather than a moment too early. That is
// the safe direction.

/** How old a fix may be before it stops counting.
 *
 *  ⚠️ Matches `maxLocationAge` on the server. A phone that has been in a pocket
 *  for twenty minutes knows where it was, not where it is, and a delivery closed
 *  from that position could be a street away. */
const val MAX_FIX_AGE_MS = 10 * 60 * 1000L

data class Gate(
    /** True when the check applies to this order at all. */
    val applies: Boolean,
    /** Metres to the customer, or null when there is no usable fix. */
    val meters: Int?,
    /** Why the button is shut, in the courier's language, or null when it is
     *  open. */
    val reason: String?,
)

fun arrivalGate(
    order: Order,
    fix: Fix?,
    radiusM: Int,
    t: Dict,
    now: Long = System.currentTimeMillis(),
): Gate {
    val lat = order.address.lat
    val lng = order.address.lng
    // ⚠️ The same three escapes the server allows, in the same order. A pickup
    // order has no address to arrive at; an order placed without a map point
    // cannot be measured; and a restaurant that sets the radius to 0 has
    // switched the rule off — usually a village where the address is a landmark.
    val off = order.type != "delivery" || lat == null || lng == null || radiusM <= 0
    if (off) return Gate(applies = false, meters = null, reason = null)

    if (fix == null) return Gate(true, null, t.gate.noFix)
    if (now - fix.at > MAX_FIX_AGE_MS) return Gate(true, null, t.gate.stale)

    val meters = Math.round(metersBetween(fix.lat, fix.lng, lat, lng)).toInt()
    return Gate(
        applies = true,
        meters = meters,
        reason = if (meters > radiusM) t.gate.tooFar(meters, radiusM) else null,
    )
}
