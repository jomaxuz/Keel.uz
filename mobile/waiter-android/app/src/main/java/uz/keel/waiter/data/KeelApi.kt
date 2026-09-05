package uz.keel.waiter.data

import uz.keel.design.ServerAddress
import uz.keel.design.TokenStore
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
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonPrimitive

/** The server having spoken, whatever it said.
 *
 *  ⚠️ **"The server says no" and "the request never arrived" are opposite
 *  facts that look identical from a `catch`.** One is a sign-in, the other is a
 *  network — and confusing them is how a waiter in a basement met a password
 *  field, typed the right password, and blamed themselves. Anything that is not
 *  an `ApiError` is the request not having arrived. */
class ApiError(
    val status: Int,
    override val message: String,
    /** The refusal's own fields, when it carried any — some refusals are
     *  *requests*: a till action this person may not do answers 403/409 with
     *  the permission's name, so the screen can ask a manager for a PIN instead
     *  of showing a dead end. A dead end is what teaches a room to share one
     *  code. */
    val data: JsonObject = JsonObject(emptyMap()),
) : Exception(message)

/**
 * Every call this app makes, and the four headers that say which install is
 * asking.
 *
 * ⚠️ **The base URL is runtime state, not a build constant.** One binary serves
 * every restaurant: which server it talks to is decided by the account, set once
 * when the phone is handed over. A `BuildConfig.API_URL` would mean one APK per
 * customer — the mistake `NEXT_PUBLIC_*` already made on the web side.
 */
class KeelApi(private val tokens: TokenStore) {

    @Volatile var apiBase: String = ""
        private set
    @Volatile var uploadsBase: String = ""
        private set

    fun useServer(address: String) {
        apiBase = ServerAddress.apiBase(address)
        uploadsBase = ServerAddress.uploadsBase(address)
    }

    /** Which install this is. ⚠️ Sent on every request: the server binds an
     *  account to an install when it sees these (`handlers/logindevice.go`),
     *  which is what stops one login being shared between two people. */
    var device: DeviceInfo? = null

    private val json = Json {
        // ⚠️ The server adds fields constantly, for the panel and the till. A
        // strict decoder would make every unrelated backend release crash this
        // app on a phone in the middle of service.
        ignoreUnknownKeys = true
        explicitNulls = false
        coerceInputValues = true
    }

    private val client = HttpClient(OkHttp) {
        expectSuccess = false
        install(ContentNegotiation) { json(json) }
    }

    // ---- The wire ----

    private suspend inline fun <reified T> call(
        path: String,
        method: HttpMethod = HttpMethod.Get,
        body: JsonObject? = null,
        bearer: String? = null,
    ): T {
        check(apiBase.isNotEmpty()) { "server address not set" }
        val url = apiBase + path
        val res: HttpResponse = when (method) {
            HttpMethod.Post -> client.post(url) { prepare(bearer, body) }
            HttpMethod.Put -> client.put(url) { prepare(bearer, body) }
            HttpMethod.Delete -> client.delete(url) { prepare(bearer, body) }
            else -> client.get(url) { prepare(bearer, null) }
        }
        if (!res.status.isSuccess()) throw errorFrom(res)
        if (res.status.value == 204) return json.decodeFromString("{}")
        return res.body()
    }

    private fun io.ktor.client.request.HttpRequestBuilder.prepare(bearer: String?, body: JsonObject?) {
        if (bearer != null) header("Authorization", "Bearer $bearer")
        device?.let { d ->
            header("X-Keel-Device", d.id)
            header("X-Keel-App", d.app)
            header("X-Keel-Platform", "android")
            // ⚠️ A model name, never a serial: the row exists so somebody in
            // the office can say "that is my old phone", and a string nobody
            // recognises makes the release button a guess.
            if (d.name.isNotEmpty()) header("X-Keel-Device-Name", d.name)
        }
        if (body != null) {
            // ⚠️ **Serialised here and handed over as text, not given to
            // ContentNegotiation to work out.** Passing a `JsonObject` as the
            // body makes Ktor *guess* a serializer at runtime; it walks into the
            // object's elements, meets `JsonLiteral` — kotlinx's own internal
            // type, which has no registered serializer — and throws
            // `SerializationException` **before the request leaves the device**.
            //
            // ⚠️ This failed every POST, PUT and DELETE and nothing else: reads
            // have no body, so the app looked entirely healthy right up to the
            // first write. And because the throw is not an `ApiError`, the
            // screens read it as "the request never arrived" and said "Internet
            // yo'q" — on a phone with working internet, which is the one
            // sentence guaranteed to send somebody to look in the wrong place.
            //
            // `TextContent` is already an `OutgoingContent`, so content
            // negotiation leaves it alone entirely.
            setBody(TextContent(json.encodeToString(JsonObject.serializer(), body),
                ContentType.Application.Json))
        }
    }

