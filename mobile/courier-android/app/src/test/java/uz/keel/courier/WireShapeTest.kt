package uz.keel.courier

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.keel.courier.data.Courier
import uz.keel.courier.data.CourierOrderRow
import uz.keel.courier.data.CourierStats
import uz.keel.courier.data.LoginResult
import uz.keel.courier.data.Order
import uz.keel.courier.data.RestaurantEnvelope

// The models, fed the shape the server actually sends.
//
// ⚠️ **This exists because a renamed field is silent here and loud nowhere.**
// `Models.kt` is a hand-written second copy of the panel's types, and kotlinx
// fills a default for any key it does not find — so a wrong name is not an
// error, it is a confident zero on a card a courier is about to collect cash
// against. The waiter build shipped three of those at once.
//
// ⚠️ **The JSON below is copied from the Go handlers**, never invented here: a
// fixture written from the Kotlin side agrees with the Kotlin side by
// construction, and would have passed on the day every price read zero.

private val json = Json {
    ignoreUnknownKeys = true
    explicitNulls = false
    coerceInputValues = true
}

class WireShapeTest {

    /** `handlers/courier.go` → `CourierLogin`. */
    @Test
    fun `signing in carries the token and who signed in`() {
        val res = json.decodeFromString<LoginResult>(
            """{"token":"ey.J.courier",
                "courier":{"id":"c1","name":"Aziz","phone":"998901112233",
                           "username":"aziz","status":"free","vehicle":"moto",
                           "isActive":true}}""",
        )
        assertEquals("ey.J.courier", res.token)
        assertEquals("Aziz", res.courier.name)
        assertEquals("free", res.courier.status)
    }

    /** `handlers/courier.go` → `CourierOrders`.
     *
     *  ⚠️ **`customer` and `address` are objects on the wire.** Declaring either
     *  as a string is the one version of this mistake that throws rather than
     *  reading zero — on the waiter build it was the whole orders list refusing
     *  to open. */
    @Test
    fun `an order carries the address as an object with a point`() {
        val list = json.decodeFromString<List<Order>>(
            """[{"id":"o1","number":"41","status":"on_the_way","type":"delivery",
                 "customer":{"name":"Dilnoza","phone":"998901234567"},
                 "address":{"text":"Amir Temur 1","comment":"3-qavat",
                            "lat":41.311081,"lng":69.240562},
                 "items":[{"name":"Lag'mon","qty":1,"comment":"achchiq emas"}],
                 "total":48000,"paymentMethod":"cash"}]""",
        )
        val o = list.single()
        assertEquals("Dilnoza", o.customer.name)
        assertEquals(41.311081, o.address.lat!!, 1e-6)
        assertEquals("achchiq emas", o.items.single().comment)
        assertEquals(48000.0, o.total, 0.0)
    }

    /** ⚠️ **An order with no map point must decode, not throw.** One typed over
     *  the phone has no coordinates at all, and it is exactly the order the
     *  arrival check has to be switched off for. */
    @Test
    fun `an order without a point is an order`() {
        val o = json.decodeFromString<Order>(
            """{"id":"o2","number":"42","status":"confirmed","type":"delivery",
                "customer":{"name":"Bek","phone":"998900000000"},
                "address":{"text":"Chilonzor 5"},
                "items":[],"total":30000,"paymentMethod":"card"}""",
        )
        assertNull(o.address.lat)
        assertTrue(o.address.text.isNotEmpty())
    }

    /** `handlers/courier.go` → `CourierStats`. */
    @Test
    fun `cash in hand is its own number, not the day's takings`() {
        val s = json.decodeFromString<CourierStats>(
            """{"today":{"orders":6,"earnings":90000,"cash":240000,"total":420000},
                "week":{"orders":31,"earnings":465000,"cash":1200000,"total":2100000},
                "month":{"orders":120,"earnings":1800000,"cash":4400000,"total":8200000},
                "all":{"orders":900,"earnings":13500000,"cash":31000000,"total":61000000},
                "active":2,"cashInHand":240000,"cashSettled":30760000}""",
        )
        assertEquals(90000.0, s.today.earnings, 0.0)
        // ⚠️ The number the earnings screen exists for: everything collected
        // less everything handed back. `all.cash` alone only ever grows.
        assertEquals(240000.0, s.cashInHand, 0.0)
    }

    /** `handlers/courier.go` → `CourierHistory`. */
    @Test
    fun `a delivered row carries what it earned and when`() {
        val rows = json.decodeFromString<List<CourierOrderRow>>(
            """[{"id":"o1","number":"41","total":48000,"paymentMethod":"cash",
                 "earned":15000,"deliveredAt":"2026-09-06T18:22:00Z",
                 "address":{"text":"Amir Temur 1"}}]""",
        )
        assertEquals(15000.0, rows.single().earned, 0.0)
        assertEquals("2026-09-06T18:22:00Z", rows.single().deliveredAt)
    }

    /** `handlers/restaurant.go` → the delivery block.
     *
     *  ⚠️ **Only the radius is read.** The profile carries addresses, keys and
     *  delivery settings, and parsing the rest would be a second copy of the
     *  panel's largest type kept up to date for nothing. */
    @Test
    fun `the arrival radius comes off a profile full of things this app ignores`() {
        val env = json.decodeFromString<RestaurantEnvelope>(
            """{"restaurant":{"name":"Osh","phone":"998901112233",
                 "delivery":{"enabled":true,"mode":"zones","minOrder":50000,
                             "arrivalRadiusM":150,"baseFee":10000}}}""",
        )
        assertEquals(150, env.restaurant.delivery.arrivalRadiusM)
    }

    /** ⚠️ A field the server has not sent yet must read as "off", never throw:
     *  a restaurant on an older build has no arrival radius, and the check is
     *  simply not applied there. */
    @Test
    fun `a profile with no radius reads as no check`() {
        val env = json.decodeFromString<RestaurantEnvelope>("""{"restaurant":{"name":"Osh"}}""")
        assertEquals(0, env.restaurant.delivery.arrivalRadiusM)
    }

    @Test
    fun `a courier row survives a status this build has not heard of`() {
        // ⚠️ Text rather than an enum: a value from a newer panel must not throw
        // while parsing the answer that would have explained it.
        val c = json.decodeFromString<Courier>("""{"id":"c1","name":"Aziz","status":"paused"}""")
        assertEquals("paused", c.status)
    }
}
