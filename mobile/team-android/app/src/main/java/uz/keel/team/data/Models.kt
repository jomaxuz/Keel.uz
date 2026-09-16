package uz.keel.team.data

import kotlinx.serialization.Serializable

// What the server sends this phone.
//
// ⚠️ **A hand-written second copy of the panel's types** (`lib/types.ts`), kept
// honest by a test fed the JSON the Go handlers actually write. kotlinx fills a
// default for any key it does not find, so a renamed field is not an error: it
// is a month of somebody's hours reading zero on the screen they check their pay
// against.

/** Who is signed in.
 *
 *  ⚠️ **`perms` decides which screens exist**, and it is a list of permissions
 *  rather than a job title: the spelling of a title grants nothing
 *  (`models/staffrole.go`), so every question here is asked in permissions. */
@Serializable
data class Staff(
    val id: String = "",
    val branchId: String = "",
    val name: String = "",
    val phone: String = "",
    val username: String = "",
    /** Free text: "oshpaz", "ofitsiant", "kassir". */
    val position: String = "",
    val perms: List<String> = emptyList(),
    val isActive: Boolean = true,
)

@Serializable
data class LoginResult(val token: String = "", val staff: Staff = Staff())

/** One shift, as the clock button gets it back. */
@Serializable
data class Shift(
    val id: String = "",
    /** The working day this is filed under, local YYYY-MM-DD. */
    val date: String = "",
    val minutes: Int = 0,
)

@Serializable
data class StaffMe(
    val staff: Staff = Staff(),
    val openShift: Shift? = null,
)

/** One day of the attendance calendar.
 *
 *  ⚠️ **`status` is the server's word and its colour is the panel's.** A shift
 *  that is amber in the office cannot be green in somebody's hand, so the label
 *  is this app's and the colour comes from the shared `StatusColor`. */
@Serializable
data class StaffDay(
    val date: String = "",
    val weekday: Int = 0,
    /** Minutes the roster asks for; 0 on a day off. */
    val expected: Int = 0,
    val worked: Int = 0,
    /** Signed: positive is overtime. */
    val diff: Int = 0,
    val open: Boolean = false,
    /** `off` · `absent` · `under` · `ok` · `over` · `extra` · `open` ·
     *  `upcoming`. ⚠️ Text, not an enum: a status a newer server invents must
     *  not throw while parsing the month that would have explained it. */
    val status: String = "off",
)

@Serializable
data class StaffTotals(
    val days: Int = 0,
    val absent: Int = 0,
    val expected: Int = 0,
    val worked: Int = 0,
    val diff: Int = 0,
)

@Serializable
data class StaffTrend(val current: Int = 0, val previous: Int = 0)

/** ⚠️ **The same endpoint the panel's payroll reads.** This is a second
 *  *drawing* of one answer, never a second calculation: a phone that added up
 *  its own hours would disagree with the payroll screen, and the disagreement
 *  would be found on pay day. */
@Serializable
data class StaffReport(
    val days: List<StaffDay> = emptyList(),
    val totals: StaffTotals = StaffTotals(),
    val today: StaffTrend = StaffTrend(),
    val week: StaffTrend = StaffTrend(),
    val month: StaffTrend = StaffTrend(),
    val periodPay: Double = 0.0,
)

// ---- The market run ----

/** One ingredient as the buyer's phone needs it. */
@Serializable
data class BuyCatalogRow(
    val id: String = "",
    val name: String = "",
    /** The purchase unit — kilos, litres, pieces. Never the recipe's grams. */
    val unit: String = "",
    /** ⚠️ **What it cost last time, and the only guard the price has.** A price
     *  typed on a phone reprices every dish that uses the ingredient, and
     *  `9 000` entered as `90 000` looks like an ordinary number afterwards. */
    val lastPrice: Double = 0.0,
    /** How the market sells it, when somebody wrote it down. ⚠️ Sent so the
     *  phone can offer the choice, never so it can perform the conversion: the
     *  arithmetic lands on a shelf and belongs to the server. */
    val packName: String = "",
    val packQty: Double = 0.0,
)

@Serializable
data class BuyCatalog(val ingredients: List<BuyCatalogRow> = emptyList())

@Serializable
data class ShoppingRow(
    val ingredientId: String = "",
    val name: String = "",
    val unit: String = "",
    val onHand: Double = 0.0,
    val suggested: Double = 0.0,
    val price: Double = 0.0,
)

/** One call to make. ⚠️ Grouped by supplier because that is how ordering is
 *  done — one group per phone number is one call. A buyer already standing at a
 *  market has one list to walk, so this screen flattens them. */
