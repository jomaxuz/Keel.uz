package uz.keel.guest.data

import kotlinx.serialization.Serializable

// What the public API sends this phone.
//
// ⚠️ **A hand-written second copy of the site's types** (`frontend/src/lib/types.ts`),
// kept honest by a test fed the JSON the Go handlers actually write. kotlinx
// fills a default for any key it does not find, so a renamed field is not an
// error: it is a menu of dishes that all cost nothing.
//
// ⚠️ **Only what a guest screen draws is here.** The public endpoints answer with
// far more — costs, recipes, fiscal codes — and mirroring all of it would be a
// second catalogue to keep in step for no screen's benefit.

/** A name in the three languages a restaurant here is read in.
 *
 *  ⚠️ **Uzbek is the base and the other two are optional**, exactly as the
 *  panel stores them: a category the owner never translated must fall back to
 *  the word they typed, not to an empty chip. Every screen asks through
 *  `pick()`, never by reading `name` directly. */
interface Named {
    val name: String
    val nameRu: String
    val nameEn: String
}

/** The name in this language, falling back to the one that always exists. */
fun Named.pick(lang: String): String = when (lang) {
    "ru" -> nameRu.ifBlank { name }
    "en" -> nameEn.ifBlank { name }
    else -> name
}

@Serializable
data class Category(
    val id: String = "",
    override val name: String = "",
    override val nameRu: String = "",
    override val nameEn: String = "",
    val slug: String = "",
    val sortOrder: Int = 0,
    val isActive: Boolean = true,
    val imageUrl: String = "",
) : Named

@Serializable
data class MenuItem(
    val id: String = "",
    val categoryId: String = "",
    override val name: String = "",
    override val nameRu: String = "",
    override val nameEn: String = "",
    val description: String = "",
    val descriptionRu: String = "",
    val descriptionEn: String = "",
    /** Whole so'm. ⚠️ Never tiyin — the currency has none in practice and a
     *  screen that divided by a hundred would price a plov at 450. */
    val price: Double = 0.0,
    /** What it used to cost, when the restaurant is showing a reduction.
     *
     *  ⚠️ **Nullable, and null is not zero.** Zero would draw a struck-through
     *  "0 so'm" beside every ordinary dish on the menu. */
    val oldPrice: Double? = null,
    val imageUrl: String = "",
    /** ⚠️ **The one field that decides whether a dish can be ordered**, and it
     *  arrives already answering for both reasons: the owner switched it off,
     *  or the branch's stop list took it off today. The phone must not compute
     *  that a second way. */
    val isAvailable: Boolean = true,
    /** ⚠️ **Never re-ordered or defaulted on the phone.** The panel decides
     *  which question comes first and whether it is compulsory, and a screen
     *  that picked a default for a required group would send an order the guest
     *  never actually chose. */
    val options: List<MenuOption> = emptyList(),
) : Named {
    /** The description in this language, falling back to the base. */
    fun describe(lang: String): String = when (lang) {
        "ru" -> descriptionRu.ifBlank { description }
        "en" -> descriptionEn.ifBlank { description }
        else -> description
    }

    val discounted: Boolean get() = (oldPrice ?: 0.0) > price
}

/** One category with its dishes, the shape `GET /menu` answers in.
 *
 *  ⚠️ **Grouped by the server, not here.** The site reads the same shape, and a
 *  phone that regrouped a flat list would sooner or later disagree with it about
 *  which category an item is in — on the screen the whole app exists for. */
@Serializable
data class MenuGroup(
    val category: Category = Category(),
    val items: List<MenuItem> = emptyList(),
)

/** How this restaurant looks and what it is called.
 *
 *  ⚠️ **Read at launch even though the build already carries the name and the
 *  colour.** The build is a photograph taken on the day it was made; a
 *  restaurant that changes its logo would otherwise have to wait for a new
 *  release to see it. What the build carries is the *launcher* identity — the
 *  icon and the splash, which genuinely cannot change without one. */
@Serializable
data class Restaurant(
    val name: String = "",
    val description: String = "",
    val logoUrl: String = "",
    val coverUrl: String = "",
    val phones: List<String> = emptyList(),
    val currency: String = "",
    /** Whether the kitchen is taking orders right now. ⚠️ Computed by the
     *  server from the branch's hours, because a phone's clock is the one thing
     *  this product never trusts. */
    val isOpenNow: Boolean = true,
    /** Where the restaurant is, so the address picker opens somewhere useful.
     *
     *  ⚠️ **The map opens on the restaurant, not on the guest.** A location
     *  permission asked before anybody has said they want delivery is a
     *  permission most people refuse, and a map that opens on the null island
     *  is a map somebody closes. */
    val address: GeoPoint = GeoPoint(),
)

