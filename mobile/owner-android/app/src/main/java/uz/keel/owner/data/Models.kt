package uz.keel.owner.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// The shapes the panel's API sends.
//
// ⚠️ **A hand-written second copy of `frontend/src/lib/types.ts`, and only the
// fields this application reads.** Nothing checks that they agree: a field
// renamed on the server compiles here and arrives null at runtime, on a phone,
// in an owner's hand. Two rules keep that survivable — `ignoreUnknownKeys`, so
// an unrelated backend release cannot crash the app, and *only what is drawn*,
// so a stale field is one somebody would have noticed on screen.
//
// ⚠️ **Every list defaults to empty, never null.** Go's nil slices marshal as
// `null` — the trap this repository has been bitten by twice — and a `List<T>`
// that arrives null crashes the screen drawing it.

@Serializable
data class AdminUser(
    val id: String = "",
    val username: String = "",
    val name: String = "",
    /** "owner" or "manager". ⚠️ A manager sees one branch; the lens is not a
     *  courtesy, it is what the server already enforces. */
    val role: String = "",
    val branchId: String? = null,
)

@Serializable
data class LoginResponse(val token: String = "", val user: AdminUser = AdminUser())

@Serializable
data class Branch(
    val id: String = "",
    val name: String = "",
    val isActive: Boolean = true,
)

/** Money actually in hand, and the work that has not become money yet. */
@Serializable
data class StatsPeriod(
    val orders: Int = 0,
    /** ⚠️ Collected, not placed: cash handed over on delivery, or a card the
     *  bank confirmed. An owner reading "revenue" means this one. */
    val revenue: Double = 0.0,
    /** Placed, not cancelled, not yet collected — food in the kitchen and on the
     *  road. Real work, and not takings. */
    val pending: Double = 0.0,
    val debt: Double = 0.0,
    val paid: Int = 0,
    val avgOrder: Double = 0.0,
    val cancelled: Int = 0,
    val delivered: Int = 0,
    val cashTotal: Double = 0.0,
)

@Serializable
data class StatsDish(
    val name: String = "",
    val qty: Int = 0,
    /** What that dish took in over the period.
     *
     *  ⚠️ **`total`, which is the server's own name for it** — see
     *  `handlers/adminstats.go`. It was `revenue` here, and every price in the
     *  best-sellers list read 0 so'm: kotlinx cannot know the name is wrong, it
     *  fills the default and the screen draws a confident zero. That is the
     *  failure mode this whole file is a warning about, arriving in the one
     *  place an owner reads for money. */
    val total: Double = 0.0,
)

@Serializable
data class StatsDay(val date: String = "", val orders: Int = 0, val revenue: Double = 0.0)

@Serializable
data class AdminStats(
    val from: String? = null,
    val to: String? = null,
    val period: StatsPeriod = StatsPeriod(),
    val top: List<StatsDish> = emptyList(),
    val series: List<StatsDay> = emptyList(),
)

// ---- What is waiting, and what already happened ----
//
// ⚠️ **Two lists, and they are not the same kind of thing.** A queue is work
// somebody does; a loss is a fact somebody asks about. The screen keeps them
// apart because acting on the second one the way you act on the first is how an
// owner ends up interrogating a waiter over an ordinary void.

@Serializable
data class AlertCount(val pending: Int = 0, val newestAt: String? = null)

@Serializable
data class PreorderAlert(
    val upcoming: Int = 0,
    val newestAt: String? = null,
    val dueAt: String? = null,
    val dueWaiting: Int = 0,
)

@Serializable
data class PosAlert(
    val unaccepted: Int = 0,
    val afterMins: Int = 0,
    val failed: Int = 0,
    val unmapped: Int = 0,
)

@Serializable
data class PrintAlert(val failed: Int = 0)

@Serializable
data class AdminAlerts(
    val orders: AlertCount = AlertCount(),
    val preorders: PreorderAlert = PreorderAlert(),
    val reservations: AlertCount = AlertCount(),
    val pos: PosAlert = PosAlert(),
    val print: PrintAlert = PrintAlert(),
)

/** Something that already happened and has a plain explanation more often than
 *  not. ⚠️ The screen never draws a conclusion from one of these. */
@Serializable
data class LossAlert(
    val id: String = "",
    val kind: String = "",
    val at: String = "",
    val by: String? = null,
    val authBy: String? = null,
    val amount: Double = 0.0,
    val reason: String? = null,
    val subject: String? = null,
    /** The document it came from. ⚠️ **Which document depends on `kind`**: a
     *  sale for a void or a discount, a cash shift for a short drawer, a dish for
     *  a raised recipe. The id alone does not say which, so the screen asks the
     *  kind before it opens anything (`opensCheck`). */
    val refId: String = "",
    val number: String = "",
    val table: String = "",
    val afterPrecheck: Boolean = false,
)