@Serializable
data class ShoppingGroup(
    val supplierId: String = "",
    val name: String = "",
    val rows: List<ShoppingRow> = emptyList(),
)

@Serializable
data class ShoppingList(val groups: List<ShoppingGroup> = emptyList())

/** What this buyer is still holding.
 *
 *  ⚠️ **`balance` can be negative and is shown that way.** A buyer who ran out
 *  and paid for the last crate themselves is owed money, and a purse clamped at
 *  zero would be silent about exactly the debt they are waiting on. */
@Serializable
data class AdvanceBalance(
    val issued: Double = 0.0,
    val spent: Double = 0.0,
    val balance: Double = 0.0,
)

@Serializable
data class ShoppingLine(
    val id: String = "",
    /** Empty when whoever wrote the list typed a name the catalogue does not
     *  have. ⚠️ Allowed on purpose: a list somebody cannot finish writing is a
     *  list they write on paper, where nothing here can see it. */
    val ingredientId: String = "",
    val name: String = "",
    val unit: String = "",
    val qty: Double = 0.0,
    /** What came back, kept beside what was asked rather than replacing it:
     *  "asked for ten, brought six" is the sentence this document exists for. */
    val gotQty: Double = 0.0,
    val price: Double = 0.0,
    val gotAt: String = "",
    /** The market did not have it. ⚠️ Its own answer, not a quantity of zero. */
    val missing: Boolean = false,
    /** What the person who asked for it counted when it turned up.
     *
     *  ⚠️ **Nullable, and absent is not zero.** A line nobody has checked and a
     *  line that arrived empty are different facts, and a screen that cannot
     *  tell them apart accuses somebody. Read it as `tookQty ?: gotQty`. */
    val tookQty: Double? = null,
)

@Serializable
data class ShoppingOrder(
    val id: String = "",
    /** ⚠️ A string, not a date: the driver hands every date back in UTC, and
     *  "which day" is exactly the question that would then be off by one. */
    val forDate: String = "",
    /** Which half of the morning this is — `market` or `store`.
     *
     *  ⚠️ **One request becomes one list per place.** The storekeeper answers in
     *  ten minutes and the buyer at nine, and a shared document would spend the
     *  morning in a state neither of them has a word for.
     *
     *  ⚠️ Empty defaults to the market: every list written before this existed
     *  was one, and reading the blank the other way would put a fortnight of
     *  finished market runs into a storekeeper's queue. */
    val source: String = "market",
    /** The request both halves were written in. */
    val groupId: String = "",
    /** Which branch answers it — its own store room unless a chain points it at
     *  a central one. */
    val supplyBranchId: String = "",
    /** `sent` · `shipped` · `done`. ⚠️ Three, and the third is the point:
     *  "bought" said the money had been spent and nothing about whether the
     *  goods reached the person who asked. */
    val status: String = "sent",
    val lines: List<ShoppingLine> = emptyList(),
    val createdBy: String = "",
    val shippedBy: String = "",
    val acceptedBy: String = "",
) {
    /** Where this one is answered from. */
    val fromStore: Boolean get() = source == "store"

    /** Still waiting for the person it was sent to. */
    val open: Boolean get() = status == "sent"

    /** On its way, and nobody has counted it. ⚠️ The one state worth a colour on
     *  every screen that draws these. */
    val waiting: Boolean get() = status == "shipped"
}

@Serializable
data class ShoppingOrders(val orders: List<ShoppingOrder> = emptyList())

/** One row of the shortage the store worked out, as a list's starting point. */
@Serializable
data class ShoppingDraftRow(
    val ingredientId: String = "",
    val name: String = "",
    val unit: String = "",
    val qty: Double = 0.0,
    val onHand: Double = 0.0,
    val packName: String = "",
    val packQty: Double = 0.0,
    /** Who will answer a line for it. ⚠️ Shown while the list is being written:
     *  the split is the server's, but a writer who cannot see it has no way to
     *  notice the one ingredient filed wrongly — and the wrong filing surfaces
     *  as a request that sat all morning on a phone belonging to somebody who
     *  was never going to answer it. */
    val source: String = "market",
)

/** One ingredient the catalogue already has, for the list writer to pick.
 *
 *  ⚠️ **The whole catalogue, not only what is short.** A list can ask for
 *  something above its minimum, and with only the shortage on offer the writer
 *  had to type a name — which creates a second ingredient no tech card points
 *  at. */
@Serializable
data class ShoppingCatalogRow(
    val ingredientId: String = "",
    val name: String = "",
    val unit: String = "",
    val packName: String = "",
    val packQty: Double = 0.0,
    val source: String = "market",
)

