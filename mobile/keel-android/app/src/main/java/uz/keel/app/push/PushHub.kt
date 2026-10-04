package uz.keel.app.push

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.google.firebase.FirebaseApp
import com.google.firebase.messaging.FirebaseMessaging
import kotlinx.coroutines.tasks.await
import uz.keel.app.KeelLook
import uz.keel.app.data.AccountStore
import uz.keel.app.data.Kind
import uz.keel.app.data.ShellApi
import uz.keel.app.data.Workspace
import uz.keel.design.fetchPushToken
import uz.keel.design.pushErrorDetail

// One phone, one Firebase token, up to three accounts listening on it.
//
// ⚠️ **Registered for every account at once, not when a role is opened.** In
// the single-role apps registration happened on the role's first screen, which
// was the only screen. Here an owner who also works the floor may spend all day
// in the floor — and the owner's loss alerts must still arrive. So every
// signed-in account gets this phone's row as soon as there is a token, on every
// launch (the token can be re-issued) and whenever the language changes (the
// server writes the words, so the language travels with the row).
//
// ⚠️ **Signing one account out never deletes the token.** The single-role apps
// did, as the last step of signing out, and there it was right. Here the token
// is shared: deleting it would silence the other accounts on the phone with
// nothing anywhere saying so. Only the account's own row is dropped; the token
// itself goes when the last account does.
class PushHub(
    private val context: Context,
    private val api: ShellApi,
    private val accounts: AccountStore,
    private val look: KeelLook,
) {
    /** `PushState.key` in every role's spelling. */
    var key by mutableStateOf("asking")
        private set
    var detail by mutableStateOf<String?>(null)
        private set
    /** Bumped by "retry" on a settings screen; the shell re-runs on it. */
    var nonce by mutableIntStateOf(0)
        private set

    private var token: String? = null

    val state: KeelPush get() = KeelPush(key, detail, ::retry)

    fun retry() {
        nonce += 1
    }

    fun denied() {
        key = "denied"
        detail = null
    }

    /** Subscribe every signed-in account. Safe to call again: the server keys
     *  the row on the token and moves it to whoever registers it. */
    suspend fun sync() {
        val address = accounts.address ?: return
        val list = accounts.accounts.value
        if (list.isEmpty()) return
        // ⚠️ **Its own state, and the likeliest one on a first build.** The
        // Firebase plugin is applied only when `google-services.json` names
        // `uz.keel.app` — a step taken in the Firebase console. Without it
        // there is no project, and "failed" would send somebody hunting for a
        // network fault.
        if (FirebaseApp.getApps(context).isEmpty()) {
            key = "noProject"
            detail = "google-services.json: uz.keel.app"
            return
        }
        key = "asking"
        detail = null
        try {
            val value = fetchPushToken { FirebaseMessaging.getInstance().token.await() }
            token = value
            var firstError: Throwable? = null
            for (a in list) {
                // ⚠️ A staff row says which section it is for, because the
                // server picks the Android channel from it. A waiter's row is
                // the floor's: the kitchen's "ready" is the message that cannot
                // wait, and it is the one that must not land on a quieter
                // channel.
                val app = when (a.kindOf) {
                    Kind.Staff -> if (Workspace.Waiter in a.workspaces()) "waiter" else "team"
                    else -> null
                }
                try {
                    api.registerPush(address, a.kindOf, value, look.lang.value.code, app)
                } catch (e: Throwable) {
                    if (firstError == null) firstError = e
                }
            }
            if (firstError != null) {
                key = "failed"
                detail = pushErrorDetail(firstError)
            } else {
                key = "working"
            }
        } catch (e: Throwable) {
            key = "failed"
            detail = pushErrorDetail(e)
        }
    }

    /** Drop this phone's row for one account — **before** its token is
     *  cleared, or the request goes out unauthenticated and the row stays. */
    suspend fun forget(kind: Kind) {
        val address = accounts.address ?: return
        // The token this process registered, or the one Firebase holds — a
        // sign-out in the first seconds after launch comes before any sync.
        val value = token ?: runCatching {
            if (FirebaseApp.getApps(context).isEmpty()) null
            else FirebaseMessaging.getInstance().token.await()
        }.getOrNull() ?: return
        runCatching { api.forgetPush(address, kind, value) }
        val othersLeft = accounts.accounts.value.any { it.kindOf != kind }
        if (!othersLeft) {
            token = null
            runCatching { FirebaseMessaging.getInstance().deleteToken().await() }
            key = "asking"
            detail = null
        }
    }
}