    /** ⚠️ **The server's own words, in the reader's own language.** Error text is
     *  written in Uzbek at the call site and translated by the server from the
     *  `lang` cookie / header — "lag'mon bugun tugadi" is an answer a waiter can
     *  take back to the table, and replacing it with "Xatolik" throws away the
     *  only useful thing in the response. */
    private suspend fun errorFrom(res: HttpResponse): ApiError {
        val raw = runCatching { res.bodyAsText() }.getOrDefault("")
        val obj = runCatching { json.decodeFromString<JsonObject>(raw) }.getOrNull()
        val message = obj?.get("error")?.jsonPrimitive?.contentOrNull
            ?: obj?.get("message")?.jsonPrimitive?.contentOrNull
            ?: res.status.description
        return ApiError(res.status.value, message, obj ?: JsonObject(emptyMap()))
    }

    private fun staff() = tokens.staffToken

    // ---- Staff (/staff) ----

    suspend fun staffLogin(username: String, password: String): LoginResponse =
        call("/staff/login", HttpMethod.Post, buildJsonObject {
            put("username", JsonPrimitive(username))
            put("password", JsonPrimitive(password))
        })

    suspend fun staffMe(): StaffMe = call("/staff/me", bearer = staff())

    suspend fun staffReport(): StaffReport = call("/staff/report", bearer = staff())

    /** ⚠️ Position is mandatory — the server checks it against the branch and
     *  refuses from too far away (`geofenceBlocked`). The device that can answer
     *  "where am I" is the one in the employee's hand, which is why this moved
     *  onto the phone at all. */
    suspend fun staffClock(action: String, lat: Double, lng: Double, accuracy: Double): Shift =
        call("/staff/clock", HttpMethod.Post, buildJsonObject {
            put("action", JsonPrimitive(action))
            put("lat", JsonPrimitive(lat))
            put("lng", JsonPrimitive(lng))
            put("accuracy", JsonPrimitive(accuracy))
            put("code", JsonPrimitive(""))
        }, bearer = staff())

    /** ⚠️ **`platform` is "android" and the token is an FCM one, not Expo's.**
     *  The backend currently relays every notification through Expo
     *  (`internal/push/expo.go`), which will not accept this token — see the
     *  README, "Push: backendga tegadigan yagona ish". */
    suspend fun registerPush(token: String, lang: String): Unit =
        call("/staff/push", HttpMethod.Post, buildJsonObject {
            put("token", JsonPrimitive(token))
            put("platform", JsonPrimitive("android"))
            put("lang", JsonPrimitive(lang))
            put("app", JsonPrimitive("waiter"))
        }, bearer = staff())

    /** ⚠️ Called on sign-out, and it is not tidiness: a token left behind sends
     *  the next evening's tables to whoever went home, and they cannot switch it
     *  off from their side. Awaited **before** the token is cleared, or the
     *  request goes out unauthenticated and the row stays. */
    suspend fun forgetPush(token: String): Unit =
        call("/staff/push", HttpMethod.Delete, buildJsonObject {
            put("token", JsonPrimitive(token))
        }, bearer = staff())

    // ---- The floor and the checks (/staff/checks) ----

    suspend fun branch(): BranchInfo = call("/staff/branch", bearer = staff())

    suspend fun checks(): ChecksResponse = call("/staff/checks", bearer = staff())

    suspend fun check(id: String): Check = call("/staff/checks/$id", bearer = staff())

    suspend fun openCheck(tableId: String, guests: Int = 0): Check =
        call("/staff/checks", HttpMethod.Post, buildJsonObject {
            put("tableId", JsonPrimitive(tableId))
            put("guests", JsonPrimitive(guests))
        }, bearer = staff())

    suspend fun updateCheck(id: String, guests: Int): Check =
        call("/staff/checks/$id", HttpMethod.Put, buildJsonObject {
            put("guests", JsonPrimitive(guests))
        }, bearer = staff())