@Serializable
data class ShoppingDraft(
    val rows: List<ShoppingDraftRow> = emptyList(),
    val catalog: List<ShoppingCatalogRow> = emptyList(),
)

@Serializable
data class Purchase(val id: String = "", val total: Double = 0.0)

/** What the server says came of a recorded run. */
@Serializable
data class BuyResult(
    val purchase: Purchase = Purchase(),
    /** What had to be invented, by name. ⚠️ Shown to whoever pressed the button
     *  rather than left for a manager to discover: an ingredient created at a
     *  market has no unit, no minimum and no card. */
    val created: List<String> = emptyList(),
    /** The run was already here — a retry that crossed with its own first
     *  attempt. Not an error, and the screen says so rather than showing one. */
    val already: Boolean = false,
)

/** What comes back from writing a list: one document, or two.
 *
 *  ⚠️ **The split is the server's answer, and the screen reads it back.** The
 *  writer chose none of it, so a phone that simply said "sent" would leave them
 *  with no way to notice that the lemons they meant for the market went to a
 *  storekeeper — the one mistake this routing can make, and one a person fixes
 *  in the catalogue in ten seconds if they are told. */
@Serializable
data class CreatedOrders(val orders: List<ShoppingOrder> = emptyList())

/** What comes back from finishing a half, or from signing for one.
 *
 *  ⚠️ **Shipping is no longer a delivery.** What reaches a shelf is what
 *  somebody at the restaurant counted, so `purchase` is filled when the list is
 *  accepted — not when the buyer leaves the market. */
@Serializable
data class FinishResult(
    val order: ShoppingOrder = ShoppingOrder(),
    val purchase: Purchase? = null,
    val already: Boolean = false,
)

// ---- Counting the store ----

/** One room that is counted on its own.
 *
 *  ⚠️ **A restaurant does not have one store, it has a bar and a kitchen**
 *  (models/warehouse.go). They are counted by different people on different
 *  evenings, and one list covering both lets the bar's shortfall cancel against
 *  the kitchen's surplus — arithmetically fine and impossible to act on. */
@Serializable
data class Warehouse(
    val id: String = "",
    val name: String = "",
    val note: String = "",
    val kind: String = "",
    val sort: Int = 0,
    val isActive: Boolean = true,
)

@Serializable
data class Warehouses(val warehouses: List<Warehouse> = emptyList())

/** One line of the sheet: what to count, and nothing else.
 *
 *  ⚠️ **There is no expected figure here and that is the whole point of the
 *  sheet** (handlers/stocktake.go). A row saying "there should be 9.4 kg" beside
 *  an empty box is a row that gets 9.4 written into it, and the error lands in
 *  the one figure this module exists to produce — the variance — where nothing
 *  downstream can tell it from a theft. The server stopped sending it; a field
 *  added here would read as zero and quietly be believed. */
@Serializable
data class StocktakeSheetRow(
    val ingredientId: String = "",
    val name: String = "",
    /** The purchase unit — the way it is counted on a shelf, never the recipe's
     *  grams. */
    val unit: String = "",
)

@Serializable
data class StocktakeSheet(
    val rows: List<StocktakeSheetRow> = emptyList(),
    /** When this store was last counted, or absent if it never was.
     *
     *  ⚠️ **Nullable, and the two answers read differently.** "Measured since
     *  the 3rd" and "measured from everything that ever arrived" are different
     *  claims about the same number, and a screen that showed only the first
     *  would put a date on a figure that has none. */
    val since: String? = null,
)

/** What a saved count was out by, on one ingredient.
 *
 *  ⚠️ **Comes back with the save and never before it.** The count is
 *  insert-only, so by the time these arrive they are a finding rather than a
 *  target: the numbers can no longer be moved to meet them. */
@Serializable
data class SavedCountLine(
    val ingredientId: String = "",
    val counted: Double = 0.0,
    val expected: Double = 0.0,
    val diff: Double = 0.0,
    /** What the difference was worth at that day's prices. Negative is a
     *  shortfall. */
    val value: Int = 0,
)

/** The count the server recorded. */
@Serializable
data class SavedCount(
    val id: String = "",
    val at: String = "",
    val lines: List<SavedCountLine> = emptyList(),
    val note: String = "",
    /** What the whole count was out by, in money — the line an owner reads.
     *
     *  ⚠️ **Negative is a shortfall and is shown that way.** A figure clamped at
     *  zero would be silent about exactly the thing a count is taken for. */
    val value: Int = 0,
)

/** Which install this is, for the header the server binds an account to. */
data class DeviceInfo(val id: String, val app: String, val name: String)

/** The error body every Keel handler writes. */
@Serializable
data class WireError(val error: String? = null, val message: String? = null)
