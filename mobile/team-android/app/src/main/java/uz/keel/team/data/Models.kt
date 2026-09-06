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
)

@Serializable
data class ShoppingOrder(
    val id: String = "",
    /** ⚠️ A string, not a date: the driver hands every date back in UTC, and
     *  "which day" is exactly the question that would then be off by one. */
    val forDate: String = "",
    val status: String = "sent",
    val lines: List<ShoppingLine> = emptyList(),
    val createdBy: String = "",
)

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

@Serializable
data class FinishResult(
    val order: ShoppingOrder = ShoppingOrder(),
    val purchase: Purchase? = null,
    val already: Boolean = false,
)

/** Which install this is, for the header the server binds an account to. */
data class DeviceInfo(val id: String, val app: String, val name: String)

/** The error body every Keel handler writes. */
@Serializable
data class WireError(val error: String? = null, val message: String? = null)
