package uz.keel.courier

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.keel.courier.data.Order
import uz.keel.courier.data.OrderAddress
import uz.keel.courier.i18n.UZ
import uz.keel.courier.location.Fix
import uz.keel.courier.location.MAX_FIX_AGE_MS
import uz.keel.courier.location.arrivalGate
import uz.keel.courier.location.metersBetween

// The rule that decides whether the "delivered" button is pressable.
//
// ⚠️ **The server decides; this decides what the button looks like** — and the
// two have to agree, or the app is the thing that appears broken. A courier
// shown an open button and then refused is standing in front of a customer
// reading a failure they cannot act on. So this pins the three escapes, the
// staleness window and the direction of the inequality, all of which are copied
// from `arrivalBlocked` in `internal/handlers/courier.go`.

private const val LAT = 41.311081
private const val LNG = 69.240562

private fun order(
    type: String = "delivery",
    lat: Double? = LAT,
    lng: Double? = LNG,
) = Order(
    id = "1",
    number = "41",
    status = "on_the_way",
    type = type,
    address = OrderAddress(text = "Amir Temur 1", lat = lat, lng = lng),
)

private fun fix(lat: Double = LAT, lng: Double = LNG, ageMs: Long = 0) =
    Fix(lat, lng, 12.0, System.currentTimeMillis() - ageMs)

class GateTest {

    /** ⚠️ The same haversine the Go handler runs. A hundred metres north of a
     *  point in Tashkent is a hundred metres, not ninety or a hundred and ten —
     *  and a formula that drifted would open the button on the wrong street. */
    @Test
    fun `metres are metres`() {
        // ~111.3 km per degree of latitude.
        val d = metersBetween(LAT, LNG, LAT + 0.0009, LNG)
        assertEquals(100.0, d, 2.0)
    }

    @Test
    fun `at the door the button is open`() {
        val g = arrivalGate(order(), fix(), radiusM = 150, t = UZ)
        assertTrue(g.applies)
        assertNull(g.reason)
        assertEquals(0, g.meters)
    }

    @Test
    fun `too far names the distance and the radius`() {
        // ⚠️ Both numbers, because "you are too far" tells a courier nothing
        // they can act on: 300 against 150 says "walk to the door", and 3000
        // says "you are at the wrong address".
        val g = arrivalGate(order(), fix(lat = LAT + 0.0027), radiusM = 150, t = UZ)
        assertTrue(g.applies)
        assertTrue(g.meters!! > 250)
        assertTrue(g.reason!!.contains("150"))
    }

    /** ⚠️ **The three escapes the server allows, in the same order.** Getting
     *  any of them wrong shuts a button the server would have accepted — on a
     *  pickup order, on one typed over the phone with no map point, or in a
     *  village where the restaurant switched the rule off. */
    @Test
    fun `the check does not apply where the server does not apply it`() {
        assertFalse(arrivalGate(order(type = "pickup"), fix(), 150, UZ).applies)
        assertFalse(arrivalGate(order(lat = null, lng = null), fix(), 150, UZ).applies)
        assertFalse(arrivalGate(order(), fix(), radiusM = 0, t = UZ).applies)
    }

    /** ⚠️ **A fix from a pocket is not a position.** The server refuses one
     *  older than ten minutes, so the button has to as well — otherwise it opens
     *  and the press is refused, which reads as a broken app. */
    @Test
    fun `a stale fix shuts the button rather than measuring from it`() {
        val g = arrivalGate(order(), fix(ageMs = MAX_FIX_AGE_MS + 1000), 150, UZ)
        assertTrue(g.applies)
        assertNull(g.meters)
        assertEquals(UZ.gate.stale, g.reason)
    }

    @Test
    fun `no fix at all says so rather than saying too far`() {
        val g = arrivalGate(order(), null, 150, UZ)
        assertEquals(UZ.gate.noFix, g.reason)
    }
}
