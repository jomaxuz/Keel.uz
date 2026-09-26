package uz.keel.guest.data

import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.request.HttpRequestBuilder
import io.ktor.client.request.delete
import io.ktor.client.request.get
import io.ktor.client.request.header
import io.ktor.client.request.post
import io.ktor.client.request.put
import io.ktor.client.request.setBody
import io.ktor.client.statement.HttpResponse
import io.ktor.client.statement.bodyAsText
import io.ktor.http.ContentType
import io.ktor.http.HttpMethod
import io.ktor.http.content.TextContent
import io.ktor.http.isSuccess
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import uz.keel.design.TokenStore
import uz.keel.guest.Brand
import uz.keel.guest.CartLine

/** The server having spoken, whatever it said.
 *
 *  ⚠️ **"The server says no" and "the request never arrived" are opposite facts
 *  that look identical from a `catch`.** One is a closed kitchen, the other is a
 *  lift — and confusing them is how a guest concludes the restaurant is shut. */
class ApiError(val status: Int, override val message: String) : Exception(message)

/** The error body every Keel handler writes. */
@Serializable
data class WireError(val error: String? = null, val message: String? = null)

/**
 * Every call this application makes.
 *
 * ⚠️ **No server screen.** The address is a constant of the build
 * (`Brand.serverUrl`) — see Brand.kt: a guest cannot be asked anything before
 * the menu.
 *
 * ⚠️ **The token is optional on almost everything.** The menu, the categories
 * and the restaurant are public, and they have to stay reachable by somebody who
 * has never signed in: an app that asked for a phone number before showing a
 * price is an app that gets one launch.
 */
class KeelApi(private val tokens: TokenStore) {

