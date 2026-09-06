package uz.keel.courier.data

import kotlinx.serialization.Serializable

// What the server sends this phone.
//
// ⚠️ **A hand-written second copy of the panel's types** (`lib/types.ts`), and
// the copies are kept honest by a test fed the JSON the Go handlers actually
// write — never JSON invented on this side, which would agree with this side by
// construction. kotlinx fills a default for any key it does not find, so a
// renamed field is not an error here: it is a price of zero on a card a courier
// is about to collect cash against.

/** `off` · `free` · `busy`.
 *
 *  ⚠️ Kept as text rather than an enum: the status also arrives from the panel,
 *  and a value this build has not heard of must not throw while parsing the
 *  answer that would have explained it. */
@Serializable
data class Courier(
    val id: String = "",
    val name: String = "",
    val phone: String = "",
    val username: String = "",
    val status: String = "off",
    val vehicle: String = "",
    val isActive: Boolean = true,
)

@Serializable
data class LoginResult(val token: String = "", val courier: Courier = Courier())

@Serializable
data class OrderCustomer(val name: String = "", val phone: String = "")

@Serializable
data class OrderAddress(
    val text: String = "",
    val comment: String = "",
    /** ⚠️ Nullable, and the difference matters: an order typed over the phone
     *  has no map point, and the arrival check is switched off for it rather
     *  than measured from zero — which is a spot in the Gulf of Guinea. */
    val lat: Double? = null,
    val lng: Double? = null,
)

@Serializable
data class OrderItem(
    val name: String = "",
    val qty: Int = 1,
    val comment: String = "",
)

@Serializable
data class Order(
    val id: String = "",
    val number: String = "",
    val status: String = "",
    /** `delivery` · `pickup` · `dinein`. */
    val type: String = "delivery",
    val customer: OrderCustomer = OrderCustomer(),
    val address: OrderAddress = OrderAddress(),
    val items: List<OrderItem> = emptyList(),
    val total: Double = 0.0,
    /** `cash` · `card` · a provider's name. ⚠️ Only "cash" changes what the
     *  courier does, and it changes it at the door. */
    val paymentMethod: String = "cash",
)

/** One delivered order, as the history lists it. */
@Serializable
data class CourierOrderRow(
    val id: String = "",
    val number: String = "",
    val total: Double = 0.0,
    val paymentMethod: String = "cash",
    val earned: Double = 0.0,
    val deliveredAt: String = "",
    val address: OrderAddress = OrderAddress(),
)

@Serializable
data class CourierPeriod(
    val orders: Int = 0,
    val earnings: Double = 0.0,
    /** Cash physically collected in this window. */
    val cash: Double = 0.0,
    val total: Double = 0.0,
)

@Serializable
data class CourierStats(
    val today: CourierPeriod = CourierPeriod(),
    val week: CourierPeriod = CourierPeriod(),
    val month: CourierPeriod = CourierPeriod(),
    val all: CourierPeriod = CourierPeriod(),
    /** ⚠️ **The number this screen exists for**: everything collected less
     *  everything handed back. `all.cash` alone only ever grows, so a courier
     *  who settled up this morning would still be shown a debt. */
    val cashInHand: Double = 0.0,
    val cashSettled: Double = 0.0,
)

/** One position, on its way to the restaurant. */
@Serializable
data class Point(
    val lat: Double,
    val lng: Double,
    val accuracy: Double,
    /** Unix milliseconds, from the fix itself rather than from when it was
     *  sent: the server judges an arrival by how old the fix is. */
    val at: Long,
)

/** As much of the restaurant profile as this app reads.
 *
 *  ⚠️ **Only the arrival radius.** The profile carries addresses, keys and
 *  delivery settings; parsing the parts this app has no use for would be a
 *  second copy of the panel's largest type, kept up to date for nothing. */
@Serializable
data class DeliverySettings(val arrivalRadiusM: Int = 0)

@Serializable
data class RestaurantProfile(val delivery: DeliverySettings = DeliverySettings())

@Serializable
data class RestaurantEnvelope(val restaurant: RestaurantProfile = RestaurantProfile())

/** Which install this is, for the header the server binds an account to. */
data class DeviceInfo(val id: String, val app: String, val name: String)

/** The error body every Keel handler writes. */
@Serializable
data class WireError(val error: String? = null, val message: String? = null)
