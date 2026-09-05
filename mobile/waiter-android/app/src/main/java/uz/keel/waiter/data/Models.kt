package uz.keel.waiter.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// The shapes the server sends.
//
// ⚠️ **A hand-written second copy of `frontend/src/lib/types.ts`, and that is
// the largest single cost of leaving TypeScript.** Nothing checks that these
// agree: a field renamed on the server compiles here and arrives null at
// runtime, on a phone, during service. Two rules keep it survivable:
//
//  1. `ignoreUnknownKeys = true` on the decoder — the server adds fields
//     constantly (the panel's, the till's), and a strict decoder would make
//     every unrelated backend release crash this app.
//  2. **Only the fields this app reads are declared.** A model that mirrored
//     all 4756 lines of types.ts would be four thousand lines nobody maintains,
//     and the ones that went stale would be invisible.
//
// ⚠️ **Every list defaults to empty, never null.** Go's nil slices marshal as
// `null` — the tuple this codebase has been bitten by twice — and a `List<T>`
// that arrives null crashes the screen drawing it. Declared nullable-with-default
// so the decoder tolerates both spellings.

@Serializable
data class Staff(
    val id: String = "",
    val branchId: String? = null,
    val name: String = "",
    val phone: String = "",
    val username: String = "",
    val position: String = "",
    val canWaiter: Boolean = false,
    val canCashier: Boolean = false,
    val canKitchen: Boolean = false,
    val roleName: String? = null,
    val isActive: Boolean = true,
)

@Serializable
data class StaffMe(val staff: Staff, val openShift: Shift? = null)

@Serializable
data class LoginResponse(val token: String, val staff: Staff)

@Serializable
data class Shift(
    val id: String = "",
    val staffId: String = "",
    val date: String = "",
    @SerialName("in") val clockIn: String = "",
    val out: String? = null,
    val minutes: Int = 0,
)

@Serializable
data class Category(
    val id: String = "",
    val name: String = "",
    val nameRu: String? = null,
    val nameEn: String? = null,
    val sortOrder: Int = 0,
)

@Serializable
data class MenuItem(
    val id: String = "",
    val categoryId: String = "",
    val name: String = "",
    val nameRu: String? = null,
    val nameEn: String? = null,
    val price: Double = 0.0,
    val imageUrl: String = "",
    val isAvailable: Boolean = true,
    val soldOut: Boolean = false,
    /** Parts of one portion this dish may be sold in, as percents (25, 50, 75).
     *  ⚠️ Empty means whole portions only — the safe reading. A missing value
     *  taken as "divisible" would put a half-portion button on two hundred
     *  dishes nobody meant to divide. */
    val portions: List<Int>? = null,
)

@Serializable
data class MenuGroup(val category: Category, val items: List<MenuItem> = emptyList())

@Serializable
data class TableZone(
    val id: String = "",
    val name: String = "",
    val bookable: Boolean = true,
    val sort: Int = 0,
)

@Serializable
data class FloorTable(
    val id: String = "",
    val number: String = "",
    val seats: Int = 0,
    val isActive: Boolean = true,
    /** ⚠️ Empty means the default zone — every table drawn before zones existed
     *  has no id here, and a screen that filtered them out would show an empty
     *  room to exactly those restaurants. */
    val zoneId: String? = null,
)

@Serializable
data class BookingSettings(
    val enabled: Boolean = false,
    val zones: List<TableZone> = emptyList(),
    val tables: List<FloorTable> = emptyList(),
)

@Serializable
data class BranchInfo(
    val id: String = "",
    val name: String = "",
    val currency: String = "",
    val booking: BookingSettings = BookingSettings(),
    val servicePercent: Double = 0.0,
)

@Serializable
data class CheckLineVoid(val at: String = "", val by: String = "", val reason: String = "")

