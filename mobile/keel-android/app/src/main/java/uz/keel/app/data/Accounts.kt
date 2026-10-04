package uz.keel.app.data

import androidx.compose.runtime.Immutable
import androidx.compose.runtime.MutableState
import androidx.compose.runtime.mutableStateOf
import kotlinx.serialization.Serializable
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.json.Json
import uz.keel.design.TokenStore

// Who is signed in on this phone, and what each of them may open.
//
// ⚠️ **The tokens stay where the four apps kept them** (`admin_token`,
// `staff_token`, `courier_token`) and this list only describes them. The role
// code under `uz.keel.<role>` reads its token by that name and was not
// rewritten for Keel; a second copy of the token here would be one more thing
// to clear on sign-out, and the one that gets forgotten is a session that
// outlives the person.
//
// ⚠️ **One account per kind, not per person.** The server has three separate
// account tables and a person can hold one of each — an owner who also takes a
// courier run has two passwords, or one password on two rows. The phone holds
// at most one of each, because each token has exactly one slot, and a second
// owner signing in replaces the first rather than sitting beside them.

/** The three account tables on the server, and where each one's token lives. */
enum class Kind(val code: String, val tokenKey: String) {
    Admin("admin", TokenStore.ADMIN_TOKEN),
    Staff("staff", TokenStore.STAFF_TOKEN),
    Courier("courier", TokenStore.COURIER_TOKEN);

    companion object {
        fun of(code: String): Kind? = entries.firstOrNull { it.code == code }
    }
}

/** What somebody opens: a role's screens. Two of them belong to one account —
 *  the floor and the team section are both a staff login. */
enum class Workspace(val key: String, val kind: Kind) {
    Owner("owner", Kind.Admin),
    Waiter("waiter", Kind.Staff),
    Courier("courier", Kind.Courier),
    Team("team", Kind.Staff);

    companion object {
        fun of(key: String?): Workspace? = entries.firstOrNull { it.key == key }

        /** The notification channel a message was sent on, as the server names
         *  it (`internal/push` → `KitchenChannel` …), to the workspace that
         *  answers it. */
        fun ofChannel(channel: String?): Workspace? = when (channel) {
            "kitchen" -> Waiter
            "delivery" -> Courier
            "owner" -> Owner
            "team" -> Team
            else -> null
        }
    }
}

/** One signed-in account, as much of it as choosing a workspace needs.
 *
 *  ⚠️ **No token in here** — see the note at the top of the file. */
@Immutable
@Serializable
data class Account(
    val kind: String,
    val name: String,
    /** `owner` or `manager` for the panel's accounts; empty otherwise. */
    val role: String = "",
    /** The staff member's job, as the office wrote it ("Ofitsiant"). */
    val position: String = "",
    val roleName: String? = null,
    val perms: List<String> = emptyList(),
    val canWaiter: Boolean = false,
    val canCashier: Boolean = false,
) {
    val kindOf: Kind get() = Kind.of(kind) ?: Kind.Staff

    /** What this account opens, in the order it is offered.
     *
     *  ⚠️ **The floor is offered to whoever the till would let carry a plate**
     *  — the server's own rule (`models.Staff.Can(PermWaiter)`): the role's
     *  list when there is one, the pre-role flags when there is not, and
     *  cashier implying waiter. A rule of its own here would show a waiter a
     *  floor the server then refuses, which is worse than not showing it.
     *
     *  ⚠️ **The team section is for everybody else, and for a waiter only when
     *  it holds something the floor does not.** A waiter's own hours and shift
     *  are already on the floor's profile tab; offering them twice is a choice
     *  between two identical doors. A waiter who also does the market run, or
     *  counts the store, gets both. */
    fun workspaces(): List<Workspace> = when (kindOf) {
        Kind.Admin -> listOf(Workspace.Owner)
        Kind.Courier -> listOf(Workspace.Courier)
        Kind.Staff -> buildList {
            val waiter = if (perms.isNotEmpty()) {
                "waiter" in perms || "cashier" in perms
            } else {
                canWaiter || canCashier
            }
            if (waiter) add(Workspace.Waiter)
            if (!waiter || perms.any { it in TEAM_PERMS }) add(Workspace.Team)
        }
    }

    companion object {
        /** What the team section has tabs for beyond the profile — the
         *  `can…Here` checks under `uz.keel.team.ui.screens`. */
        val TEAM_PERMS = setOf("buy", "buyorder", "stockissue", "stock", "marking", "label")
    }
}

/** The list, persisted beside the tokens it describes. */
class AccountStore(private val tokens: TokenStore) {

    private val json = Json { ignoreUnknownKeys = true; explicitNulls = false }

    /** ⚠️ **Only accounts whose token is still there.** A role clears its own
     *  token when the server refuses it — a list that trusted itself would offer
     *  a workspace that opens straight onto "sign in again". */
    val accounts: MutableState<List<Account>> = mutableStateOf(load())

    /** The workspace open last, so a launch goes back to it instead of asking. */
    val last: MutableState<Workspace?> = mutableStateOf(Workspace.of(tokens.read(LAST)))

    val address: String? get() = tokens.serverAddress

    fun workspaces(): List<Workspace> =
        accounts.value.flatMap { it.workspaces() }.distinct().sortedBy { it.ordinal }

    fun accountFor(w: Workspace): Account? = accounts.value.firstOrNull { it.kindOf == w.kind }

    fun signedIn(kind: Kind): Boolean = accounts.value.any { it.kindOf == kind }

    fun put(account: Account, token: String) {
        tokens.write(account.kindOf.tokenKey, token)
        accounts.value = accounts.value.filter { it.kindOf != account.kindOf } + account
        save()
    }

    /** A fresher copy of an account already here — perms changed in the
     *  office. ⚠️ Never adds: a refresh racing a sign-out must not bring the
     *  account back. */
    fun refresh(account: Account) {
        if (!signedIn(account.kindOf)) return
        accounts.value = accounts.value.map { if (it.kindOf == account.kindOf) account else it }
        save()
    }

    fun drop(kind: Kind) {
        tokens.drop(kind.tokenKey)
        accounts.value = accounts.value.filter { it.kindOf != kind }
        save()
    }

    fun dropAll() {
        Kind.entries.forEach { tokens.drop(it.tokenKey) }
        accounts.value = emptyList()
        save()
    }

    fun useServer(address: String) {
        tokens.serverAddress = address
    }

    fun forgetServer() {
        tokens.serverAddress = null
    }

    fun remember(w: Workspace) {
        last.value = w
        tokens.write(LAST, w.key)
    }

    private fun load(): List<Account> {
        val raw = tokens.read(KEY) ?: return emptyList()
        val list = runCatching { json.decodeFromString(ListSerializer(Account.serializer()), raw) }
            .getOrDefault(emptyList())
        return list.filter { !tokens.read(it.kindOf.tokenKey).isNullOrEmpty() }
            .distinctBy { it.kindOf }
    }

    private fun save() {
        tokens.write(KEY, json.encodeToString(ListSerializer(Account.serializer()), accounts.value))
    }

    companion object {
        const val KEY = "keel_accounts"
        const val LAST = "keel_workspace"
    }
}
