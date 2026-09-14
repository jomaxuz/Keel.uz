package uz.keel.owner

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.keel.owner.data.AdminStats
import uz.keel.owner.data.CheckDetail
import uz.keel.owner.data.LossAlerts
import uz.keel.owner.data.MoneyPosition
import uz.keel.owner.data.Order
import uz.keel.owner.data.ShoppingList
import uz.keel.owner.data.opensCheck

// The models, fed the shape the server actually sends.
//
// ⚠️ **This exists because a renamed field is silent here and loud nowhere.**
// `Models.kt` is a hand-written second copy of the panel's types; kotlinx fills
// a default for any key it does not find, so a wrong name is not an error, it
// is a confident zero on an owner's screen. Three of them shipped at once:
// best-seller prices all read 0 so'm (`revenue` for `total`), the shopping list
// was permanently empty (`lines` for `rows`), and the orders list refused to
// open at all — `customer` and `address` are objects on the wire and were
// declared as strings, which is the one version of this mistake that throws.
//
// ⚠️ **The JSON below is copied from the Go handlers, not invented.** A fixture
// written from the Kotlin side would agree with the Kotlin side by
// construction, and would have passed on the day every price was zero.

private val json = Json {
    ignoreUnknownKeys = true
    explicitNulls = false
    coerceInputValues = true
}

class WireShapeTest {
    /** `handlers/adminstats.go` → `topDish` and `statsPeriod`. */
    @Test
    fun `best sellers carry their money`() {
        val stats = json.decodeFromString<AdminStats>(
            """
            {"from":"2026-09-01","to":"2026-09-05",
             "period":{"orders":12,"revenue":480000,"pending":90000,"debt":0,
                       "avgOrder":40000,"paid":12,"cancelled":1,"delivered":11,
                       "cashTotal":300000},
             "top":[{"name":"Lag'mon","qty":7,"total":224000},
                    {"name":"Osh","qty":4,"total":128000}],
             "series":[{"date":"2026-09-05","orders":12,"revenue":480000}]}
            """.trimIndent(),
        )
        assertEquals(480000.0, stats.period.revenue, 0.0)
        assertEquals(2, stats.top.size)
        assertEquals(7, stats.top[0].qty)
        // The whole point: a number, not the default.
        assertEquals(224000.0, stats.top[0].total, 0.0)
    }

    /** `handlers/shoppinglist.go` → `shoppingGroup` and `shoppingRow`. */
    @Test
    fun `the shopping list is grouped by who you ring`() {
        val list = json.decodeFromString<ShoppingList>(
            """
            {"groups":[{"supplierId":"s1","name":"Anvar aka","phone":"+998901112233",
                        "rows":[{"ingredientId":"i1","name":"Kartoshka","unit":"kg",
                                 "onHand":2.5,"minQty":10,"suggested":7.5,
                                 "price":9000,"cost":67500}],
                        "cost":67500}],
             "cost":67500,"since":"2026-08-30"}
            """.trimIndent(),
        )
        assertEquals(1, list.groups.size)
        assertEquals("Anvar aka", list.groups[0].name)
        assertEquals(1, list.groups[0].rows.size)
        assertEquals(7.5, list.groups[0].rows[0].suggested, 0.0)
        assertEquals("kg", list.groups[0].rows[0].unit)
    }

    /** `models.Order` as `/admin/orders` returns it. */
    @Test
    fun `an order names its guest through the object the server sends`() {
        val orders = json.decodeFromString<List<Order>>(
            """
            [{"id":"o1","number":"A-104","status":"pending","type":"delivery",
              "customer":{"name":"Dilnoza","phone":"+998935550001"},
              "address":{"text":"Chilonzor 9, 12-uy","lat":41.28,"lng":69.2,
                         "comment":"domofon ishlamaydi"},
              "items":[{"name":"Lag'mon","qty":2,"price":32000}],
              "total":64000,"createdAt":"2026-09-05T09:05:00Z","comment":""}]
            """.trimIndent(),
        )
        assertEquals(1, orders.size)
        assertEquals("Dilnoza", orders[0].customer.name)
        assertEquals("+998935550001", orders[0].customer.phone)
        assertEquals("Chilonzor 9, 12-uy", orders[0].address.text)
        assertEquals(64000.0, orders[0].total, 0.0)
        assertEquals(2, orders[0].items[0].qty)
    }

    /** ⚠️ A dine-in order has no address object worth reading, and the list
     *  must still open — the failure that made this file worth writing was a
     *  missing shape, not a missing value. */
    @Test
    fun `an order with nothing in it still decodes`() {
        val orders = json.decodeFromString<List<Order>>(
            """[{"id":"o2","number":"A-105","status":"pending","type":"dinein",
                 "customer":{"name":"","phone":""},"address":{"text":"","lat":0,"lng":0},
                 "items":[],"total":0,"tableNumber":"7"}]""",
        )
        assertEquals("7", orders[0].tableNumber)
        assertEquals("", orders[0].address.text)
    }

