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
    val id: String = "",
    val name: String = "",
    val qty: Int = 0,
    val revenue: Double = 0.0,
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
)

@Serializable
data class LossAlerts(val alerts: List<LossAlert> = emptyList())

// ---- Orders ----

@Serializable
data class OrderItemRow(
    val name: String = "",
    val qty: Int = 0,
    val price: Double = 0.0,
)

@Serializable
data class Order(
    val id: String = "",
    val number: String = "",
    val status: String = "",
    val type: String = "",
    val customerName: String = "",
    val phone: String = "",
    val address: String = "",
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
    val need: Double = 0.0,
    val unit: String = "",
)

@Serializable
data class ShoppingGroup(
    val supplier: String = "",
    val lines: List<ShoppingLine> = emptyList(),
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
