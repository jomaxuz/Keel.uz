package uz.keel.app

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import uz.keel.app.data.Workspace
import uz.keel.app.i18n.UZ
import uz.keel.app.i18n.WORDS
import uz.keel.app.ui.KeelRoot
import uz.keel.app.ui.LocalShellWords
import uz.keel.app.ui.Route
import uz.keel.design.DesignWords
import uz.keel.design.KeelBackground
import uz.keel.design.KeelWaiterTheme
import uz.keel.design.Lang
import uz.keel.design.LangHost
import uz.keel.design.LocalLangHost
import uz.keel.design.LocalNotice
import uz.keel.design.LocalWords
import uz.keel.design.Note
import uz.keel.design.NoticeHost

// Keel's one activity.
//
// ⚠️ **One activity for four roles, and no router.** Each role brought its own
// single level of navigation and keeps it; above them there is exactly one
// decision — which workspace — and it lives in `KeelRoot`. A navigation library
// here would be a dependency carrying that one decision.
class MainActivity : ComponentActivity() {

    /** A tap from outside — a notification, the courier's shift — held in state
     *  rather than read once, because `singleTask` delivers a second tap through
     *  `onNewIntent` while this activity is alive, which is the common case. */
    private var pending by mutableStateOf<Route?>(null)

    override fun onCreate(savedInstanceState: Bundle?) {
        // ⚠️ **Before `super.onCreate`**: it swaps the splash theme for the
        // app's own. And no keep-on-screen condition — the only thing worth
        // waiting for (the saved accounts) was read in `KeelApp.onCreate`,
        // and a splash held for a network answer is a hang in a basement.
        installSplashScreen()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        pending = routeOf(intent)

        val app = application as KeelApp

        setContent {
            val notice = remember { mutableStateOf<Note?>(null) }
            val words = WORDS[app.look.lang.value] ?: UZ
            // ⚠️ **The shell's own locals, which every role then overrides with
            // its own.** The design module asks for the language host and its
            // half-dozen words through the composition; a control under a
            // provider that forgot them fails when pressed, not when compiled.
            CompositionLocalProvider(
                LocalNotice provides notice,
                LocalShellWords provides words,
                LocalLangHost provides object : LangHost {
                    override val current: Lang get() = app.look.lang.value
                    override fun nameOf(l: Lang): String = WORDS[l]!!.lang
                    override fun set(l: Lang) = app.look.setLang(l)
                },
                LocalWords provides DesignWords(ok = "OK", retry = words.retry, loading = words.loading),
            ) {
                KeelWaiterTheme(app.look.theme.value) {
                    KeelBackground {
                        KeelRoot(app, pending) { pending = null }
                        NoticeHost(notice)
                    }
                }
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        routeOf(intent)?.let { pending = it }
    }

    /** ⚠️ **The same keys whether this service drew the notification or the
     *  system did.** With the app in the background Firebase draws it from the
     *  `notification` block and copies the `data` keys into the tap — so
     *  `channel`, `checkId` and `tab` arrive here under the server's names,
     *  which is why the service uses those names too. */
    private fun routeOf(intent: Intent?): Route? {
        val x = intent?.extras ?: return null
        val checkId = x.getString(EXTRA_CHECK_ID)
        val channel = x.getString(EXTRA_CHANNEL)
        val tab = x.getString(EXTRA_TAB)
        if (checkId == null && channel == null) return null
        val ws = if (checkId != null) Workspace.Waiter else Workspace.ofChannel(channel)
        // The single-role apps opened these two on a fixed tab; kept.
        val defaultTab = when (ws) {
            Workspace.Courier -> "orders"
            Workspace.Team -> "profile"
            else -> null
        }
        return Route(ws, checkId, tab ?: defaultTab)
    }

    companion object {
        const val EXTRA_CHANNEL = "channel"
        const val EXTRA_CHECK_ID = "checkId"
        const val EXTRA_TAB = "tab"
    }
}
