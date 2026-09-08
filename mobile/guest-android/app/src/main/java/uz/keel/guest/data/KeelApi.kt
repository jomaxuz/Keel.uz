package uz.keel.guest.data

import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.request.HttpRequestBuilder
import io.ktor.client.request.get
import io.ktor.client.request.header
import io.ktor.client.request.post
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
import kotlinx.serialization.json.JsonObject
import uz.keel.design.TokenStore
import uz.keel.guest.Brand

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

    private suspend inline fun <reified T> call(
        path: String,
        method: HttpMethod = HttpMethod.Get,
        body: JsonObject? = null,
    ): T {
        val url = Brand.apiBase + path
        val res: HttpResponse = when (method) {
            HttpMethod.Post -> client.post(url) { prepare(body) }
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
    suspend fun restaurant(): Restaurant = call("/restaurant")

    /** The menu, grouped by category the way the site reads it. */
    suspend fun menu(): List<MenuGroup> = call("/menu")
}
