package uz.keel.owner

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Test
import uz.keel.owner.data.AdminStats
import uz.keel.owner.data.Order
import uz.keel.owner.data.ShoppingList

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
}