    private val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
        coerceInputValues = true
    }

    private val client = HttpClient(OkHttp) {
        expectSuccess = false
        install(ContentNegotiation) { json(json) }
    }

    private fun token() = tokens.read(TokenStore.USER_TOKEN)

    /** Whether somebody is signed in. ⚠️ Asked of the store rather than kept as
     *  a field: the token is cleared from more than one place, and a cached
     *  boolean is how a signed-out phone keeps drawing a name. */
    fun signedIn(): Boolean = !token().isNullOrEmpty()

    /** Keep a session. ⚠️ Written through the same store the site's token key
     *  uses (`user_token`), because a guest who signed in on the site and opened
     *  the app is the ordinary case. */
    fun signIn(token: String) = tokens.write(TokenStore.USER_TOKEN, token)

    fun signOut() = tokens.drop(TokenStore.USER_TOKEN)

    private suspend inline fun <reified T> call(
        path: String,
        method: HttpMethod = HttpMethod.Get,
        body: JsonObject? = null,
    ): T {
        val url = Brand.apiBase + path
        val res: HttpResponse = when (method) {
            HttpMethod.Post -> client.post(url) { prepare(body) }
            HttpMethod.Put -> client.put(url) { prepare(body) }
            HttpMethod.Delete -> client.delete(url) { prepare(body) }
            else -> client.get(url) { prepare(null) }
        }
        if (!res.status.isSuccess()) throw errorFrom(res)
        return res.body()
    }

    private fun HttpRequestBuilder.prepare(body: JsonObject?) {
        token()?.let { header("Authorization", "Bearer $it") }
        // ⚠️ **The language travels with every request.** Server-written text —
        // an error, an order status, a push — is translated by the writer, not
        // by the phone (`middleware.Lang`), so a request that did not say which
        // language it is for comes back in Uzbek on a Russian phone.
        header("Accept-Language", lang)
        if (body != null) {
            // ⚠️ **Serialised here and handed over as text.** Giving Ktor a
            // `JsonObject` makes it guess a serializer at runtime; it walks into
            // the object, meets kotlinx's internal `JsonLiteral`, and throws
            // *before the request leaves the device* — which screens then report
            // as "no internet" on a phone with working internet.
            setBody(
                TextContent(
                    json.encodeToString(JsonObject.serializer(), body),
                    ContentType.Application.Json,
                ),
            )
        }
    }

    /** Which language the server should answer in. ⚠️ A field rather than a
     *  parameter on every call: it changes in one place — the header's switch —
     *  and threading it through thirty call sites is thirty chances to forget. */
    @Volatile
    var lang: String = "uz"

    /** ⚠️ **The server's own sentence.** "Bu filialga yetkazib berilmaydi" is
     *  something a guest can act on; "400" is not. */
    private suspend fun errorFrom(res: HttpResponse): ApiError {
        val raw = runCatching { res.bodyAsText() }.getOrDefault("")
        val parsed = runCatching { json.decodeFromString<WireError>(raw) }.getOrNull()
        val message = parsed?.error ?: parsed?.message ?: res.status.description
        return ApiError(res.status.value, message)
    }

    // ---- What the first screen is made of ----

    /** The restaurant itself: its name, its logo, and whether it is open.
     *
     *  ⚠️ Read at launch even though the build carries a name and a colour —
     *  see Models.kt. */
    suspend fun restaurant(): RestaurantResponse = call("/restaurant")

    /** The menu, grouped by category the way the site reads it. */
    suspend fun menu(): List<MenuGroup> = call("/menu")

    // ---- Ordering ----

    /** Which methods this restaurant can actually take money with. */
    /** Every active branch. ⚠️ Pickup must name one: without it the server
     *  quietly sends the order to the first branch, and a guest in a chain
     *  walks to the wrong building. */
    suspend fun branches(): BranchList = call("/brands")

    suspend fun paymentMethods(): PaymentMethods = call("/payment-methods")

    /** Whether this address can be delivered to, by whom, and for how much.
     *
     *  ⚠️ **Asked of the server, always.** The zones are polygons the owner drew
     *  and the fee has four rules behind it; a phone that decided "you are 3 km
     *  away, that is 15 000" would be a second implementation of the one number
     *  a guest complains about. */
    suspend fun deliveryQuote(lat: Double, lng: Double, subtotal: Double): DeliveryQuote =
        call("/delivery/quote", HttpMethod.Post, deliveryQuoteBody(lat, lng, subtotal))

    /** What this basket comes to, with the discounts, the fee and the points.
     *
     *  ⚠️ **Every figure on the checkout comes from here.** Nothing is added up
     *  on the phone: a total the guest computed that disagrees with the one they
     *  are charged is the worst thing this screen can do, and it is the easiest
     *  to cause — a promo rule the app does not know about is enough. */
    suspend fun quote(
        lines: List<CartLine>,
        type: String,
        point: GeoPoint?,
        promoCode: String,
        usePoints: Double,
        branchId: String = "",
    ): OrderQuote = call(
        "/orders/quote",
        HttpMethod.Post,
        quoteBody(lines, type, point, promoCode, usePoints, branchId),
    )

    /** Place it.
     *
     *  ⚠️ **`channel` is "android", and it is the reason the app is worth
     *  shipping to somebody who has to justify it.** Folded into "web" — which
     *  is what the server did before it knew the word — a restaurant cannot
     *  answer "is anybody using the app we paid for" from anywhere in the
     *  panel. Attribution, never authorisation. */
    suspend fun createOrder(
        lines: List<CartLine>,
        type: String,
        name: String,
        phone: String,
        point: GeoPoint?,
        comment: String,
        paymentMethod: String,
        promoCode: String,
        usePoints: Double,
        branchId: String = "",
    ): Order = call(
        "/orders",
        HttpMethod.Post,
        orderBody(lines, type, name, phone, point, comment, paymentMethod, promoCode, usePoints, branchId),
    )

    /** Following one, by the number printed on it.
     *
     *  ⚠️ **Public and keyed by the number**, exactly as the site's tracking page
     *  is: a guest who ordered without signing in has to be able to watch it,
     *  and an order they cannot follow is a phone call to the restaurant. */
    suspend fun order(number: String): Order = call("/orders/$number")

    // ---- The guest's own account ----
    //
    // ⚠️ **Everything above this line works signed out**, and that is the design
    // rather than an accident: the menu, the basket and even placing an order
    // need no account. Signing in adds the guest's history, their points and
    // their favourites — it is never the price of admission.

    /** Ask for a code. ⚠️ The phone number is sent as typed; normalising it here
     *  would be a second implementation of a rule the server already has, and
     *  the two would disagree about the day somebody types a leading zero. */
    suspend fun requestCode(phone: String): CodeSent = call(
        "/auth/phone/request",
        HttpMethod.Post,
        buildJsonObject { put("phone", JsonPrimitive(phone)) },
    )

    /** Hand the code back and get a session.
     *
     *  ⚠️ **The name is sent with it, and only used for a new account.** Asking
     *  a returning guest their name again — on the screen where they are already
     *  proving who they are — is the kind of form that gets abandoned. */
    suspend fun verifyCode(phone: String, code: String, name: String): SignIn = call(
        "/auth/phone/verify",
        HttpMethod.Post,
        buildJsonObject {
            put("phone", JsonPrimitive(phone))
            put("code", JsonPrimitive(code))
            put("name", JsonPrimitive(name))
        },
    )

    suspend fun me(): User = call("/users/me")

    /** This guest's orders, newest first. ⚠️ Joined with what the phone
     *  remembers rather than replacing it — the first order is almost always
     *  placed signed out. */
    suspend fun myOrders(): List<Order> = call("/users/me/orders")

    /** Change the name, or the list of saved addresses.
     *
     *  ⚠️ **The server replaces the whole list**, so the caller sends what the
     *  list should become, never the one row it wants added. Sending a single
     *  address here would delete the other four, and the guest would find out
     *  the next time they ordered. */
    suspend fun updateMe(
        firstName: String,
        lastName: String,
        addresses: List<UserAddress>,
    ): User = call(
        "/users/me",
        HttpMethod.Put,
        buildJsonObject {
            put("firstName", JsonPrimitive(firstName))
            put("lastName", JsonPrimitive(lastName))
            put(
                "addresses",
                JsonArray(
                    addresses.map {
                        buildJsonObject {
                            put("label", JsonPrimitive(it.label))
                            put("text", JsonPrimitive(it.text))
                            put("lat", JsonPrimitive(it.lat))
                            put("lng", JsonPrimitive(it.lng))
                            put("comment", JsonPrimitive(it.comment))
                        }
                    },
                ),
            )
        },
    )

    /** Which language this guest reads, so the server writes its messages in it.
     *
     *  ⚠️ **The language belongs to the account, not to the phone.** Order
     *  updates are written by the server and pushed; without this they arrive
     *  in the restaurant's default language on a phone reading Russian. */
    suspend fun setLang(lang: String) {
        // ⚠️ Swallowed: a guest who switched language and got an error about it
        // would be told off for a preference the phone has already applied.
        runCatching {
            client.put(Brand.apiBase + "/users/me/lang") {
                prepare(buildJsonObject { put("lang", JsonPrimitive(lang)) })
            }
        }
    }

    suspend fun loyalty(): Loyalty = call("/users/me/loyalty")

    /** Keep or drop a dish. Answers with the whole list, so the screen never has
     *  to guess what it now holds. */
    suspend fun toggleFavorite(menuItemId: String): FavoriteState =
        call("/users/me/favorites/$menuItemId", HttpMethod.Post)

    // ---- This phone ----

    /** ⚠️ **The language travels with the token.** A notification is written by
     *  the server, so it is the one piece of text here the phone cannot
     *  translate for itself. */
    suspend fun registerDevice(token: String, lang: String) {
        call<Unit>(
            "/users/me/device",
            HttpMethod.Post,
            buildJsonObject {
                put("token", JsonPrimitive(token))
                put("lang", JsonPrimitive(lang))
                put("platform", JsonPrimitive("android"))
            },
        )
    }

    /** ⚠️ **Called before the token is cleared, on sign-out.** The other order
     *  sends the request unauthenticated, the row stays, and the next person to
     *  hold this phone is told about somebody else's dinner. */
    suspend fun forgetDevice(token: String) {
        call<Unit>(
            "/users/me/device",
            HttpMethod.Delete,
            buildJsonObject { put("token", JsonPrimitive(token)) },
        )
    }
}