/** A point on the map, in the order the server writes it. ⚠️ `lat` then `lng`,
 *  and every provider disagrees about that — the coordinate order lives in one
 *  place on the web side for exactly this reason (`lib/map/`). Here there is one
 *  engine and one order, and it is this one. */
@Serializable
data class GeoPoint(
    val lat: Double = 0.0,
    val lng: Double = 0.0,
    val text: String = "",
)

/** One selectable value inside an option group. */
@Serializable
data class OptionChoice(
    override val name: String = "",
    override val nameRu: String = "",
    override val nameEn: String = "",
    /** Added to the dish price, and it may be negative. */
    val priceDelta: Double = 0.0,
) : Named

/** A question the kitchen needs answered before it can cook the dish.
 *
 *  ⚠️ **`required` is enforced on the phone and again on the server.** The
 *  screen's job is to stop somebody ordering a pizza with no size; the server's
 *  is to stop a stale app doing it. Neither alone is enough — the first is a
 *  courtesy, the second is the record. */
@Serializable
data class MenuOption(
    override val name: String = "",
    override val nameRu: String = "",
    override val nameEn: String = "",
    val required: Boolean = false,
    val multiple: Boolean = false,
    val choices: List<OptionChoice> = emptyList(),
) : Named

/** What a guest asked for and what the server worked it out to.
 *
 *  ⚠️ **Every figure on the checkout comes from here, and none is added up on
 *  the phone.** A discount, a delivery fee and a points balance each have rules
 *  the app does not know — and a total the guest computed that disagrees with
 *  the one they are charged is the single worst thing this screen can do. */
@Serializable
data class OrderQuote(
    val subtotal: Double = 0.0,
    val discountTotal: Double = 0.0,
    val deliveryFee: Double = 0.0,
    val total: Double = 0.0,
    /** Why a typed code was refused. ⚠️ Never fatal: the order still goes. */
    val codeError: String = "",
    val codeApplied: Boolean = false,
    /** Whether this can be ordered at all — closed, out of range, below the
     *  minimum. */
    val available: Boolean = true,
    val minOrder: Double = 0.0,
    val belowMinimum: Boolean = false,
    val branchId: String = "",
    val branchName: String = "",
    val prepMinutes: Int = 0,
    /** ⚠️ **Known only once the branch is.** The guest browsed one branch's menu
     *  and a delivery may be taken by another entirely; the order would be
     *  refused for these anyway, and saying so turns a wasted checkout into
     *  something the guest can act on. */
    val soldOut: List<String> = emptyList(),
    val pointsSpent: Double = 0.0,
    val pointsBalance: Double = 0.0,
    val pointsMax: Double = 0.0,
    val pointsEarn: Double = 0.0,
)

/** Whether this address can be delivered to, by whom, and for how much. */
@Serializable
data class DeliveryQuote(
    val available: Boolean = false,
    val deliveryFee: Double = 0.0,
    val zone: String = "",
    val distanceKm: Double = 0.0,
    val minOrder: Double = 0.0,
    val branchId: String = "",
    val branchName: String = "",
    val prepMinutes: Int = 0,
)

/** How this restaurant takes money.
 *
 *  ⚠️ **Asked, never assumed.** Cash is always in the list; a provider appears
 *  only once it is switched on and fully credentialed, and an app that drew a
 *  Payme button from a hard-coded list would offer a payment that cannot
 *  complete. */
@Serializable
data class PaymentMethods(
    val methods: List<String> = emptyList(),
)

@Serializable
data class OrderItemOption(
    val name: String = "",
    val choice: String = "",
    val priceDelta: Double = 0.0,
)

@Serializable
data class OrderItem(
    val menuItemId: String = "",
    val name: String = "",
    val price: Double = 0.0,
    val qty: Int = 0,
    val comment: String = "",
    val options: List<OrderItemOption> = emptyList(),
)

/** One step of the order's life, with the moment it happened. */
@Serializable
data class StatusEvent(
    val status: String = "",
    val at: String = "",
)

/** An order, as the guest tracks it.
 *
 *  ⚠️ **`payUrl` is only ever on the answer to placing one**, never on a later
 *  read: it is a bank link minted for this attempt, and a screen that kept it
 *  would send somebody to a dead page days later. */
@Serializable
data class Order(
    val id: String = "",
    val number: String = "",
    val status: String = "pending",
    val type: String = "delivery",
    val items: List<OrderItem> = emptyList(),
    val subtotal: Double = 0.0,
    val discountTotal: Double = 0.0,
    val deliveryFee: Double = 0.0,
    val total: Double = 0.0,
    val paymentMethod: String = "cash",
    val createdAt: String = "",
    val statusHistory: List<StatusEvent> = emptyList(),
    val cancelReason: String = "",
    val payUrl: String = "",
)
