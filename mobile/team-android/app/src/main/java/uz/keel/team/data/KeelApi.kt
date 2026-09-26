package uz.keel.team.data

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
 *  that look identical from a `catch`.** One is a sign-in, the other is a
 *  basement — and confusing them is how somebody meets a password field on a
 *  phone with no signal, types the right password, watches it fail, and blames
 *  themselves. */
class ApiError(val status: Int, override val message: String) : Exception(message)

/** One line of a market run. */
data class BuyLine(
    val ingredientId: String?,
    val newName: String?,
    val qty: Double,
    val price: Double,
    /** ⚠️ Whether the figures count packs. A flag, not a converted number: the
     *  factor is a fact about the ingredient and the result lands on a shelf,
     *  so the server does the arithmetic. */
    val pack: Boolean,
)

/** One line of a shopping list being written. */
data class OrderLine(
    val ingredientId: String?,
    val name: String,
    val qty: Double,
    val pack: Boolean,
)

/**
 * Every call this application makes.
 *
 * ⚠️ **The staff token** (`staff_token`, the browser's spelling). It opens what
 * an employee may do: their own attendance, their own report, and — where the
 * account carries the permission — the market run.
 */
class KeelApi(private val tokens: TokenStore) {

    @Volatile
    var apiBase: String = ""
        private set

    fun useServer(address: String) {
        apiBase = ServerAddress.apiBase(address)
    }

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

