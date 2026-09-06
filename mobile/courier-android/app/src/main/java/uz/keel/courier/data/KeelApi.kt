package uz.keel.courier.data

import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
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
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import uz.keel.design.ServerAddress
import uz.keel.design.TokenStore

/** The server having spoken, whatever it said.
 *
 *  ⚠️ **"The server says no" and "the request never arrived" are opposite facts
 *  that look identical from a `catch`**, and on this app they decide what a
 *  courier is shown: one is a sign-in, the other is a basement. Confusing them
 *  is how somebody meets a password field on a phone with no signal, types the
 *  right password, watches it fail, and blames themselves. */
class ApiError(val status: Int, override val message: String) : Exception(message)

/**
 * Every call this application makes.
 *
 * ⚠️ **The courier's own token, not a till's** (`courier_token`, the browser's
 * spelling). It opens exactly what a courier may do: read their own orders,
 * move one along, report where they are.
 */
class KeelApi(private val tokens: TokenStore) {

    @Volatile
    var apiBase: String = ""
        private set

    fun useServer(address: String) {
        apiBase = ServerAddress.apiBase(address)
    }

    /** Which install this is — the headers that let the panel say which phone
     *  an account is bound to (`handlers/logindevice.go`). */
    var device: DeviceInfo? = null

    private val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
        coerceInputValues = true
    }

    private val client = HttpClient(OkHttp) {
        expectSuccess = false
        install(ContentNegotiation) { json(json) }
    }

    private fun token() = tokens.read(TokenStore.COURIER_TOKEN)

    private suspend inline fun <reified T> call(
        path: String,
        method: HttpMethod = HttpMethod.Get,
        body: JsonObject? = null,
        auth: Boolean = true,
    ): T {
        check(apiBase.isNotEmpty()) { "server address not set" }
        val url = apiBase + path
        val res: HttpResponse = when (method) {
            HttpMethod.Post -> client.post(url) { prepare(auth, body) }
            HttpMethod.Put -> client.put(url) { prepare(auth, body) }
            HttpMethod.Delete -> client.delete(url) { prepare(auth, body) }
            else -> client.get(url) { prepare(auth, null) }
        }
        if (!res.status.isSuccess()) throw errorFrom(res)
        return res.body()
    }

    private fun io.ktor.client.request.HttpRequestBuilder.prepare(auth: Boolean, body: JsonObject?) {
        if (auth) token()?.let { header("Authorization", "Bearer $it") }
        device?.let { d ->
            header("X-Keel-Device", d.id)
            header("X-Keel-App", d.app)
            header("X-Keel-Platform", "android")
            if (d.name.isNotEmpty()) header("X-Keel-Device-Name", d.name)
        }
        if (body != null) {
            // ⚠️ **Serialised here and handed over as text.** Giving Ktor a
            // `JsonObject` makes it guess a serializer at runtime; it walks into
            // the object, meets kotlinx's internal `JsonLiteral`, and throws
            // *before the request leaves the device* — which the screens then
            // report as "no internet", on a phone with working internet. That
            // cost the waiter app an afternoon.
            setBody(
                TextContent(
                    json.encodeToString(JsonObject.serializer(), body),
                    ContentType.Application.Json,
                ),
            )
        }
    }

    /** ⚠️ **The server's own sentence, in the courier's language.** The API
     *  writes its refusals in Uzbek and translates them from the request, and
     *  the useful half of the arrival refusal is the sentence — "you are 300 m
     *  away" — not the status code. */
    private suspend fun errorFrom(res: HttpResponse): ApiError {
        val raw = runCatching { res.bodyAsText() }.getOrDefault("")
        val parsed = runCatching { json.decodeFromString<WireError>(raw) }.getOrNull()
        val message = parsed?.error ?: parsed?.message ?: res.status.description
        return ApiError(res.status.value, message)
    }

    // ---- Getting in ----

    suspend fun login(username: String, password: String): LoginResult = call(
        "/courier/login",
        HttpMethod.Post,
        auth = false,
        body = buildJsonObject {
            put("username", JsonPrimitive(username))
            put("password", JsonPrimitive(password))
        },
    )

    suspend fun me(): Courier = call("/courier/me")

    // ---- The shift ----

    suspend fun setStatus(status: String) {
        call<JsonObject>(
            "/courier/status",
            HttpMethod.Put,
            body = buildJsonObject { put("status", JsonPrimitive(status)) },
        )
    }

    /** ⚠️ **A batch, because the phone buffers.** A courier rides through
     *  basements and lifts; the points that could not be sent wait rather than
     *  being dropped, and the server keeps the newest one anyway. */
    suspend fun sendLocation(points: List<Point>) {
        call<JsonObject>(
            "/courier/location",
            HttpMethod.Post,
            body = buildJsonObject {
                put(
                    "points",
                    JsonArray(
                        points.map { p ->
                            buildJsonObject {
                                put("lat", JsonPrimitive(p.lat))
                                put("lng", JsonPrimitive(p.lng))
                                put("accuracy", JsonPrimitive(p.accuracy))
                                put("at", JsonPrimitive(p.at))
                            }
                        },
                    ),
                )
            },
        )
    }

    // ---- The evening's work ----

    suspend fun orders(): List<Order> = call("/courier/orders")

    /** ⚠️ The one call that can be refused for a reason the courier can act on:
     *  the server checks where they are before it accepts "delivered". */
    suspend fun advance(id: String, status: String) {
        call<JsonObject>(
            "/courier/orders/$id/status",
            HttpMethod.Put,
            body = buildJsonObject { put("status", JsonPrimitive(status)) },
        )
    }

    suspend fun stats(): CourierStats = call("/courier/stats")

    suspend fun history(): List<CourierOrderRow> = call("/courier/history")

    // ---- This phone ----

    /** ⚠️ **The language travels with the token.** A notification is written by
     *  the server, so it is the one piece of text here the phone cannot
     *  translate for itself. */
    suspend fun registerPush(token: String, lang: String) {
        call<JsonObject>(
            "/courier/push",
            HttpMethod.Post,
            body = buildJsonObject {
                put("token", JsonPrimitive(token))
                put("platform", JsonPrimitive("android"))
                put("lang", JsonPrimitive(lang))
            },
        )
    }

    /** ⚠️ Called on sign-out, and it is not tidiness: a courier's phone is the
     *  one most likely to be sold or handed on, and a token left behind
     *  delivers customer names, phones and addresses to whoever holds it next. */
    suspend fun forgetPush(token: String) {
        call<JsonObject>(
            "/courier/push",
            HttpMethod.Delete,
            body = buildJsonObject { put("token", JsonPrimitive(token)) },
        )
    }

    // ---- The rule the button follows ----

    /** How close the restaurant wants a courier before "delivered" opens.
     *
     *  ⚠️ **Asked, never assumed.** A city-centre restaurant may want 100 m and
     *  a village one 500 or nothing at all, and the number here has to be the
     *  number the server will judge by — a button that opens and is then refused
     *  is a broken app, read in front of a customer. */
    suspend fun arrivalRadiusM(): Int =
        call<RestaurantEnvelope>("/restaurant", auth = false).restaurant.delivery.arrivalRadiusM
}
