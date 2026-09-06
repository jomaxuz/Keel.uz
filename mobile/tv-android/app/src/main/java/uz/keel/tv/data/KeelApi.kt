package uz.keel.tv.data

import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.request.get
import io.ktor.client.request.header
import io.ktor.client.request.post
import io.ktor.client.request.setBody
import io.ktor.client.statement.HttpResponse
import io.ktor.client.statement.bodyAsText
import io.ktor.http.ContentType
import io.ktor.http.content.TextContent
import io.ktor.http.encodeURLParameter
import io.ktor.http.isSuccess
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import uz.keel.design.ServerAddress
import uz.keel.design.TokenStore

/** The server having spoken, whatever it said.
 *
 *  ⚠️ **"The server says no" and "the request never arrived" are opposite facts
 *  that look identical from a `catch`**, and on this application they are the
 *  two states the whole design turns on: one means somebody unpaired this screen
 *  from the panel and it must show a code again, the other means the wifi
 *  blinked and it must keep playing. Getting them the wrong way round is a
 *  dining room full of guests looking at a six-character pairing code. */
class ApiError(val status: Int, override val message: String) : Exception(message)

/**
 * Every call a television makes. Five of them.
 *
 * ⚠️ **Two are made without a token, and that is not an oversight.** A set being
 * paired has no identity yet — that is what pairing is for. What protects those
 * two is the code's short life and the poll secret, never a bearer.
 */
class KeelApi(private val tokens: TokenStore) {

    @Volatile
    var apiBase: String = ""
        private set

    fun useServer(address: String) {
        apiBase = ServerAddress.apiBase(address)
    }

    /** Which install this is — the headers that let the panel say which
     *  television a row belongs to (`handlers/logindevice.go`). */
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

    private fun token() = tokens.read(TokenStore.TV_TOKEN)

    private suspend inline fun <reified T> call(
        path: String,
        post: Boolean = false,
        body: JsonObject? = null,
        auth: Boolean = true,
    ): T {
        check(apiBase.isNotEmpty()) { "server address not set" }
        val url = apiBase + path
        val res: HttpResponse =
            if (post) client.post(url) { prepare(auth, body) }
            else client.get(url) { prepare(auth, null) }
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
            // *before the request leaves the device* — which this app would
            // then read as "the network is down" on a working network, and go
            // on showing a dead pairing code. The waiter build lost an
            // afternoon to it.
            setBody(
                TextContent(
                    json.encodeToString(JsonObject.serializer(), body),
                    ContentType.Application.Json,
                ),
            )
        }
    }

    /** ⚠️ **The server's own words.** The API writes its refusals in Uzbek and
     *  translates them from the request; the useful half of a 401 is the
     *  sentence, not the number — even though this application shows neither to
     *  the room. */
    private suspend fun errorFrom(res: HttpResponse): ApiError {
        val raw = runCatching { res.bodyAsText() }.getOrDefault("")
        val parsed = runCatching { json.decodeFromString<WireError>(raw) }.getOrNull()
        val message = parsed?.error ?: parsed?.message ?: res.status.description
        return ApiError(res.status.value, message)
    }

    // ---- Before there is an identity ----

    suspend fun pairStart(installId: String, appVersion: String): TVPairStart =
        call(
            "/tv/pair/start",
            post = true,
            auth = false,
            body = buildJsonObject {
                put("installId", JsonPrimitive(installId))
                put("appVersion", JsonPrimitive(appVersion))
            },
        )

    /** ⚠️ **Answered by the poll secret, never by the code.** The code is
     *  readable by every guest in the room; if it were enough to ask this
     *  question, anybody with a phone could poll faster than the app and collect
     *  the token meant for the wall. */
    suspend fun pairStatus(installId: String, pollSecret: String): TVPairStatus =
        call(
            "/tv/pair/status?installId=${installId.encodeURLParameter()}" +
                "&pollSecret=${pollSecret.encodeURLParameter()}",
            auth = false,
        )

    // ---- Once it has one ----

    /** The heartbeat: who am I, am I still paired, and what time is it there.
     *
     *  ⚠️ A 401 means the screen was unpaired from the panel. The answer is to
     *  forget the token and show a code — never to retry, which would leave a
     *  set that was deliberately taken out of service still playing. */
    suspend fun me(appVersion: String): TVMe =
        call("/tv/me?appVersion=${appVersion.encodeURLParameter()}")

    suspend fun playlist(): TVPlaylistWire = call("/tv/playlist")

    suspend fun board(): TVBoardWire = call("/tv/board")
}