    private fun token() = tokens.read(TokenStore.STAFF_TOKEN)

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
        // ⚠️ **The language on every request, read from the store rather than
        // held here.** Server-written text — a refusal, an order status, the
        // owner's briefing — is translated by the writer (`middleware.Lang`),
        // and a phone has no cookie for it to read the choice from. Taken from
        // the same key `Prefs` writes, so the two cannot drift: a field set at
        // startup would be the field somebody forgets to update when the
        // language is changed on the settings screen.
        header("X-Keel-Lang", tokens.read(TokenStore.LANG) ?: "uz")
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
            // report as "no internet" on a phone with working internet.
            setBody(
                TextContent(
                    json.encodeToString(JsonObject.serializer(), body),
                    ContentType.Application.Json,
                ),
            )
        }
    }

    /** ⚠️ **The server's own sentence.** "You are 400 m from the branch" is
     *  something somebody can act on, and it is the one refusal on this app
     *  that is not a fault. */
    private suspend fun errorFrom(res: HttpResponse): ApiError {
        val raw = runCatching { res.bodyAsText() }.getOrDefault("")
        val parsed = runCatching { json.decodeFromString<WireError>(raw) }.getOrNull()
        val message = parsed?.error ?: parsed?.message ?: res.status.description
        return ApiError(res.status.value, message)
    }

    // ---- Getting in ----

    suspend fun login(username: String, password: String): LoginResult = call(
        "/staff/login",
        HttpMethod.Post,
        auth = false,
        body = buildJsonObject {
            put("username", JsonPrimitive(username))
            put("password", JsonPrimitive(password))
        },
    )

    suspend fun me(): StaffMe = call("/staff/me")

    // ---- The shift ----

    /** Open or close a shift, from the device that can answer "where am I".
     *
     *  ⚠️ **The position travels with the punch.** The server compares it to the
     *  branch's own point (`geofenceBlocked`), and a punch without one is the
     *  reason this screen had to move off the web page it lived on. */
    suspend fun clock(action: String, lat: Double, lng: Double, accuracy: Double): Shift = call(
        "/staff/clock",
        HttpMethod.Post,
        body = buildJsonObject {
            put("action", JsonPrimitive(action))
            put("lat", JsonPrimitive(lat))
            put("lng", JsonPrimitive(lng))
            put("accuracy", JsonPrimitive(accuracy))
        },
    )

    suspend fun report(): StaffReport = call("/staff/report")

    // ---- The market ----

    suspend fun buyList(): ShoppingList = call("/staff/buy/list")

    suspend fun buyCatalog(): BuyCatalog = call("/staff/buy/catalog")

    suspend fun buyBalance(): AdvanceBalance = call("/staff/buy/balance")

    suspend fun buyOrders(openOnly: Boolean = false): ShoppingOrders =
        call("/staff/buy/orders" + if (openOnly) "?open=1" else "")

    suspend fun buyOrderDraft(): ShoppingDraft = call("/staff/buy/orders/draft")

    /** Record a free-form run.
     *
     *  ⚠️ **`clientId` is the whole of offline safety here.** A market has worse
     *  signal than a dining room: without an id minted before the first attempt,
     *  a resend after a timeout is a second delivery — the shelf raised twice,
     *  the invoice paid twice, the same price written into the history twice. */
    suspend fun buyCreate(clientId: String, supplier: String, lines: List<BuyLine>): BuyResult =
        call(
            "/staff/buy",
            HttpMethod.Post,
            body = buildJsonObject {
                put("clientId", JsonPrimitive(clientId))
                put("supplier", JsonPrimitive(supplier))
                put(
                    "lines",
                    JsonArray(
                        lines.map { l ->
                            buildJsonObject {
                                l.ingredientId?.let { put("ingredientId", JsonPrimitive(it)) }
                                l.newName?.let { put("newName", JsonPrimitive(it)) }
                                put("qty", JsonPrimitive(l.qty))
                                put("price", JsonPrimitive(Math.round(l.price)))
                                put("pack", JsonPrimitive(l.pack))
                            }
                        },
                    ),
                )
            },
        )

    /** Write the list, and let the server decide who answers each line.
     *
     *  ⚠️ **It comes back as one list or two.** The person writing it never
     *  chooses: what has to be bought goes to the buyer and what is already in
     *  the building goes to the storekeeper, and sorting that is knowledge about
     *  the store rather than about an empty bar. */
    suspend fun createBuyOrder(forDate: String, lines: List<OrderLine>): CreatedOrders = call(
        "/staff/buy/orders",
        HttpMethod.Post,
        body = buildJsonObject {
            put("forDate", JsonPrimitive(forDate))
            put(
                "lines",
                JsonArray(
                    lines.map { l ->
                        buildJsonObject {
                            l.ingredientId?.let { put("ingredientId", JsonPrimitive(it)) }
                            put("name", JsonPrimitive(l.name))
                            put("qty", JsonPrimitive(l.qty))
                            put("pack", JsonPrimitive(l.pack))
                        }
                    },
                ),
            )
        },
    )

    /** Tick one line of a list.
     *
     *  ⚠️ **The shelf does not move here.** Ticking is a fact about the list; the
     *  stock only changes when the trip is finished — a line that raised it on a
     *  tick would put food on a shelf while the buyer was still at the market,
     *  and an untick would then have to take it off, which nothing downstream
     *  could tell from a theft. */
    suspend fun markOrderLine(
        orderId: String,
        lineId: String,
        qty: Double? = null,
        price: Double? = null,
        pack: Boolean? = null,
        missing: Boolean? = null,
        clear: Boolean? = null,
    ): ShoppingOrder = call(
        "/staff/buy/orders/$orderId/lines/$lineId",
        HttpMethod.Put,
        body = buildJsonObject {
            qty?.let { put("qty", JsonPrimitive(it)) }
            // Whole so'm: a Double goes out as "30000.0".
            price?.let { put("price", JsonPrimitive(Math.round(it))) }
            pack?.let { put("pack", JsonPrimitive(it)) }
            missing?.let { put("missing", JsonPrimitive(it)) }
            clear?.let { put("clear", JsonPrimitive(it)) }
        },
    )

    /** My half is done and it is on its way.
     *
     *  ⚠️ **Not a delivery.** Nothing reaches a shelf here: what the buyer says
     *  he handed over is a claim and what the restaurant counted is the fact, so
     *  the purchase is written when somebody signs for it. See
     *  handlers/buyorderflow.go.
     *
     *  ⚠️ Posted to `/ship`; the old `/finish` path is the same handler on the
     *  server, kept for phones that have not updated. */
    suspend fun shipOrder(orderId: String): FinishResult =
        call("/staff/buy/orders/$orderId/ship", HttpMethod.Post, body = buildJsonObject {})

    /** Count what turned up and sign for it.
     *
     *  ⚠️ **Send only the rows whose figure differs.** A row left out keeps what
     *  it was told — accepting without retyping means "this is right", which is
     *  the ordinary case, and a default of zero would empty the whole list of
     *  whoever was quickest to agree with it.
     *
     *  ⚠️ `clientId` is the offline guarantee the market run already has: a
     *  phone with no signal retries, and without it the retry is a second
     *  delivery. */
    suspend fun acceptOrder(
        orderId: String,
        clientId: String,
        counted: Map<String, Double>,
    ): FinishResult = call(
        "/staff/buy/orders/$orderId/accept",
        HttpMethod.Post,
        body = buildJsonObject {
            put("clientId", JsonPrimitive(clientId))
            put(
                "lines",
                JsonArray(
                    counted.map { (lineId, qty) ->
                        buildJsonObject {
                            put("lineId", JsonPrimitive(lineId))
                            put("qty", JsonPrimitive(qty))
                        }
                    },
                ),
            )
        },
    )

    // ---- Marking codes, read with the camera ----

    /** What carries a marking code, so the phone can ask what is in the box.
     *
     *  ⚠️ Only flagged products: a list of the whole menu would bury the six
     *  things this screen is for under four hundred that carry no code. */
    suspend fun markItems(): MarkItems = call("/staff/marking/items")

    /** File what the camera read.
     *
     *  ⚠️ **Sent in batches rather than one at a time.** A store room is where a
     *  phone has no signal, and a request per bottle turns one bad minute into
     *  forty failures. ⚠️ Every code is still answered separately by the server
     *  — a box with one unreadable sticker is not a failed box. */
    suspend fun receiveMarks(menuItemId: String, codes: List<String>): MarkResult = call(
        "/staff/marking/receive",
        HttpMethod.Post,
        body = buildJsonObject {
            if (menuItemId.isNotEmpty()) put("menuItemId", JsonPrimitive(menuItemId))
            put("codes", JsonArray(codes.map { JsonPrimitive(it) }))
        },
    )

    /** How many of this product the branch is still holding, by code. */
    suspend fun markHeld(menuItemId: String): MarkHeld =
        call("/staff/marking/stock" + if (menuItemId.isEmpty()) "" else "?menuItemId=$menuItemId")

    // ---- The shop's own labels ----

    /** What needs a sticker: no code at all, never printed, or a changed price. */
    suspend fun labelCandidates(): LabelCandidates = call("/staff/labels/candidates")

    /** Queue stickers.
     *
     *  ⚠️ **How many is the person's answer.** A crate of forty may need forty
     *  stickers or one for the shelf edge, and only the person holding it knows
     *  which — so the number is asked for and never inferred from a delivery. */
    suspend fun printLabels(copies: Map<String, Int>): LabelResult = call(
        "/staff/labels",
        HttpMethod.Post,
        body = buildJsonObject {
            put(
                "items",
                JsonArray(
                    copies.map { (id, n) ->
                        buildJsonObject {
                            put("id", JsonPrimitive(id))
                            put("copies", JsonPrimitive(n))
                        }
                    },
                ),
            )
        },
    )

    // ---- Counting the store ----

    /** The rooms this employee's branch keeps things in.
     *
     *  ⚠️ **Asked before the sheet.** A count is one room, and the phone has to
     *  know which one before it can ask what is in it. */
    suspend fun warehouses(): Warehouses = call("/staff/warehouses")

    /** What to count in one room.
     *
     *  ⚠️ **The store is a query parameter and an empty one is a real answer** —
     *  the undivided store, which is every restaurant that never split one.
     *  Sending an empty `warehouseId` is deliberate: the server reads a blank as
     *  that store, and leaving the parameter off entirely means the same thing. */
    suspend fun stocktakeSheet(warehouseId: String): StocktakeSheet =
        call("/staff/stocktake/sheet" + if (warehouseId.isEmpty()) "" else "?warehouseId=$warehouseId")

    /** Record a count.
     *
     *  ⚠️ **Only the rows somebody actually typed into.** A blank box is "not
     *  reached yet", never zero — sending untouched rows as zero reads the next
     *  morning as a catastrophic shortfall, and it is the classic way a
     *  half-finished count gets saved anyway.
     *
     *  ⚠️ **The branch is not sent and could not be.** The server takes it off
     *  the employee (handlers/staffstock.go): a count resets the baseline every
     *  later shortfall is measured from, so a phone that could name a branch
     *  could reset somebody else's.
     *
     *  ⚠️ **The store is omitted when it is the undivided one**, rather than
     *  sent as an empty string: Go decodes the body into an ObjectID, and `""`
     *  is a parse error rather than a zero id — the count would be refused with
     *  a message about a field nobody chose. */
    suspend fun saveStocktake(
        warehouseId: String,
        counted: Map<String, Double>,
        note: String,
    ): SavedCount = call(
        "/staff/stocktake",
        HttpMethod.Post,
        body = buildJsonObject {
            if (warehouseId.isNotEmpty()) put("warehouseId", JsonPrimitive(warehouseId))
            put("note", JsonPrimitive(note))
            put(
                "lines",
                JsonArray(
                    counted.map { (ingredientId, qty) ->
                        buildJsonObject {
                            put("ingredientId", JsonPrimitive(ingredientId))
                            put("counted", JsonPrimitive(qty))
                        }
                    },
                ),
            )
        },
    )

    // ---- This phone ----

    /** ⚠️ **The language travels with the token.** A notification is written by
     *  the server, so it is the one piece of text here the phone cannot
     *  translate for itself. */
    suspend fun registerPush(token: String, lang: String) {
        call<JsonObject>(
            "/staff/push",
            HttpMethod.Post,
            body = buildJsonObject {
                put("token", JsonPrimitive(token))
                put("platform", JsonPrimitive("android"))
                put("lang", JsonPrimitive(lang))
            },
        )
    }

    /** ⚠️ Called on sign-out: a token left behind sends somebody else's pay slip
     *  to a phone that has changed hands. */
    suspend fun forgetPush(token: String) {
        call<JsonObject>(
            "/staff/push",
            HttpMethod.Delete,
            body = buildJsonObject { put("token", JsonPrimitive(token)) },
        )
    }
}
