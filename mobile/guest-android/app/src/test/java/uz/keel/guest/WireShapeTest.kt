package uz.keel.guest

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.keel.guest.data.MenuGroup
import uz.keel.guest.data.MenuItem
import uz.keel.guest.data.Order
import uz.keel.guest.data.OrderQuote
import uz.keel.guest.data.Restaurant
import uz.keel.guest.data.RestaurantResponse
import uz.keel.guest.data.GeoPoint
import uz.keel.guest.data.deliveryQuoteBody
import uz.keel.guest.data.orderBody
import uz.keel.guest.data.pick
import uz.keel.guest.data.quoteBody
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive

// The models, fed the shape the server actually sends.
//
// ⚠️ **This exists because a renamed field is silent here and loud nowhere.**
// `data/Models.kt` is a hand-written second copy of the site's types, and
// kotlinx fills a default for any key it does not find — so a wrong name is not
// an error, it is a menu on which every dish costs nothing.
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

    /** `handlers/public.go` → `GetMenu`, grouped by category. */
    @Test
    fun `the menu arrives grouped, priced and translated`() {
        val groups = json.decodeFromString<List<MenuGroup>>(
            """[{"category":{"id":"c1","name":"Issiq taomlar","nameRu":"Горячее",
                             "slug":"issiq","sortOrder":1,"isActive":true,"imageUrl":""},
                 "items":[{"id":"m1","categoryId":"c1","name":"Osh","nameRu":"Плов",
                           "description":"qo'y go'shtli","descriptionRu":"с бараниной",
                           "price":45000,"oldPrice":null,"imageUrl":"/uploads/osh.jpg",
                           "isAvailable":true,"options":[]}]}]""",
        )
        val item = groups.single().items.single()
        assertEquals(45000.0, item.price, 0.0)
        // ⚠️ The base name is what a language with no translation falls back to,
        // and every screen asks through `pick` for exactly that reason.
        assertEquals("Плов", item.pick("ru"))
        assertEquals("Osh", item.pick("en"))
        assertEquals("Горячее", groups.single().category.pick("ru"))
    }

    /** ⚠️ **`oldPrice` is nullable and null is not zero.** Read as zero it would
     *  strike out "0 so'm" beside every ordinary dish on the menu. */
    @Test
    fun `an ordinary dish has no old price at all`() {
        val plain = json.decodeFromString<MenuItem>(
            """{"id":"m1","name":"Osh","price":45000,"oldPrice":null,"isAvailable":true}""",
        )
        assertNull(plain.oldPrice)
        assertFalse(plain.discounted)

        val cut = json.decodeFromString<MenuItem>(
            """{"id":"m2","name":"Lag'mon","price":38000,"oldPrice":45000,"isAvailable":true}""",
        )
        assertTrue(cut.discounted)
    }

    /** `handlers/public.go` → `GetMenuItem`, with the questions the kitchen
     *  needs answered. */
    @Test
    fun `option groups carry their choices and whether an answer is compulsory`() {
        val item = json.decodeFromString<MenuItem>(
            """{"id":"m1","name":"Pitsa","price":60000,"isAvailable":true,
                "options":[{"name":"O'lcham","nameRu":"Размер","required":true,
                            "multiple":false,
                            "choices":[{"name":"O'rta","priceDelta":0},
                                       {"name":"Katta","nameRu":"Большая","priceDelta":15000}]}]}""",
        )
        val group = item.options.single()
        assertTrue(group.required)
        assertFalse(group.multiple)
        assertEquals(15000.0, group.choices[1].priceDelta, 0.0)
        assertEquals("Большая", group.choices[1].pick("ru"))
    }

    /** `handlers/orders.go` → `OrderQuote`. ⚠️ Every figure on the checkout is
     *  one of these, and none is added up on the phone. */
    @Test
    fun `the quote carries the money, the refusals and the points`() {
        val q = json.decodeFromString<OrderQuote>(
            """{"subtotal":90000,"discounts":[],"discountTotal":9000,"deliveryFee":15000,
                "total":96000,"codeApplied":true,"available":true,"minOrder":50000,
                "payableSubtotal":81000,"belowMinimum":false,"branchId":"b1",
                "branchName":"Chilonzor","prepMinutes":35,"soldOut":["Lag'mon"],
                "pointsSpent":0,"pointsBalance":12000,"pointsMax":9000,"pointsEarn":900}""",
        )
        assertEquals(96000.0, q.total, 0.0)
        assertEquals(15000.0, q.deliveryFee, 0.0)
        assertTrue(q.codeApplied)
        // ⚠️ Known only once the branch is — the guest browsed one branch's menu
        // and a delivery may be taken by another entirely.
        assertEquals(listOf("Lag'mon"), q.soldOut)
        assertEquals(900.0, q.pointsEarn, 0.0)
    }

    /** ⚠️ **A refused quote still parses.** "We do not deliver here" arrives as
     *  a normal answer with `available:false`, not as an error — and an app that
     *  threw on it would tell the guest the app is broken instead of telling
     *  them the address is out of range. */
    @Test
    fun `an address out of range is an answer, not a failure`() {
        val q = json.decodeFromString<OrderQuote>(
            """{"subtotal":90000,"discountTotal":0,"deliveryFee":0,"total":90000,
                "codeApplied":false,"available":false,"minOrder":50000,
                "payableSubtotal":90000,"belowMinimum":false,
                "pointsSpent":0,"pointsBalance":0,"pointsMax":0,"pointsEarn":0}""",
        )
        assertFalse(q.available)
        assertEquals(0.0, q.deliveryFee, 0.0)
    }

    /** `handlers/orders.go` → `CreateOrder`, whose answer carries the bank link
     *  when the method is an online one. */
    @Test
    fun `a placed order carries its number and, for an online method, the link`() {
        val order = json.decodeFromString<Order>(
            """{"id":"o1","number":"CHL-A71-4509","status":"pending","type":"delivery",
                "items":[{"menuItemId":"m1","name":"Osh","price":45000,"qty":2,
                          "comment":"","options":[]}],
                "subtotal":90000,"deliveryFee":15000,"total":105000,
                "paymentMethod":"payme","deliveryZone":"Markaz","distanceKm":3.2,
                "createdAt":"2026-09-08T12:00:00Z","payUrl":"https://checkout.paycom.uz/abc"}""",
        )
        assertEquals("CHL-A71-4509", order.number)
        assertEquals(2, order.items.single().qty)
        assertEquals("https://checkout.paycom.uz/abc", order.payUrl)
    }

    /** ⚠️ **The profile is wrapped, and this test is here because it was read
     *  wrongly the first time.** `/restaurant` answers
     *  `{restaurant, brand, branch, design, …}` — decoding the top level as the
     *  restaurant itself compiles, parses, throws nothing, and leaves every
     *  field at its default: a nameless restaurant that is always open and sits
     *  on the null island off the coast of Africa, which the address picker
     *  would then open on.
     *
     *  ⚠️ **`isOpenNow` is on the wrapper**, because it is computed from the
     *  *branch's* hours rather than the company document — and read at the
     *  wrong level it defaults to `true`, telling a guest that a closed kitchen
     *  is taking orders. The JSON below is trimmed from what b5somsa.keel.uz
     *  actually returns. */
    @Test
    fun `the profile is wrapped, and the open flag sits on the wrapper`() {
        val res = json.decodeFromString<RestaurantResponse>(
            """{"restaurant":{"id":"r1","name":"B5 Somsa","description":"",
                  "logoUrl":"https://b5somsa.keel.uz/uploads/73fba65e.jpg",
                  "coverUrl":"","phones":["+998901112233"],
                  "address":{"lat":41.308868,"lng":69.165443,"text":"Kukcha"},
                  "currency":"UZS",
                  "theme":{"brand":"#2563eb","brandDark":"","font":"soft"}},
                "brand":{"name":"B5 Somsa"},
                "branch":{"name":"B5 Beshqayrag'och"},
                "designLocked":false,
                "isOpenNow":false}""",
        )
        assertEquals("B5 Somsa", res.restaurant.name)
        assertEquals(41.308868, res.restaurant.address.lat, 0.000001)
        assertEquals(69.165443, res.restaurant.address.lng, 0.000001)
        // ⚠️ Computed by the server from the branch's hours: a phone's clock is
        // the one thing this product never trusts.
        assertFalse(res.isOpenNow)
    }

    /** ⚠️ **Empty is 2GIS, and it has to stay that way.** Every install that
     *  predates the setting runs on 2GIS; reading the zero value as anything
     *  else would change the map under all of them at once — the same rule an
     *  empty `source` follows in the catalogue, and the one the site follows. */
    @Test
    fun `an unset map provider is 2gis, and each provider takes its own key`() {
        val plain = json.decodeFromString<Restaurant>("""{"name":"X"}""")
        assertEquals("2gis", plain.provider)

        val twogis = json.decodeFromString<Restaurant>(
            """{"name":"X","mapProvider":"","mapApiKey":"GIS","mapYandexKey":"Y",
                "mapGoogleKey":"G"}""",
        )
        assertEquals("GIS", twogis.mapKey)

        val yandex = json.decodeFromString<Restaurant>(
            """{"name":"X","mapProvider":"yandex","mapApiKey":"GIS","mapYandexKey":"Y",
                "mapGoogleKey":"G"}""",
        )
        assertEquals("yandex", yandex.provider)
        assertEquals("Y", yandex.mapKey)

        // ⚠️ Google takes the site's own `mapGoogleKey` — there is no separate
        // Android field, and a field that no longer exists must not come back
        // as a silently empty key.
        val google = json.decodeFromString<Restaurant>(
            """{"name":"X","mapProvider":"google","mapGoogleKey":"G"}""",
        )
        assertEquals("G", google.mapKey)
    }

    // ---- Whole so'm on the wire ----
    //
    // ⚠️ **This is the bug that made the basket show a red error and nothing
    // else.** Money is `Double` in the app and `int` on the server; kotlinx
    // writes 5000.0 with the point, and Go refuses the *whole request* —
    // "so'rov formati noto'g'ri", no field named. `usePoints: 0.0` alone did
    // it, so every quote failed, empty options or not.
    //
    // ⚠️ Walks the tree rather than checking the fields it knows: the next
    // money field somebody adds is exactly the one nobody remembers to test.
    // Latitude and longitude are the only fractions allowed through, because
    // they are the only fractions the server declares as floats.
    @Test
    fun `no money field goes on the wire with a decimal point`() {
        val line = CartLine(
            menuItemId = "m1",
            name = "Lag'mon",
            price = 32000.0,
            qty = 2,
            options = listOf(ChosenOption("O'lcham", "Katta", 5000.0)),
        )
        val point = GeoPoint(lat = 41.31, lng = 69.24, text = "Amir Temur 1")

        assertWhole(deliveryQuoteBody(41.31, 69.24, 64000.0))
        assertWhole(quoteBody(listOf(line), "delivery", point, "", 0.0))
        assertWhole(
            orderBody(
                listOf(line), "delivery", "Ali", "+998901234567", point,
                "eshik oldiga", "cash", "", 1500.0,
            ),
        )
    }

    /** Every number in the tree is an integer, `lat`/`lng` excepted. */
    private fun assertWhole(el: JsonElement, path: String = "") {
        when (el) {
            is JsonObject -> el.forEach { (k, v) -> assertWhole(v, if (path.isEmpty()) k else "$path.$k") }
            is JsonArray -> el.forEachIndexed { i, v -> assertWhole(v, "$path[$i]") }
            is JsonPrimitive -> {
                val name = path.substringAfterLast('.')
                if (el.isString || name == "lat" || name == "lng") return
                assertFalse(
                    "$path went out as ${el.content} — Go refuses a decimal into an int field",
                    el.content.contains('.'),
                )
            }
        }
    }

    /** ⚠️ **The restaurant's own colour, read at run time.** The build bakes one
     *  in for the splash and the icon, but that is a photograph of the panel on
     *  the day it ran — and this field was missing from the model entirely, so
     *  every app fell back to Keel's orange and wore a stranger's colour on a
     *  guest's home screen. */
    @Test
    fun `the restaurant's accent comes off the profile`() {
        val themed = json.decodeFromString<Restaurant>(
            """{"name":"X","theme":{"brand":"#2563eb","brandDark":"","font":"soft"}}""",
        )
        assertEquals("#2563eb", themed.accentHex)

        // Empty means "the build's own", never Keel's orange by itself.
        assertEquals("", json.decodeFromString<Restaurant>("""{"name":"X"}""").accentHex)
    }
}