@Serializable
data class LossAlerts(val alerts: List<LossAlert> = emptyList())

/** A real id, not Go's zero ObjectID. ⚠️ `omitempty` does not drop a
 *  `[12]byte`, so an unset reference arrives as twenty-four noughts — non-empty,
 *  and therefore "present" to anything that only checks emptiness. */
fun hasId(id: String?): Boolean = !id.isNullOrEmpty() && id.any { it != '0' }

/** The kinds whose `refId` is a sale — the only ones a tap can open as a check. */
private val CHECK_KINDS = setOf("void_after_precheck", "big_discount", "check_cancelled")

fun LossAlert.opensCheck(): Boolean = kind in CHECK_KINDS && hasId(refId)

// ---- One sale, opened (`handlers/adminchecks.go` → `checkDetail`) ----

@Serializable
data class CheckOption(val name: String = "", val choice: String = "", val priceDelta: Double = 0.0)

@Serializable
data class CheckLine(
    val name: String = "",
    val qty: Int = 0,
    val price: Double = 0.0,
    /** Zero for a voided line: it stays on the bill's face, out of its total. */
    val sum: Double = 0.0,
    val options: List<CheckOption> = emptyList(),
    val comment: String = "",
    val guest: Int = 0,
    val course: Int = 0,
    val firedAt: String? = null,
    val voidedBy: String = "",
    val voidReason: String = "",
    val voidedAt: String? = null,
    val wasted: Boolean = false,
) {
    val voided: Boolean get() = voidedAt != null || voidedBy.isNotEmpty()
}

@Serializable
data class CheckDiscount(
    val name: String = "",
    val amount: Double = 0.0,
    val code: String = "",
    val by: String = "",
    val authBy: String = "",
    val reason: String = "",
)

@Serializable
data class CheckRefund(
    val at: String? = null,
    val by: String = "",
    val reason: String = "",
    val amount: Double = 0.0,
    val method: String = "",
)

@Serializable
data class CheckDetail(
    val id: String = "",
    val number: String = "",
    val table: String = "",
    val guests: Int = 0,
    val server: String = "",
    val closedBy: String = "",
    val openedAt: String? = null,
    val closedAt: String? = null,
    val items: Int = 0,
    val subtotal: Double = 0.0,
    val discount: Double = 0.0,
    val service: Double = 0.0,
    val servicePercent: Int = 0,
    val total: Double = 0.0,
    val paymentMethod: String = "",
    val fiscal: String = "",
    val open: Boolean = false,
    val cancelled: Boolean = false,
    val refunded: Boolean = false,
    val split: Boolean = false,
    val lines: List<CheckLine> = emptyList(),
    val discounts: List<CheckDiscount> = emptyList(),
    val openedBy: String = "",
    val precheckAt: String? = null,
    val refund: CheckRefund? = null,
    val fiscalError: String = "",
    val fiscalSign: String = "",
)

// ---- Where the money is (`models/bank.go` → `MoneyPosition`) ----

@Serializable
data class MoneyPlace(
    /** "safe" | "drawer" | "courier" | "advance" | "bank" | "rail" */
    val kind: String = "",
    /** ⚠️ Uzbek words from the server for the first four kinds; the screen
     *  names those itself and uses this only for a bank account or a rail. */
    val name: String = "",
    val amount: Double = 0.0,
    val note: String = "",
    /** Counted (goes stale) or added up from documents (goes wrong). */
    val counted: Boolean = false,
    val at: String? = null,
)

@Serializable
data class MoneyPosition(
    val cash: List<MoneyPlace> = emptyList(),
    val bank: List<MoneyPlace> = emptyList(),
    val rails: List<MoneyPlace> = emptyList(),
    val cashTotal: Double = 0.0,
    val bankTotal: Double = 0.0,
    val railsTotal: Double = 0.0,
    val cashLimit: Double = 0.0,
    val overLimit: Boolean = false,
    val branchId: String = "",
)

// ---- Orders ----

@Serializable
data class OrderItemRow(
    val name: String = "",
    val qty: Int = 0,
    val price: Double = 0.0,
)

/** Who the order is for. ⚠️ **Nested, because the server nests it.** Flattened
 *  to `customerName` here it was not a wrong value but a decode failure: the
 *  key is present and holds an object, and the list refused to open at all. */
