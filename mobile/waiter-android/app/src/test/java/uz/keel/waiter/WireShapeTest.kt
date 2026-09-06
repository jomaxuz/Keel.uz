package uz.keel.waiter

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.keel.waiter.data.BranchInfo
import uz.keel.waiter.data.Check
import uz.keel.waiter.data.ChecksResponse
import uz.keel.waiter.data.LoginResponse
import uz.keel.waiter.data.MenuGroup
import uz.keel.waiter.data.StaffMe
import uz.keel.waiter.data.StaffReport

// The models, fed the shape the server actually sends.
//
// ⚠️ **This exists because a renamed field is silent here and loud nowhere.**
// `Models.kt` is a hand-written second copy of the panel's types, and kotlinx
// fills a default for any key it does not find — so a wrong name is not an
// error, it is a confident zero on a bill somebody is about to hand a guest.
// The Expo build shipped three of these at once, and this application had no
// test of any kind until it was the last of the five without one.
//
// ⚠️ **The JSON below is copied from the Go handlers** (`handlers/till.go` →
// `checkView` and `checkLine`), never invented here: a fixture written from the
// Kotlin side agrees with the Kotlin side by construction, and would have
// passed on the day every price read zero.

private val json = Json {
    ignoreUnknownKeys = true
    explicitNulls = false
    coerceInputValues = true
}

class WireShapeTest {

    /** `handlers/till.go` → `StaffLogin`. */
    @Test
    fun `signing in carries the token and what the account may do`() {
        val res = json.decodeFromString<LoginResponse>(
            """{"token":"ey.J.staff",
                "staff":{"id":"s1","name":"Dilnoza","username":"dil","position":"ofitsiant",
                         "perms":["waiter","void"],"canWaiter":true,"canKitchen":false}}""",
        )
        assertEquals("ey.J.staff", res.token)
        assertEquals("Dilnoza", res.staff.name)
    }

    /** `handlers/till.go` → `StaffMe`. */
    @Test
    fun `an open shift is present or absent, never a zero`() {
        val open = json.decodeFromString<StaffMe>(
            """{"staff":{"id":"s1","name":"Dilnoza"},
                "openShift":{"id":"sh1","date":"2026-09-06","minutes":0}}""",
        )
        assertTrue(open.openShift != null)
        assertNull(json.decodeFromString<StaffMe>("""{"staff":{"id":"s1","name":"D"}}""").openShift)
    }

    /** `handlers/till.go` → `checkView` and `checkLine`.
     *
     *  ⚠️ **`fired` and `unfired` are the two the whole screen is built on**: what
     *  the kitchen has versus what is still a draft. Read as false and zero, a
     *  waiter would send a table's food twice — or think they had. */
    @Test
    fun `a check carries what is fired, what is not, and what it comes to`() {
        val res = json.decodeFromString<ChecksResponse>(
            """{"checks":[{"id":"c1","number":"41","status":"open","tableId":"t1",
                  "tableNumber":"6","guests":3,"serverId":"s1","serverName":"Dilnoza",
                  "openedAt":"2026-09-06T18:02:00Z","openMin":34,
                  "lines":[{"lineId":"l1","menuItemId":"m1","name":"Lag'mon","price":48000,
                            "qty":2,"sum":96000,"fired":true,
                            "readyAt":"2026-09-06T18:20:00Z"},
                           {"lineId":"l2","menuItemId":"m2","name":"Choy","price":8000,
                            "qty":1,"sum":8000,"fired":false,"portion":50,
                            "comment":"achchiq emas"}],
                  "subtotal":104000,"unfired":1,"readyWaiting":1,"served":0,
                  "service":10400,"servicePercent":10,"total":114400}],
                "soldOut":["m9"]}""",
        )
        val c = res.checks.single()
        assertEquals("41", c.number)
        assertEquals("6", c.tableNumber)
        assertEquals(3, c.guests)
        assertEquals(1, c.unfired)
        assertEquals(1, c.readyWaiting)
        assertEquals(114400.0, c.total, 0.0)
        assertTrue(c.lines[0].fired)
        assertFalse(c.lines[1].fired)
        // ⚠️ Half a portion is 50, and absent is a whole one — not zero, which
        // would be a line worth nothing.
        assertEquals(50, c.lines[1].portion)
        assertNull(c.lines[0].portion)
        assertEquals("achchiq emas", c.lines[1].comment)
        // What the kitchen has run out of, so a dish cannot be sent again.
        assertEquals(listOf("m9"), res.soldOut)
    }