/** What the favourites list holds after a tap. */
@kotlinx.serialization.Serializable
data class FavoriteState(
    val on: Boolean = false,
    val favorites: List<String> = emptyList(),
)

// ---- The bodies, built where a test can read them ----
//
// ⚠️ **Separate functions rather than inline `buildJsonObject`, and the reason
// is one bug.** Money is `Double` in this application and `int` on the server,
// and kotlinx writes 5000.0 with the point — which Go refuses for the whole
// request, answering "so'rov formati noto'g'ri". Inline, that was invisible to
// every test: the shape only existed inside a suspend function that needs a
// server. Out here `WireShapeTest` reads it and fails on the next field that
// forgets. See `Cart.quoteJson`.

internal fun deliveryQuoteBody(lat: Double, lng: Double, subtotal: Double): JsonObject =
    buildJsonObject {
        put("lat", JsonPrimitive(lat))
        put("lng", JsonPrimitive(lng))
        put("subtotal", JsonPrimitive(subtotal.toLong()))
    }

internal fun quoteBody(
    lines: List<CartLine>,
    type: String,
    point: GeoPoint?,
    promoCode: String,
    usePoints: Double,
    branchId: String = "",
): JsonObject = buildJsonObject {
    put("type", JsonPrimitive(type))
    // Pickup only: delivery's branch is the address's, decided by the server.
    if (type != "delivery" && branchId.isNotBlank()) put("branchId", JsonPrimitive(branchId))
    put("promoCode", JsonPrimitive(promoCode))
    put("usePoints", JsonPrimitive(usePoints.toLong()))
    if (point != null) {
        put(
            "address",
            buildJsonObject {
                put("lat", JsonPrimitive(point.lat))
                put("lng", JsonPrimitive(point.lng))
            },
        )
    }
    put("items", JsonArray(lines.map { it.quoteJson() }))
}

internal fun orderBody(
    lines: List<CartLine>,
    type: String,
    name: String,
    phone: String,
    point: GeoPoint?,
    comment: String,
    paymentMethod: String,
    promoCode: String,
    usePoints: Double,
    branchId: String = "",
): JsonObject = buildJsonObject {
    if (type != "delivery" && branchId.isNotBlank()) put("branchId", JsonPrimitive(branchId))
    put(
        "customer",
        buildJsonObject {
            put("name", JsonPrimitive(name))
            put("phone", JsonPrimitive(phone))
        },
    )
    put("type", JsonPrimitive(type))
    put(
        "address",
        buildJsonObject {
            put("text", JsonPrimitive(point?.text ?: ""))
            put("lat", JsonPrimitive(point?.lat ?: 0.0))
            put("lng", JsonPrimitive(point?.lng ?: 0.0))
            put("comment", JsonPrimitive(comment))
        },
    )
    put("items", JsonArray(lines.map { it.orderJson() }))
    put("paymentMethod", JsonPrimitive(paymentMethod))
    put("promoCode", JsonPrimitive(promoCode))
    put("usePoints", JsonPrimitive(usePoints.toLong()))
    put("channel", JsonPrimitive("android"))
}
