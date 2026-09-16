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
import uz.keel.team.data.ShoppingOrder
import uz.keel.team.data.SavedCount
import uz.keel.team.data.StocktakeSheet
import uz.keel.team.data.Warehouses
import uz.keel.team.data.LabelCandidates
import uz.keel.team.data.MarkItems
import uz.keel.team.data.MarkResult
import uz.keel.team.ui.screens.canCountHere
import uz.keel.team.ui.screens.canLabelHere
import uz.keel.team.ui.screens.canScanHere
import uz.keel.team.ui.screens.canIssueHere
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
    fun `writing a shopping list is exactly the permission, and nothing else`() {
        assertTrue(canWriteHere(Staff(perms = listOf("buyorder", "void"))))
        assertTrue(canWriteHere(Staff(perms = listOf("buyorder", "stock"))))
        // ⚠️ **The permission on its own is enough, and it did not used to be.**
        // This screen also demanded management or the store — "a cashier's phone
        // is not where buying gets planned" — and that was wrong about who
        // notices things running out: the barman with an empty fridge and the
        // cook who used the last of the flour, neither of whom voids checks or
        // counts shelves. An owner now ticks a box on the person, and a second
        // condition here would silently overrule the tick they just made.
        assertTrue(canWriteHere(Staff(perms = listOf("buyorder"))))
        assertFalse(canWriteHere(Staff(perms = listOf("void", "stock"))))
        assertFalse(canWriteHere(Staff()))
    }

    // ⚠️ **Issuing from the store is its own permission, not a corner of
    // `stock`.** Counting a shelf and emptying it are different acts; folded
    // together, every person given a phone to count the fridge would also hold
    // the button that sends a case of vodka across town.
    @Test
    fun `handing goods out is its own permission`() {
        assertTrue(canIssueHere(Staff(perms = listOf("stockissue"))))
        assertFalse(canIssueHere(Staff(perms = listOf("stock", "buyorder"))))
        assertFalse(canIssueHere(Staff()))
    }

    // ⚠️ **A request splits by where it is answered from, and the phone reads
    // that back.** The writer chose none of it, so a screen that could not tell
    // the two apart would leave them no way to notice that the lemons they meant
    // for the market went to a storekeeper.
    @Test
    fun `a list says which half of the morning it is`() {
        val payload = """
            {"orders":[
              {"id":"a","forDate":"2026-09-08","source":"store","status":"sent",
               "groupId":"g1","lines":[{"id":"l1","name":"Kola","qty":5}]},
              {"id":"b","forDate":"2026-09-08","source":"market","status":"shipped",
               "groupId":"g1","shippedBy":"Aziz",
               "lines":[{"id":"l2","name":"Limon","qty":5,"gotQty":4,
                         "gotAt":"2026-09-08T09:00:00Z","tookQty":3.5}]}
            ]}
        """.trimIndent()
        val orders = json.decodeFromString<ShoppingOrders>(payload).orders
        assertTrue(orders[0].fromStore)
        assertTrue(orders[0].open)
        assertFalse(orders[1].fromStore)
        assertTrue(orders[1].waiting)
        assertEquals("Aziz", orders[1].shippedBy)
        // ⚠️ **Counted is its own field and stays nullable.** A line nobody
        // checked and one that arrived empty are different facts, and a screen
        // that cannot tell them apart accuses somebody.
        assertEquals(null, orders[0].lines[0].tookQty)
        assertEquals(3.5, orders[1].lines[0].tookQty)
    }

    // ---- Marking codes and the shop's own labels ----

    /** `handlers/staffmarking.go` → `StaffMarkItems`. */
    @Test
    fun `the marked products carry the shop's own code beside the name`() {
        val items = json.decodeFromString<MarkItems>(
            """{"items":[{"id":"m1","name":"Coca-Cola 0.5","barcode":"2100000000010"},
                         {"id":"m2","name":"Pepsi 1.0","barcode":""}]}""",
        )
        assertEquals(2, items.items.size)
        assertEquals("Coca-Cola 0.5", items.items[0].name)
        // ⚠️ The shop's own EAN, never the state's DataMatrix. The two live in
        // different fields the whole length of this system, because one of them
        // may reach a fiscal receipt and the other never may.
        assertEquals("2100000000010", items.items[0].barcode)
        assertEquals("", items.items[1].barcode)
    }

    /** `handlers/staffmarking.go` → `StaffReceiveMarks`.
     *
     *  ⚠️ **Three answers, never one number.** A box of forty with one
     *  unreadable sticker is thirty-nine bottles filed and one still in
     *  somebody's hand — and duplicates come back by code, because the person
     *  is holding that bottle. */
    @Test
    fun `a scanned box answers with what was taken and what was refused`() {
        val res = json.decodeFromString<MarkResult>(
            """{"added":38,
                "duplicates":["0104780123456789215Ab7"],
                "bad":["4780123456789"]}""",
        )
        assertEquals(38, res.added)
        assertEquals(1, res.duplicates.size)
        // An EAN-13 read instead of a DataMatrix: not a marking code at all.
        assertEquals(listOf("4780123456789"), res.bad)
    }

    /** ⚠️ **A clean box answers with empty lists, not with absent fields.** Read
     *  as null the screen would crash on the one delivery where nothing went
     *  wrong — the JSON trap this codebase has been bitten by twice. */
    @Test
    fun `a clean box still carries both lists`() {
        val res = json.decodeFromString<MarkResult>("""{"added":40}""")
        assertEquals(40, res.added)
        assertTrue(res.duplicates.isEmpty())
        assertTrue(res.bad.isEmpty())
    }

    /** `handlers/staffmarking.go` → `StaffLabelCandidates`. */
    @Test
    fun `a label candidate says why it is on the list`() {
        val rows = json.decodeFromString<LabelCandidates>(
            """{"items":[{"id":"p1","name":"Guruch 1kg","price":18000,"barcode":"",
                          "reason":"noBarcode","wasPrice":0},
                         {"id":"p2","name":"Shakar 1kg","price":12000,
                          "barcode":"2100000000027","reason":"price","wasPrice":11000}]}""",
        )
        // ⚠️ The order matters: no code stops a sale outright, a changed price
        // only misdescribes one.
        assertEquals("noBarcode", rows.items[0].reason)
        assertEquals("price", rows.items[1].reason)
        assertEquals(11000.0, rows.items[1].wasPrice, 0.0)
    }

    /** ⚠️ **Two permissions, not one.** One scans what the state issued, the
     *  other prints what the shop invented — and a shop may trust the same
     *  person with neither, either or both. */
    @Test
    fun `scanning and printing are asked separately`() {
        assertTrue(canScanHere(Staff(perms = listOf("marking"))))
        assertFalse(canScanHere(Staff(perms = listOf("label", "stock"))))
        assertTrue(canLabelHere(Staff(perms = listOf("label"))))
        assertFalse(canLabelHere(Staff(perms = listOf("marking"))))
        // ⚠️ The title grants nothing, however it is spelled.
        assertFalse(canScanHere(Staff(position = "Omborchi")))
    }

    // ---- Counting the store ----

    /** `handlers/staffstock.go` → `StaffWarehouses`. */
    @Test
    fun `the stores are the rooms this branch counts one at a time`() {
        val w = json.decodeFromString<Warehouses>(
            """{"warehouses":[
                 {"id":"w1","branchId":"b1","name":"Bar","note":"Dilnoza sanaydi",
                  "sort":0,"isActive":true,"createdAt":"2026-01-01T00:00:00Z",
                  "updatedAt":"2026-01-01T00:00:00Z"},
                 {"id":"w2","branchId":"b1","name":"Oshxona","kind":"production",
                  "sort":1,"isActive":true,"createdAt":"2026-01-01T00:00:00Z",
                  "updatedAt":"2026-01-01T00:00:00Z"}]}""",
        )
        assertEquals(2, w.warehouses.size)
        assertEquals("Bar", w.warehouses[0].name)
        assertEquals("production", w.warehouses[1].kind)
    }

    /** `handlers/stocktake.go` → `stocktakeSheet`.
     *
     *  ⚠️ **The sheet carries no expected figure, and this test is what keeps it
     *  that way.** A field added here would read as zero on today's server and
     *  be drawn beside an empty box — which is the sheet that gets the expected
     *  figure written into it. */
    @Test
    fun `the sheet says what to count and never what should be there`() {
        val s = json.decodeFromString<StocktakeSheet>(
            """{"rows":[{"ingredientId":"i1","name":"Kartoshka","unit":"kg"},
                        {"ingredientId":"i2","name":"Kola 0.5","unit":"dona"}],
                "since":"2026-09-13T15:00:00Z"}""",
        )
        assertEquals(2, s.rows.size)
        assertEquals("Kartoshka", s.rows[0].name)
        assertEquals("dona", s.rows[1].unit)
        assertEquals("2026-09-13T15:00:00Z", s.since)
    }

    /** ⚠️ **A store nobody has ever counted sends no date, and the two answers
     *  are different sentences.** "Measured since the 13th" and "measured from
     *  everything that ever arrived" are different claims about one number, and
     *  read as an empty string the screen would print the first over the
     *  second. */
    @Test
    fun `a store that was never counted has no since`() {
        val s = json.decodeFromString<StocktakeSheet>("""{"rows":[],"since":null}""")
        assertEquals(null, s.since)
        assertTrue(s.rows.isEmpty())
    }

    /** `handlers/stocktake.go` → `saveStocktake`, the 201 it answers with.
     *
     *  ⚠️ **The variance comes back with the save and only then.** The count is
     *  insert-only by the time these numbers arrive, so they are a finding
     *  rather than a target — and read as zero they would report a clean count
     *  to somebody standing in front of a shelf that is short. */
    @Test
    fun `the saved count carries the variance it was taken to find`() {
        val res = json.decodeFromString<SavedCount>(
            """{"id":"st1","branchId":"b1","warehouseId":"w1",
                "at":"2026-09-16T17:40:00Z",
                "lines":[{"ingredientId":"i1","counted":7.5,"expected":9.4,
                          "diff":-1.9,"value":-11400},
                         {"ingredientId":"i2","counted":24,"expected":24,
                          "diff":0,"value":0}],
                "note":"ikki quti sinib ketdi","value":-11400,"by":"Dilnoza",
                "createdAt":"2026-09-16T17:40:00Z"}""",
        )
        assertEquals(-11400, res.value)
        assertEquals(-1.9, res.lines[0].diff, 0.001)
        assertEquals(9.4, res.lines[0].expected, 0.001)
        // A line that matched is still a line: it is proof somebody walked up to
        // that shelf, which is exactly what a count is for.
        assertEquals(0.0, res.lines[1].diff, 0.0)
    }

    /** ⚠️ **Counting is its own permission, and it is the technologist's.** Read
     *  off `position` instead, the baseline every later shortfall is measured
     *  from would belong to whoever spells a job title the same way
     *  (models/staffrole.go). */
    @Test
    fun `counting the store is exactly the stock permission`() {
        assertTrue(canCountHere(Staff(perms = listOf("stock"))))
        assertTrue(canCountHere(Staff(perms = listOf("stock", "buyorder", "stockissue"))))
        // ⚠️ The title grants nothing, however it is spelled.
        assertFalse(canCountHere(Staff(position = "Texnolog")))
        assertFalse(canCountHere(Staff(perms = listOf("stockissue", "buy"))))
        assertFalse(canCountHere(Staff()))
    }

    // ⚠️ **A list written before the split existed is a whole request, and it is
    // a market one.** Reading the blank field the other way would drop a
    // fortnight of finished market runs into a storekeeper's queue.
    @Test
    fun `a list from before the split is a market list`() {
        val o = json.decodeFromString<ShoppingOrder>(
            """{"id":"a","forDate":"2026-09-01","status":"done","lines":[]}""",
        )
        assertFalse(o.fromStore)
        assertFalse(o.waiting)
    }
}