@Serializable
data class CheckLine(
    val lineId: String = "",
    val menuItemId: String? = null,
    val name: String = "",
    val price: Double = 0.0,
    val qty: Int = 0,
    val sum: Double = 0.0,
    val comment: String? = null,
    /** Whether the kitchen has this line. The one distinction the check screen
     *  is built around: what is cooking versus what is still a draft. */
    val fired: Boolean = false,
    @SerialName("void") val voided: CheckLineVoid? = null,
    val readyAt: String? = null,
    val servedAt: String? = null,
    /** Part of one portion as a percent (50 = half). Absent is a whole one. */
    val portion: Int? = null,
)

@Serializable
data class Check(
    val id: String = "",
    val number: String = "",
    val status: String = "",
    val tableId: String? = null,
    val tableNumber: String? = null,
    val guests: Int = 0,
    val openedAt: String = "",
    val openMin: Int = 0,
    val lines: List<CheckLine> = emptyList(),
    val subtotal: Double = 0.0,
    /** Lines typed but not sent to the kitchen. The single number the floor
     *  screen is read for. */
    val unfired: Int = 0,
    val readyWaiting: Int = 0,
    val served: Int = 0,
    val service: Double = 0.0,
    val servicePercent: Double = 0.0,
    val total: Double = 0.0,
    val precheckAt: String? = null,
    /** Who has this check open on another screen right now, if anybody. Empty
     *  once the hold goes stale — a name here means somebody is on it *now*. */
    val heldBy: String? = null,
)

@Serializable
data class ChecksResponse(val checks: List<Check> = emptyList(), val soldOut: List<String> = emptyList())

@Serializable
data class SplitResponse(val check: Check, val split: Check)

@Serializable
data class PrintResponse(
    val lines: List<String> = emptyList(),
    val widthMM: Int = 0,
    /** How many of the branch's own printers took it. ⚠️ Zero is not an error
     *  and is said plainly — the honest next step is the till, not a retry. */
    val queued: Int = 0,
    /** ⚠️ Which silence a `queued: 0` is. "No printer configured" and "the till
     *  app is switched off" look identical from here and send a waiter to two
     *  completely different places. */
    val tillOff: Boolean = false,
    val check: Check,
)

@Serializable
data class AddLineRequest(val menuItemId: String, val qty: Int, val portion: Int? = null)

@Serializable
data class StaffDay(
    val date: String = "",
    val weekday: Int = 0,
    val expected: Int = 0,
    val worked: Int = 0,
    val diff: Int = 0,
    val first: String = "",
    val last: String = "",
    val open: Boolean = false,
    val status: String = "off",
    val pay: Double = 0.0,
)

@Serializable
data class StaffTotals(val worked: Int = 0, val expected: Int = 0, val diff: Int = 0)

@Serializable
data class StaffTrend(val current: Int = 0, val previous: Int = 0)

@Serializable
data class StaffReport(
    val days: List<StaffDay> = emptyList(),
    val totals: StaffTotals = StaffTotals(),
    val today: StaffTrend = StaffTrend(),
    val week: StaffTrend = StaffTrend(),
    val month: StaffTrend = StaffTrend(),
    val periodPay: Double = 0.0,
    val periodDue: Double = 0.0,
)

/** Picks the right language for *database content* — dish and category names.
 *
 *  ⚠️ **The site's own rule, ported unchanged** (`lib/i18n/content.ts`): Uzbek is
 *  the base, RU/EN are optional, and an empty translation always falls back to
 *  the base text. That is what stops a half-translated menu from showing empty
 *  rows to the waiter who most needs to read it. */
fun contentName(base: String, ru: String?, en: String?, lang: String): String = when (lang) {
    "ru" -> ru?.trim().takeUnless { it.isNullOrEmpty() } ?: base
    "en" -> en?.trim().takeUnless { it.isNullOrEmpty() } ?: base
    else -> base
}

fun MenuItem.displayName(lang: String) = contentName(name, nameRu, nameEn, lang)
fun Category.displayName(lang: String) = contentName(name, nameRu, nameEn, lang)