    /** ⚠️ **`void` is a Kotlin keyword and the wire's own name.** The property is
     *  `voided` behind a `@SerialName`, which is precisely the sort of rename a
     *  test has to pin: nothing else would notice it going missing, and a
     *  voided line drawn as an ordinary one is a dish somebody is charged for. */
    @Test
    fun `a voided line keeps who voided it and why`() {
        val c = json.decodeFromString<Check>(
            """{"id":"c1","number":"41","status":"open","openedAt":"2026-09-06T18:02:00Z",
                "lines":[{"lineId":"l1","name":"Osh","price":40000,"qty":1,"sum":0,
                          "fired":true,
                          "void":{"at":"2026-09-06T18:40:00Z","by":"Aziz","reason":"mehmon rad etdi"}}],
                "subtotal":0,"unfired":0,"total":0}""",
        )
        val v = c.lines.single().voided
        assertTrue(v != null)
        assertEquals("Aziz", v!!.by)
        assertEquals("mehmon rad etdi", v.reason)
    }

    /** `handlers/till.go` → the branch a till is bound to, with its floor. */
    @Test
    fun `the floor arrives with its zones, and a table without one is not lost`() {
        val b = json.decodeFromString<BranchInfo>(
            """{"id":"b1","name":"Chilonzor","currency":"UZS","servicePercent":10,
                "booking":{"enabled":true,
                  "zones":[{"id":"z1","name":"Zal","bookable":true,"sort":0}],
                  "tables":[{"id":"t1","number":"6","seats":4,"isActive":true,"zoneId":"z1"},
                            {"id":"t2","number":"7","seats":2,"isActive":true}]}}""",
        )
        assertEquals(1, b.booking.zones.size)
        assertEquals(2, b.booking.tables.size)
        // ⚠️ Empty means the default zone — every table drawn before zones
        // existed has none, and a screen that filtered them out would show an
        // empty room to exactly those restaurants.
        assertNull(b.booking.tables[1].zoneId)
        assertEquals(10.0, b.servicePercent, 0.0)
    }

    /** `handlers/menu.go` → the menu the floor screen orders from. */
    @Test
    fun `the menu arrives grouped, with the prices on the items`() {
        val groups = json.decodeFromString<List<MenuGroup>>(
            """[{"category":{"id":"k1","name":"Issiq taomlar","sort":1,"isActive":true},
                 "items":[{"id":"m1","name":"Lag'mon","price":48000,"isActive":true,
                           "categoryId":"k1"}]}]""",
        )
        assertEquals("Issiq taomlar", groups.single().category.name)
        assertEquals(48000.0, groups.single().items.single().price, 0.0)
    }

    /** `handlers/staffreport.go` → the waiter's own hours. */
    @Test
    fun `the report is drawn, never recomputed`() {
        val r = json.decodeFromString<StaffReport>(
            """{"days":[{"date":"2026-09-01","weekday":1,"expected":600,"worked":615,
                         "diff":15,"open":false,"status":"over"}],
                "totals":{"worked":615,"expected":600,"diff":15},
                "today":{"current":615,"previous":600},
                "week":{"current":615,"previous":0},
                "month":{"current":615,"previous":0}}""",
        )
        assertEquals("over", r.days.single().status)
        assertEquals(615, r.totals.worked)
    }
}
