package uz.keel.owner.data

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
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.contentOrNull
import uz.keel.design.ServerAddress
import uz.keel.design.TokenStore

/** The server having spoken, whatever it said.
 *
 *  ⚠️ **"The server says no" and "the request never arrived" are opposite facts
 *  that look identical from a `catch`.** One is a sign-in, the other is a
 *  network — and confusing them is how somebody meets a password field on a
 *  phone with no signal, types the right password, and blames themselves. */
class ApiError(
    val status: Int,
    override val message: String,
    val data: JsonObject = JsonObject(emptyMap()),
) : Exception(message)

/**
 * Every call this application makes.
 *
 * ⚠️ **The panel's own account, not a till's.** The owner app signs in with
 * `admin_token` — the same credential the browser panel uses — because it is the
 * same person looking at the same business from a different device. A separate
 * identity would mean a second password to lose and a second permission set to
 * disagree with the first.
 *
 * ⚠️ **The branch lens travels on the query string**, exactly as the panel's
 * does (`?branchId=`). The server clamps it to what the account may see, so a
 * manager cannot widen their view by editing a request — which is why the lens
 * is allowed to be a client-side convenience at all.
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

    /** Which install this is — the four headers that let the server bind an
     *  account to one phone (`handlers/logindevice.go`). */
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

    private fun token() = tokens.read(TokenStore.ADMIN_TOKEN)

    // ---- The wire ----

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
            // cost the waiter app an afternoon; it is not repeated here.
            setBody(TextContent(json.encodeToString(JsonObject.serializer(), body),
                ContentType.Application.Json))
        }
    }

    /** ⚠️ **The server's own words, in the reader's own language.** The API
     *  writes its refusals in Uzbek and translates them from the request; the
     *  useful half of a 409 is the sentence, not the number. */
    private suspend fun errorFrom(res: HttpResponse): ApiError {
        val raw = runCatching { res.bodyAsText() }.getOrDefault("")
        val obj = runCatching { json.decodeFromString<JsonObject>(raw) }.getOrNull()
        val message = obj?.get("error")?.jsonPrimitive?.contentOrNull
            ?: obj?.get("message")?.jsonPrimitive?.contentOrNull
            ?: res.status.description
        return ApiError(res.status.value, message, obj ?: JsonObject(emptyMap()))
    }

    /** The branch lens, as a query suffix. Empty for "every branch", which is
     *  what a single-branch restaurant always sends and never sees. */
    private fun scope(branchId: String, extra: String = ""): String {
        val parts = buildList {
            if (branchId.isNotEmpty()) add("branchId=$branchId")
            if (extra.isNotEmpty()) add(extra)
        }
        return if (parts.isEmpty()) "" else "?" + parts.joinToString("&")
    }

    // ---- Getting in ----

    suspend fun login(username: String, password: String): LoginResponse =
        call("/admin/login", HttpMethod.Post, buildJsonObject {
            put("username", JsonPrimitive(username))
            put("password", JsonPrimitive(password))
        }, auth = false)

    suspend fun me(): AdminUser = call("/admin/me")

    suspend fun branches(): List<Branch> = call("/admin/branches")

    // ---- Today ----

    /** @param from/@param to a local calendar day, `YYYY-MM-DD`.
     *
     *  ⚠️ **The day is decided on the phone and sent as a date**, never as a
     *  timestamp: every report in this product is filed by local midnight, and a
     *  moment in time would be re-interpreted in UTC on the way. */
    suspend fun stats(from: String, to: String, branchId: String): AdminStats =
        call("/admin/stats" + scope(branchId, "from=$from&to=$to"))

    /** ⚠️ Asked separately from the numbers, and that is deliberate: this screen
     *  is opened twenty times a day for the takings and read once for the
     *  briefing, so an answer the tariff does not include must not drag today's
     *  revenue down with it. */
    suspend fun insights(branchId: String, lang: String): BriefingResponse =
        call("/admin/insights" + scope(branchId, "lang=$lang"))

    suspend fun staff(branchId: String): List<StaffRow> = call("/admin/staff" + scope(branchId))

    // ---- What is waiting, and what happened ----

    suspend fun alerts(): AdminAlerts = call("/admin/alerts")

    suspend fun lossAlerts(): LossAlerts = call("/admin/alerts/loss")

    // ---- Orders ----

    suspend fun orders(status: String = "", limit: Int = 50, branchId: String = ""): List<Order> {
        val extra = buildList {
            if (status.isNotEmpty()) add("status=$status")
            add("limit=$limit")
        }.joinToString("&")
        return call("/admin/orders" + scope(branchId, extra))
    }

    /** ⚠️ A cancellation carries a reason and the server insists on one: the
     *  guest reads it on the tracking page, and "cancelled" with no sentence is
     *  the message that produces a phone call. */
    suspend fun setOrderStatus(id: String, status: String, reason: String = ""): StatusResponse =
        call("/admin/orders/$id/status", HttpMethod.Put, buildJsonObject {
            put("status", JsonPrimitive(status))
            if (reason.isNotEmpty()) put("cancelReason", JsonPrimitive(reason))
        })

    // ---- Guests' words ----

    suspend fun feedback(branchId: String, handled: Boolean): FeedbackList =
        call("/admin/feedback" + scope(branchId, "handled=" + if (handled) "1" else "0"))

    /** ⚠️ The note is required by the server too: "handled" with nothing written
     *  is a row nobody can act on a week later. */
    suspend fun handleFeedback(id: String, note: String): Feedback =
        call("/admin/feedback/$id/handled", HttpMethod.Post, buildJsonObject {
            put("note", JsonPrimitive(note))
        })

    // ---- The report ----

    suspend fun payroll(branchId: String): PayrollResponse = call("/admin/payroll" + scope(branchId))

    suspend fun shoppingList(branchId: String): ShoppingList =
        call("/admin/stock/shopping-list" + scope(branchId))

    suspend fun subscription(): Subscription = call("/admin/subscription")

    // ---- Help ----

    /** ⚠️ **Asked of the server rather than bundled**, which is what makes it one
     *  copy: the panel reads the same endpoint. A Kotlin copy of the base would
     *  be the copy that stops describing this build the first time an article is
     *  corrected and only two of the three are edited. */
    suspend fun helpArticles(lang: String): HelpArticles =
        call("/admin/support/articles?lang=$lang")

    suspend fun supportThreads(): SupportThreads = call("/admin/support/threads")

    suspend fun supportThread(id: String): SupportThreadView =
        call("/admin/support/thread?id=$id")

    /** Send a line. An empty `threadId` starts a new conversation.
     *
     *  ⚠️ **The ranked articles travel with the question, and that is what the
     *  assistant answers from** — it is given these and nothing else
     *  (`handlers/supportai.go`). Sending none is a supported state: the message
     *  still reaches a person, it simply gets no immediate answer. */
    suspend fun supportAsk(
        threadId: String,
        text: String,
        lang: String,
        articles: List<AskArticle>,
    ): SupportAskResponse =
        call("/admin/support/ask", HttpMethod.Post, buildJsonObject {
            if (threadId.isNotEmpty()) put("threadId", JsonPrimitive(threadId))
            put("text", JsonPrimitive(text))
            put("lang", JsonPrimitive(lang))
            put("articles", json.encodeToJsonElement(
                ListSerializer(AskArticle.serializer()), articles,
            ))
        })

    // ---- Notifications ----

    suspend fun registerPush(token: String, lang: String): Unit =
        call("/admin/push", HttpMethod.Post, buildJsonObject {
            put("token", JsonPrimitive(token))
            put("platform", JsonPrimitive("android"))
            put("lang", JsonPrimitive(lang))
        })

    /** ⚠️ Called on sign-out **before** the token is cleared, or the request goes
     *  out unauthenticated and the row stays — sending tomorrow's loss alerts to
     *  a phone that has changed hands. */
    suspend fun forgetPush(token: String): Unit =
        call("/admin/push", HttpMethod.Delete, buildJsonObject {
            put("token", JsonPrimitive(token))
        })
}

data class DeviceInfo(val id: String, val app: String, val name: String)
