package uz.keel.team

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.keel.team.data.AdvanceBalance
import uz.keel.team.data.BuyCatalog
import uz.keel.team.data.BuyResult
import uz.keel.team.data.LoginResult
import uz.keel.team.data.ShoppingDraft
import uz.keel.team.data.ShoppingList
import uz.keel.team.data.ShoppingOrders
import uz.keel.team.data.Staff
import uz.keel.team.data.StaffMe
import uz.keel.team.data.StaffReport
import uz.keel.team.ui.screens.canWriteHere

// The models, fed the shape the server actually sends.
//
// ⚠️ **This exists because a renamed field is silent here and loud nowhere.**
// `Models.kt` is a hand-written second copy of the panel's types, and kotlinx
// fills a default for any key it does not find — so a wrong name is not an
// error, it is a month of somebody's hours reading zero on the screen they check
// their pay against.
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

    /** `handlers/staff.go` → `StaffLogin`. */
    @Test
    fun `signing in carries the token and the permissions`() {
        val res = json.decodeFromString<LoginResult>(
            """{"token":"ey.J.staff",
                "staff":{"id":"s1","name":"Aziz","phone":"998901112233",
                         "username":"aziz","position":"oshpaz",
                         "perms":["buy","stock"],"isActive":true}}""",
        )
        assertEquals("ey.J.staff", res.token)
        // ⚠️ The permissions decide which tabs exist. Read as an empty list, a
        // buyer would open the app and find the market screen gone.
        assertEquals(listOf("buy", "stock"), res.staff.perms)
    }

    /** `handlers/staff.go` → `StaffMe`. */
    @Test
    fun `an open shift is the difference between two words on one button`() {
        val open = json.decodeFromString<StaffMe>(
            """{"staff":{"id":"s1","name":"Aziz"},
                "openShift":{"id":"sh1","date":"2026-09-06","in":"09:02","minutes":0}}""",
        )
        assertTrue(open.openShift != null)
        val closed = json.decodeFromString<StaffMe>("""{"staff":{"id":"s1","name":"Aziz"}}""")
        assertTrue(closed.openShift == null)
    }

    /** `handlers/staffreport.go` → `StaffReport`. */
    @Test
    fun `the report carries the days, their status and the totals`() {
        val r = json.decodeFromString<StaffReport>(
            """{"days":[{"date":"2026-09-01","weekday":1,"expected":600,"worked":615,
                         "diff":15,"sessions":[],"first":"09:02","last":"19:17",
                         "planStart":"09:00","planEnd":"19:00","open":false,"status":"over","pay":0},
                        {"date":"2026-09-02","weekday":2,"expected":600,"worked":0,
                         "diff":-600,"sessions":[],"first":"","last":"",
                         "planStart":"09:00","planEnd":"19:00","open":false,"status":"absent","pay":0}],
                "totals":{"days":2,"absent":1,"expected":1200,"worked":615,"diff":-585,
                          "overtime":15,"shortage":600,"pay":0},
                "from":"2026-09-01","to":"2026-09-30",
                "today":{"current":615,"previous":600,"percent":2,"hasPrev":true},
                "week":{"current":615,"previous":0,"percent":0,"hasPrev":false},
                "month":{"current":615,"previous":0,"percent":0,"hasPrev":false},
                "periodPay":1450000,"periodPaid":0,"periodDue":1450000}""",
        )
        assertEquals(2, r.days.size)
        assertEquals("over", r.days[0].status)
        assertEquals(615, r.totals.worked)
        assertEquals(-585, r.totals.diff)
        assertEquals(1450000.0, r.periodPay, 0.0)
    }

    /** `handlers/staffbuy.go` → the shortage list and the catalogue. */
    @Test
    fun `the shortage list is grouped by who you ring, and flattened here`() {
        val list = json.decodeFromString<ShoppingList>(
            """{"groups":[{"supplierId":"s1","name":"Anvar aka","phone":"+998901112233",
                           "rows":[{"ingredientId":"i1","name":"Kartoshka","unit":"kg",
                                    "onHand":2.5,"minQty":10,"suggested":7.5,
                                    "price":6000,"cost":45000}],"cost":45000}],
                "cost":45000,"since":null}""",
        )
        val row = list.groups.single().rows.single()
        assertEquals("Kartoshka", row.name)
        assertEquals(7.5, row.suggested, 0.0)
        assertEquals(6000.0, row.price, 0.0)
    }

    /** ⚠️ **The last price is the only guard the typed one has.** Read as zero,
     *  the box beside the price would be empty and `90 000` would look like an
     *  ordinary number. */
    @Test
    fun `the catalogue carries the last price and the market packaging`() {
        val cat = json.decodeFromString<BuyCatalog>(
            """{"ingredients":[{"id":"i1","name":"Yalpiz","unit":"kg","lastPrice":9000,
                                "packName":"bog'lam","packQty":0.05}]}""",
        )
        val row = cat.ingredients.single()
        assertEquals(9000.0, row.lastPrice, 0.0)
        assertEquals("bog'lam", row.packName)
        assertEquals(0.05, row.packQty, 0.0)
    }

    /** ⚠️ **Negative is a real answer.** A buyer who paid for the last crate
     *  themselves is owed money, and a purse clamped at zero would be silent
     *  about exactly the debt they are waiting on. */
    @Test
    fun `the purse can be negative`() {
        val b = json.decodeFromString<AdvanceBalance>(
            """{"staffId":"s1","staffName":"Aziz","issued":500000,"returned":0,
                "spent":620000,"balance":-120000}""",
        )
        assertEquals(-120000.0, b.balance, 0.0)
    }

    /** `handlers/buyorders.go` → one sent list, with what came back. */
    @Test
    fun `asked and brought are two fields, and missing is a third`() {
        val orders = json.decodeFromString<ShoppingOrders>(
            """{"orders":[{"id":"o1","forDate":"2026-09-07","status":"sent",
                 "createdBy":"Dilnoza","createdAt":"2026-09-06T18:00:00Z",
                 "lines":[{"id":"l1","ingredientId":"i1","name":"Kartoshka","unit":"kg",
                           "qty":10,"gotQty":6,"price":6500,"gotAt":"2026-09-07T06:20:00Z"},
                          {"id":"l2","name":"Rayhon","unit":"kg","qty":1,"missing":true}]}]}""",
        )
        val o = orders.orders.single()
        assertEquals("Dilnoza", o.createdBy)
        assertEquals(10.0, o.lines[0].qty, 0.0)
        assertEquals(6.0, o.lines[0].gotQty, 0.0)
        assertTrue(o.lines[0].gotAt.isNotEmpty())
        // ⚠️ Its own answer, not a quantity of zero: a line nobody touched and
        // one somebody looked for and could not find are different facts.
        assertTrue(o.lines[1].missing)
        assertTrue(o.lines[1].gotAt.isEmpty())
    }

    /** `handlers/buyorders.go` → the draft a list is written from. */
    @Test
    fun `the draft carries the shortage and the whole catalogue`() {
        val d = json.decodeFromString<ShoppingDraft>(
            """{"rows":[{"ingredientId":"i1","name":"Kartoshka","unit":"kg","qty":7.5,"onHand":2.5}],
                "since":null,
                "catalog":[{"ingredientId":"i2","name":"Guruch","unit":"kg"},
                           {"ingredientId":"i1","name":"Kartoshka","unit":"kg"}]}""",
        )
        assertEquals(1, d.rows.size)
        // ⚠️ The whole catalogue, not only what is short: with only the shortage
        // on offer the writer types a name, and that creates a second ingredient
        // no tech card points at.
        assertEquals(2, d.catalog.size)
    }

    /** `handlers/staffbuy.go` → what a recorded run answers with. */
    @Test
    fun `a run that was already recorded says so rather than failing`() {
        val res = json.decodeFromString<BuyResult>(
            """{"purchase":{"id":"p1","total":250000},"repriced":3,
                "created":["Rayhon"],"already":true}""",
        )
        assertEquals(250000.0, res.purchase.total, 0.0)
        assertTrue(res.already)
        // What had to be invented is named, because an ingredient created at a
        // market has no unit, no minimum and no card.
        assertEquals(listOf("Rayhon"), res.created)
    }

    /** ⚠️ **Which screens exist is asked in permissions, never in a job title.**
     *  The spelling of a title grants nothing (`models/staffrole.go`), and a
     *  cashier holding `buyorder` is not somebody whose phone plans a
     *  restaurant's buying. */
    @Test
    fun `writing a shopping list needs the permission and the standing`() {
        assertTrue(canWriteHere(Staff(perms = listOf("buyorder", "void"))))
        assertTrue(canWriteHere(Staff(perms = listOf("buyorder", "stock"))))
        // A cashier: holds the permission because the till is where they hear a
        // thing has run out, but the section would be one they never use.
        assertFalse(canWriteHere(Staff(perms = listOf("buyorder"))))
        assertFalse(canWriteHere(Staff(perms = listOf("void", "stock"))))
        assertFalse(canWriteHere(Staff()))
    }
}