@Serializable
data class OrderCustomer(val name: String = "", val phone: String = "")

/** Where it goes. ⚠️ An object for the same reason, and a delivery order always
 *  carries one — `text` is the line a person reads. */
@Serializable
data class OrderAddress(val text: String = "", val comment: String = "")

@Serializable
data class Order(
    val id: String = "",
    val number: String = "",
    val status: String = "",
    val type: String = "",
    val customer: OrderCustomer = OrderCustomer(),
    val address: OrderAddress = OrderAddress(),
    val total: Double = 0.0,
    val createdAt: String = "",
    val items: List<OrderItemRow> = emptyList(),
    val comment: String = "",
    val tableNumber: String? = null,
)

@Serializable
data class StatusResponse(val status: String = "")

// ---- Guests' words ----

@Serializable
data class Feedback(
    val id: String = "",
    val name: String = "",
    val phone: String = "",
    val rating: Int = 0,
    val text: String = "",
    val createdAt: String = "",
    val handledAt: String? = null,
    val handledNote: String? = null,
    val orderNumber: String? = null,
)

@Serializable
data class FeedbackList(
    val feedback: List<Feedback> = emptyList(),
    val total: Int = 0,
    val unhandled: Int = 0,
)

// ---- The morning briefing ----

@Serializable
data class BriefingCard(
    val key: String = "",
    val title: String = "",
    val body: String = "",
    val area: String = "",
)

@Serializable
data class BriefingResponse(
    val cards: List<BriefingCard> = emptyList(),
    val error: String? = null,
    /** ⚠️ `false` means the tariff does not include it — a state, not a fault.
     *  Absent means the question was never asked. */
    val entitled: Boolean? = null,
)

// ---- Who is at work ----

@Serializable
data class StaffRow(
    val id: String = "",
    val name: String = "",
    val position: String = "",
    /** ⚠️ Read from the server, never worked out here: the roster, a day off and
     *  a shift that ended late are three definitions that would drift the moment
     *  a second one existed. */
    val todayStatus: String = "",
    /** "HH:MM" as the server formatted it — ⚠️ never sliced from a timestamp,
     *  which arrives in UTC and would print an hour nobody worked. */
    val todayIn: String = "",
    val todayOut: String = "",
)

// ---- Payroll and the shopping list ----

@Serializable
data class PayrollRow(
    val staffId: String = "",
    val name: String = "",
    val earned: Double = 0.0,
    val paid: Double = 0.0,
    val due: Double = 0.0,
)

@Serializable
data class PayrollResponse(
    val rows: List<PayrollRow> = emptyList(),
    val earned: Double = 0.0,
    val paid: Double = 0.0,
    val due: Double = 0.0,
)

@Serializable
data class ShoppingLine(
    val name: String = "",
    val unit: String = "",
    /** How much short of its minimum the shelf is.
     *
     *  ⚠️ **`suggested`, the server's name, and a quantity — not money.** It
     *  was read as `need` and drawn through the money formatter, so a shelf
     *  three kilos short would have read "0 so'm kg" twice over: wrong name,
     *  wrong unit. */
    val suggested: Double = 0.0,
)

@Serializable
data class ShoppingGroup(
    /** The supplier's name. ⚠️ `name` on the wire — grouping is by who you
     *  ring, and the group carries that person's name, not a field called
     *  "supplier". */
    val name: String = "",
    val rows: List<ShoppingLine> = emptyList(),
)

@Serializable
data class ShoppingList(
    val groups: List<ShoppingGroup> = emptyList(),
    val cost: Double = 0.0,
)

// ---- The subscription ----

@Serializable
data class SubscriptionNotice(
    val days: Int = 0,
    /** "warn" | "urgent" | "expired". */
    val level: String = "",
    val until: String = "",
)

@Serializable
data class Subscription(
    val enabled: Boolean = false,
    val plan: String = "",
    /** ⚠️ Zero means "agreed separately", not "free" — the screen says so rather
     *  than printing a nought an owner would read as a promise. */
    val monthly: Double = 0.0,
    val paidUntil: String = "",
    /** ⚠️ **The date and a count, never a boolean.** An "active" flag goes stale
     *  at midnight while nobody is looking; a date does not.
     *
     *  ⚠️ And this screen does not warn. The till says so in the last week and
     *  the panel has the whole card — a phone that nags about money is a phone
     *  an owner stops opening. */
    val notice: SubscriptionNotice? = null,
)
