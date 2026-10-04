package uz.keel.app.data

import io.ktor.client.HttpClient
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.request.HttpRequestBuilder
import io.ktor.client.request.delete
import io.ktor.client.request.get
import io.ktor.client.request.header
import io.ktor.client.request.post
import io.ktor.client.request.setBody
import io.ktor.client.statement.HttpResponse
import io.ktor.client.statement.bodyAsText
import io.ktor.http.ContentType
import io.ktor.http.content.TextContent
import io.ktor.http.isSuccess
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import uz.keel.app.AppDevice
import uz.keel.design.ServerAddress
import uz.keel.design.TokenStore

/** The server having spoken, whatever it said — never the network failing.
 *
 *  ⚠️ The same split every Keel app makes: "wrong password" and "no signal"
 *  look identical from a `catch` and send somebody to opposite fixes. */
class ShellError(val status: Int, override val message: String) : Exception(message)

/** One account the sign-in opened. */
class Opened(val account: Account, val token: String)

/** What a sign-in came back with: what opened, and what the server refused
 *  for an account the password did match (switched off, held by another
 *  phone) — said, not dropped, because "why is my courier tab missing?" has
 *  exactly that answer. */
class SignInResult(val opened: List<Opened>, val refused: List<Pair<Kind, String>>)

/** The few calls Keel makes itself: getting in, and the phone's push row for
 *  each account. Every other request belongs to a role and goes through that
 *  role's own `KeelApi`, unchanged. */
class ShellApi(private val tokens: TokenStore, private val device: (Kind?) -> AppDevice) {

    private val json = Json { ignoreUnknownKeys = true; explicitNulls = false; coerceInputValues = true }
    private val client = HttpClient(OkHttp) { expectSuccess = false }

    /** Every account this login and password open, in one request.
     *
     *  ⚠️ **One form for four apps, and the server answers which.** Before
     *  Keel, a person chose the app and the app chose the table to look in; the
     *  owner who was handed the courier's app typed the right password into the
     *  wrong door and was told it was wrong. `/app/login` looks in all three.
     *
     *  ⚠️ **A server older than `/app/login` still signs in.** Restaurants are
     *  updated one by one, and the phone often moves first: a 404 or 405 here
     *  falls back to the three logins the single-role apps use, in turn, with
     *  Keel's own device keys — slower by two requests, and only on that
     *  server. */
    suspend fun signIn(address: String, username: String, password: String): SignInResult {
        val base = ServerAddress.apiBase(address)
        require(base.isNotEmpty()) { "bad address" }
        val body = buildJsonObject {
            put("username", JsonPrimitive(username))
            put("password", JsonPrimitive(password))
        }
        val res = client.post("$base/app/login") { prepare(null, null, body) }
        if (res.status.value == 404 || res.status.value == 405) return legacySignIn(base, body)
        if (!res.status.isSuccess()) throw errorFrom(res)
        val root = json.parseToJsonElement(res.bodyAsText()).jsonObject
        val opened = mutableListOf<Opened>()
        val refused = mutableListOf<Pair<Kind, String>>()
        for (el in root["accounts"]?.jsonArray ?: JsonArray(emptyList())) {
            val o = el.jsonObject
            val kind = Kind.of(o.str("kind")) ?: continue
            val token = o.str("token")
            if (token.isEmpty()) {
                refused += kind to o.str("error")
                continue
            }
            val inner = when (kind) {
                Kind.Admin -> o["admin"]
                Kind.Staff -> o["staff"]
                Kind.Courier -> o["courier"]
            }?.let { runCatching { it.jsonObject }.getOrNull() }
            opened += Opened(accountOf(kind, inner, fallbackName = o.str("name"), role = o.str("role")), token)
        }
        return SignInResult(opened, refused)
    }

    private suspend fun legacySignIn(base: String, body: JsonObject): SignInResult {
        val opened = mutableListOf<Opened>()
        val refused = mutableListOf<Pair<Kind, String>>()
        var firstNo: ShellError? = null
        val tries = listOf(
            Triple(Kind.Admin, "/admin/login", "user"),
            Triple(Kind.Staff, "/staff/login", "staff"),
            Triple(Kind.Courier, "/courier/login", "courier"),
        )
        for ((kind, path, field) in tries) {
            val res = client.post(base + path) { prepare(kind, null, body) }
            if (res.status.isSuccess()) {
                val o = json.parseToJsonElement(res.bodyAsText()).jsonObject
                val token = o.str("token")
                if (token.isNotEmpty()) {
                    val inner = runCatching { o[field]?.jsonObject }.getOrNull()
                    opened += Opened(accountOf(kind, inner, "", inner?.str("role") ?: ""), token)
                }
                continue
            }
            val err = errorFrom(res)
            // 401 is "not this table" — the expected answer from two of the
            // three. Anything else is the password matching and the account
            // being refused, which is worth saying.
            if (err.status == 401) {
                if (firstNo == null) firstNo = err
            } else {
                refused += kind to err.message
            }
        }
        if (opened.isEmpty()) {
            refused.firstOrNull()?.let { throw ShellError(409, it.second) }
            throw firstNo ?: ShellError(401, "")
        }
        return SignInResult(opened, refused)
    }