    /** ⚠️ **Many lines in one call, and that is the fix for "the menu cannot be
     *  pressed quickly".** One request per tap meant four coffees were four
     *  round trips on a restaurant's wifi, and for those two seconds the menu
     *  answered nothing — so the waiter tapped again. Taps are gathered for a
     *  moment and sent as one. */
    /** @param opId the phone's own id for this tap, when it is resending
     *  something it queued. ⚠️ **Only this endpoint takes one.** Every other
     *  operation states an absolute and is harmless to repeat; this one says
     *  "one more", and a resend the phone could not know had arrived is a guest
     *  charged twice. The server remembers it (`Order.AppliedOps`) and answers a
     *  repeat with the check as it stands — success, not an error, or the phone
     *  would queue it again forever. */
    suspend fun addLines(id: String, items: List<AddLineRequest>, opId: String = ""): Check =
        call("/staff/checks/$id/lines", HttpMethod.Post, buildJsonObject {
            put("items", json.encodeToJsonElement(ListSerializer(AddLineRequest.serializer()), items))
            if (opId.isNotEmpty()) put("opId", JsonPrimitive(opId))
        }, bearer = staff())

    /** ⚠️ **Only the quantity is sent.** The same endpoint writes the guest's
     *  note, and the server tells "no comment in this request" from "clear the
     *  comment" by the field being *absent* — so pressing "+" must not carry an
     *  empty one, or it wipes "piyozsiz". */
    suspend fun lineQty(id: String, lineId: String, qty: Int): Check =
        call("/staff/checks/$id/lines/$lineId", HttpMethod.Put, buildJsonObject {
            put("qty", JsonPrimitive(qty))
        }, bearer = staff())

    suspend fun commentLine(id: String, lineId: String, comment: String): Check =
        call("/staff/checks/$id/lines/$lineId", HttpMethod.Put, buildJsonObject {
            put("comment", JsonPrimitive(comment))
        }, bearer = staff())

    /** Remove a line. ⚠️ On an unfired line this is a typo being corrected and
     *  needs no reason; on a fired one the food has been cooked and the server
     *  demands a reason and may demand a manager's code. That difference is the
     *  oldest way money leaves a restaurant and is not ours to blur. */
    suspend fun voidLine(id: String, lineId: String, reason: String = "", pin: String = ""): Check =
        call("/staff/checks/$id/lines/$lineId", HttpMethod.Delete, buildJsonObject {
            if (reason.isNotEmpty()) put("reason", JsonPrimitive(reason))
            if (pin.isNotEmpty()) put("pin", JsonPrimitive(pin))
        }, bearer = staff())

    suspend fun lineServed(id: String, lineId: String, served: Boolean): Check =
        call("/staff/checks/$id/lines/$lineId/served", HttpMethod.Put, buildJsonObject {
            put("served", JsonPrimitive(served))
        }, bearer = staff())

    /** Send to the pass what has not been sent. ⚠️ Separate from adding a dish
     *  on purpose: typing is not ordering. */
    suspend fun fire(id: String): Check =
        call("/staff/checks/$id/fire", HttpMethod.Post, bearer = staff())

    /** ⚠️ A courtesy, not the mechanism: the hold expires by itself after a
     *  couple of minutes, which is what makes it safe. This only shortens the
     *  wait when somebody walked away rather than crashed. */
    suspend fun releaseCheck(id: String): Unit =
        call("/staff/checks/$id/release", HttpMethod.Post, bearer = staff())

    suspend fun moveLines(id: String, lineIds: List<String>, toCheckId: String): Check =
        call("/staff/checks/$id/lines/move", HttpMethod.Post, buildJsonObject {
            put("lineIds", json.encodeToJsonElement(ListSerializer(String.serializer()), lineIds))
            put("toCheckId", JsonPrimitive(toCheckId))
        }, bearer = staff())

    /** ⚠️ Returns **both** halves: the source keeps the screen (the waiter is
     *  still standing at that table) and the new one has to appear in the room
     *  immediately, or it reads as food that vanished. */
    suspend fun split(id: String, lineIds: List<String>): SplitResponse =
        call("/staff/checks/$id/split", HttpMethod.Post, buildJsonObject {
            put("lineIds", json.encodeToJsonElement(ListSerializer(String.serializer()), lineIds))
        }, bearer = staff())

    suspend fun merge(id: String, intoId: String): Check =
        call("/staff/checks/$id/merge", HttpMethod.Post, buildJsonObject {
            put("intoId", JsonPrimitive(intoId))
        }, bearer = staff())

    /** ⚠️ `precheck` also **records** that the table has been given its bill —
     *  the floor screen draws that, and "asked twenty minutes ago" is a
     *  different situation from "asked just now". */
    suspend fun print(id: String, kind: String = "precheck"): PrintResponse =
        call("/staff/checks/$id/print", HttpMethod.Post, buildJsonObject {
            put("kind", JsonPrimitive(kind))
        }, bearer = staff())

    // ---- Public ----

    suspend fun menu(branchId: String): List<MenuGroup> =
        call("/menu?branchId=$branchId")
}

data class DeviceInfo(val id: String, val app: String, val name: String)