    /** `models.LossAlert`. ⚠️ The same `refId` names a sale or a cash shift, and
     *  an unset one arrives as Go's zero ObjectID — non-empty. */
    @Test
    fun `only an alert about a sale opens a check`() {
        val list = json.decodeFromString<LossAlerts>(
            """
            {"alerts":[
              {"id":"a1","kind":"void_after_precheck","at":"2026-09-05T14:05:00Z",
               "by":"Aziz","amount":64000,"reason":"mehmon voz kechdi","subject":"Lag'mon",
               "refId":"66d9a1b2c3d4e5f601234567","number":"A-104","table":"6",
               "afterPrecheck":true},
              {"id":"a2","kind":"cash_short","at":"2026-09-05T23:10:00Z","by":"Dilnoza",
               "amount":30000,"refId":"66d9a1b2c3d4e5f607654321"},
              {"id":"a3","kind":"big_discount","at":"2026-09-05T20:00:00Z","amount":90000,
               "refId":"000000000000000000000000"}
            ]}
            """.trimIndent(),
        )
        assertEquals("A-104", list.alerts[0].number)
        assertEquals("6", list.alerts[0].table)
        assertTrue(list.alerts[0].afterPrecheck)
        assertTrue(list.alerts[0].opensCheck())
        assertFalse(list.alerts[1].opensCheck())
        assertFalse(list.alerts[2].opensCheck())
    }

    /** `handlers/adminchecks.go` → `checkDetail` (embeds `checkRow`). Times are
     *  moved into the restaurant's zone before sending, hence the offsets. */
    @Test
    fun `a check keeps its voided line and who took it off`() {
        val k = json.decodeFromString<CheckDetail>(
            """
            {"id":"66d9a1b2c3d4e5f601234567","number":"A-104","table":"6","guests":3,
             "server":"Aziz","closedBy":"Dilnoza",
             "openedAt":"2026-09-05T18:30:00+05:00","closedAt":"2026-09-05T20:15:00+05:00",
             "items":3,"subtotal":160000,"discount":16000,"service":14400,"servicePercent":10,
             "total":158400,"paymentMethod":"cash","open":false,
             "lines":[
               {"name":"Lag'mon","qty":2,"price":32000,"sum":64000,
                "options":[{"name":"Hajmi","choice":"Katta","priceDelta":5000}],
                "firedAt":"2026-09-05T18:35:00+05:00"},
               {"name":"Shashlik","qty":2,"price":48000,"sum":0,
                "voidedBy":"Aziz","voidReason":"mehmon voz kechdi",
                "voidedAt":"2026-09-05T19:50:00+05:00","wasted":true}
             ],
             "discounts":[{"name":"Tanish","kind":"percent","trigger":"manual","amount":16000,
                           "by":"Aziz","authBy":"Dilnoza","reason":"doimiy mehmon"}],
             "openedBy":"Aziz","precheckAt":"2026-09-05T19:40:00+05:00",
             "refund":{"at":"2026-09-05T21:00:00+05:00","by":"Dilnoza","reason":"sovuq",
                       "amount":20000,"method":"cash"},
             "fiscalSign":"123456789012"}
            """.trimIndent(),
        )
        assertEquals(2, k.lines.size)
        assertFalse(k.lines[0].voided)
        assertEquals("Katta", k.lines[0].options[0].choice)
        assertTrue(k.lines[1].voided)
        assertEquals(0.0, k.lines[1].sum, 0.0)
        assertEquals(48000.0, k.lines[1].price, 0.0)
        assertEquals("Dilnoza", k.discounts[0].authBy)
        assertEquals(20000.0, k.refund!!.amount, 0.0)
        assertEquals(158400.0, k.total, 0.0)
        assertEquals(10, k.servicePercent)
    }

    /** `models.MoneyPosition` as `/admin/money` returns it. */
    @Test
    fun `money comes back as three totals and says what was counted`() {
        val m = json.decodeFromString<MoneyPosition>(
            """
            {"cash":[{"kind":"safe","name":"Seyf","amount":-50000,"at":"2026-09-05T10:00:00Z"},
                     {"kind":"drawer","name":"Kassa yashigi","amount":1250000,"note":"Dilnoza",
                      "at":"2026-09-05T09:00:00Z"}],
             "bank":[{"kind":"bank","name":"Kapitalbank","amount":42000000,"counted":true,
                      "at":"2026-09-01T00:00:00Z"}],
             "rails":[{"kind":"rail","name":"Uzum Tezkor","amount":3100000,"note":"2026-08-31"}],
             "cashTotal":1200000,"bankTotal":42000000,"railsTotal":3100000,
             "cashLimit":1000000,"overLimit":true,"branchId":"66d9a1b2c3d4e5f601234500"}
            """.trimIndent(),
        )
        assertEquals(-50000.0, m.cash[0].amount, 0.0)
        assertEquals("Dilnoza", m.cash[1].note)
        assertTrue(m.bank[0].counted)
        assertEquals("2026-08-31", m.rails[0].note)
        assertEquals(1200000.0, m.cashTotal, 0.0)
        assertEquals(1000000.0, m.cashLimit, 0.0)
        assertTrue(m.overLimit)
    }

    /** ⚠️ Both shapes a time arrives in: `Z` from Mongo, `+05:00` from a handler
     *  that moved it into local time first. A failed parse is a blank, silently. */
    @Test
    fun `times parse with and without an offset`() {
        val z = uz.keel.owner.ui.screens.localTime("2026-09-05T14:05:00Z")
        val off = uz.keel.owner.ui.screens.localTime("2026-09-05T19:05:00+05:00")
        assertNotNull(z)
        assertNotNull(off)
        assertEquals(z!!.toInstant(), off!!.toInstant())
        assertNull(uz.keel.owner.ui.screens.localTime(""))
    }
}