    /** A fresher copy of the staff account — its permissions are what decide
     *  which workspaces it opens, and they are changed in the office. */
    suspend fun staffMe(address: String): Account? {
        val base = ServerAddress.apiBase(address)
        val token = tokens.read(Kind.Staff.tokenKey) ?: return null
        val res = client.get("$base/staff/me") { prepare(Kind.Staff, token, null) }
        if (!res.status.isSuccess()) return null
        val o = json.parseToJsonElement(res.bodyAsText()).jsonObject
        return accountOf(Kind.Staff, runCatching { o["staff"]?.jsonObject }.getOrNull(), "", "")
    }

    /** This phone's push row for one account.
     *
     *  @param app for a staff account, which of its two sections the row is
     *  for — the server picks the Android channel from it. */
    suspend fun registerPush(address: String, kind: Kind, fcm: String, lang: String, app: String?) {
        val token = tokens.read(kind.tokenKey) ?: return
        val body = buildJsonObject {
            put("token", JsonPrimitive(fcm))
            put("platform", JsonPrimitive("android"))
            put("lang", JsonPrimitive(lang))
            if (app != null) put("app", JsonPrimitive(app))
        }
        val res = client.post(ServerAddress.apiBase(address) + pushPath(kind)) { prepare(kind, token, body) }
        if (!res.status.isSuccess()) throw errorFrom(res)
    }

    /** ⚠️ Called **before** the account's token is dropped — see
     *  `RoleExit.signOut`. Failure is swallowed by the caller: the server prunes
     *  a dead row the first time it fails to deliver to it. */
    suspend fun forgetPush(address: String, kind: Kind, fcm: String) {
        val token = tokens.read(kind.tokenKey) ?: return
        val body = buildJsonObject { put("token", JsonPrimitive(fcm)) }
        client.delete(ServerAddress.apiBase(address) + pushPath(kind)) { prepare(kind, token, body) }
    }

    private fun pushPath(kind: Kind) = when (kind) {
        Kind.Admin -> "/admin/push"
        Kind.Staff -> "/staff/push"
        Kind.Courier -> "/courier/push"
    }

    private fun HttpRequestBuilder.prepare(kind: Kind?, bearer: String?, body: JsonObject?) {
        if (bearer != null) header("Authorization", "Bearer $bearer")
        header("X-Keel-Lang", tokens.read(TokenStore.LANG) ?: "uz")
        val d = device(kind)
        header("X-Keel-Device", d.id)
        header("X-Keel-App", d.app)
        header("X-Keel-Platform", "android")
        if (d.name.isNotEmpty()) header("X-Keel-Device-Name", d.name)
        if (body != null) {
            setBody(TextContent(json.encodeToString(JsonObject.serializer(), body), ContentType.Application.Json))
        }
    }

    private suspend fun errorFrom(res: HttpResponse): ShellError {
        val raw = runCatching { res.bodyAsText() }.getOrDefault("")
        val obj = runCatching { json.parseToJsonElement(raw).jsonObject }.getOrNull()
        val message = obj?.str("error")?.takeIf { it.isNotEmpty() }
            ?: obj?.str("message")?.takeIf { it.isNotEmpty() }
            ?: ""
        return ShellError(res.status.value, message)
    }

    private fun accountOf(kind: Kind, o: JsonObject?, fallbackName: String, role: String): Account {
        val name = o?.str("name")?.takeIf { it.isNotBlank() }
            ?: o?.str("username")?.takeIf { it.isNotBlank() }
            ?: fallbackName
        return Account(
            kind = kind.code,
            name = name,
            role = role.ifEmpty { o?.str("role") ?: "" },
            position = o?.str("position") ?: "",
            roleName = o?.str("roleName")?.takeIf { it.isNotEmpty() },
            perms = o?.get("perms")?.let { el ->
                runCatching { el.jsonArray.mapNotNull { it.jsonPrimitive.contentOrNull } }.getOrNull()
            } ?: emptyList(),
            canWaiter = o?.bool("canWaiter") ?: false,
            canCashier = o?.bool("canCashier") ?: false,
        )
    }

    private fun JsonObject.str(key: String): String =
        (this[key] as? JsonPrimitive)?.contentOrNull ?: ""

    private fun JsonObject.bool(key: String): Boolean? =
        (this[key] as? JsonPrimitive)?.booleanOrNull
}
