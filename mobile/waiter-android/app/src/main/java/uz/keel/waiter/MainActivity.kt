package uz.keel.waiter

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.BackHandler
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.material.icons.Icons
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.rounded.CloudUpload
import androidx.compose.material.icons.rounded.GridView
import androidx.compose.material.icons.rounded.Person
import androidx.compose.material.icons.rounded.Settings
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewmodel.compose.viewModel
import kotlinx.coroutines.launch
import uz.keel.design.DesignWords
import uz.keel.design.LocalLangHost
import uz.keel.design.LocalWords
import uz.keel.waiter.push.rememberPushRegistration
import uz.keel.design.*
import uz.keel.waiter.ui.screens.CheckScreen
import uz.keel.waiter.ui.screens.FloorScreen
import uz.keel.waiter.ui.screens.LoginScreen
import uz.keel.waiter.ui.screens.OfflineScreen
import uz.keel.waiter.ui.screens.ProfileScreen
import uz.keel.waiter.ui.screens.ServerScreen
import uz.keel.waiter.ui.screens.SettingsScreen

// Keel Waiter.
//
// ⚠️ **One level of navigation, held here, rather than a router.** The check is
// the only place a tab leads to, and the phone's back button has nothing else to
// mean. A navigation library at this size is a dependency carrying one decision;
// it goes in the moment there is a third destination.

class MainActivity : ComponentActivity() {

    /** A check id arriving from a tapped notification. Held in state rather than
     *  read once, because `singleTask` delivers a second tap through
     *  `onNewIntent` while this activity is already alive — which is exactly the
     *  common case: the app is open, the kitchen ticks a dish. */
    private var pendingCheckId by mutableStateOf<String?>(null)

    override fun onCreate(savedInstanceState: Bundle?) {
        // ⚠️ **Before `super.onCreate`, and it swaps the theme as well as drawing
        // the splash.** The activity is declared with `Theme.KeelWaiter.Splash`
        // so the launcher has something branded to show instantly; this hands it
        // over to the app's own theme. Called later, the app opens on the splash
        // theme and the first frame carries the wrong window background.
        //
        // ⚠️ **No `setKeepOnScreenCondition`, deliberately.** The only thing a
        // splash here could wait for is the saved tokens, and those are read
        // synchronously in `KeelWaiterApp.onCreate` — before this line. The
        // tempting thing to hold for is the "who is signed in?" request, and that
        // would be a hang: in a basement or on a dead wifi it does not answer,
        // and this app has an entire screen for saying so. A splash that
        // outlasts its reason is an app somebody force-quits.
        installSplashScreen()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        pendingCheckId = intent?.getStringExtra(EXTRA_CHECK_ID)

        val app = application as KeelWaiterApp

        setContent {
            val notice = remember { mutableStateOf<Note?>(null) }
            // ⚠️ **Provided here or the shared controls throw.** The design
            // module deliberately does not know this app — it asks for the
            // language and for its own half-dozen words through the
            // composition, and a `LangSwitch` under a provider that forgot them
            // fails at the moment somebody presses it, not at compile time.
            CompositionLocalProvider(
                LocalPrefs provides app.prefs,
                LocalNotice provides notice,
                LocalLangHost provides app.prefs.langHost(),
                LocalWords provides DesignWords(
                    ok = app.prefs.dict.notice.ok,
                    retry = app.prefs.dict.common.retry,
                    loading = app.prefs.dict.common.loading,
                ),
            ) {
                KeelWaiterTheme(app.prefs.theme.value) {
                    KeelBackground {
                        Root(app, pendingCheckId) { pendingCheckId = null }
                        NoticeHost(notice)
                    }
                }
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        pendingCheckId = intent.getStringExtra(EXTRA_CHECK_ID)
    }

    companion object {
        const val EXTRA_CHECK_ID = "checkId"
    }
}

private data class Open(val checkId: String, val branchId: String)

@Composable
private fun Root(app: KeelWaiterApp, pendingCheckId: String?, onConsumed: () -> Unit) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()

    val vm: SessionViewModel = viewModel(
        factory = object : ViewModelProvider.Factory {
            @Suppress("UNCHECKED_CAST")
            override fun <T : ViewModel> create(modelClass: Class<T>): T =
                SessionViewModel(app.api, app.tokens) as T
        },
    )
    val session by vm.state.collectAsState()

    var tab by remember { mutableStateOf("floor") }
    var open by remember { mutableStateOf<Open?>(null) }

    val ready = session as? Session.Ready
    val branchId = ready?.staff?.branchId ?: ""

    // ⚠️ Registered once signed in, not at launch: a permission prompt on the
    // first screen is asked before anybody knows what the app is for.
    val push = rememberPushRegistration(app.api, ready != null, app.prefs.lang.value.code)

    // A notification tap opens the table rather than the room: somebody reading
    // it on the move has already decided where they are going.
    LaunchedEffect(pendingCheckId, ready) {
        val id = pendingCheckId ?: return@LaunchedEffect
        if (ready == null) return@LaunchedEffect
        open = Open(id, branchId)
        onConsumed()
    }

    // The phone's back button: out of the check, into the room. ⚠️ Only while a
    // check is open, so back from the room still leaves the app — a waiter who
    // cannot put their phone away is a waiter fighting it.
    BackHandler(enabled = open != null) { open = null }

    val bottomInset = WindowInsets.navigationBars.asPaddingValues()
    // ⚠️ **The bar's own height *plus* the system's strip, not one or the
    // other.** The tab bar applies `navigationBarsPadding` to itself, so it sits
    // that much higher than the bottom of the layout — a list padded only by the
    // system inset still ends with its last dish behind the glass, and the last
    // dish is the one somebody just added. 56 of bar + 20 of its margins.
    val tabsInset = PaddingValues(bottom = 76.dp + bottomInset.calculateBottomPadding())

    Box(Modifier.fillMaxSize()) {
        AnimatedContent(
            targetState = session::class to (open?.checkId ?: "") to tab,
            transitionSpec = { fadeIn() togetherWith fadeOut() },
            label = "root",
        ) { _ ->
            when (val s = session) {
                is Session.Loading -> Box(Modifier.fillMaxSize(), Alignment.Center) {
                    CircularProgressIndicator(color = c.accent)
                }

                is Session.NoServer -> ServerScreen(onChosen = vm::useServer)

                // ⚠️ **Before the login screen, not instead of an error on it.** A
                // launch with no network used to land on the password field, where
                // the right password fails and the app blames the person for a
                // network they cannot see.
                is Session.Offline -> OfflineScreen(s.address) { vm.probe() }

                is Session.SignedOut -> LoginScreen(
                    address = s.address,
                    onSignIn = { u, p -> vm.signIn(s.address, u, p) },
                    onForget = vm::forgetServer,
                )

                is Session.Ready -> {
                    val o = open
                    if (o != null) {
                        CheckScreen(
                            api = app.api, outbox = app.outbox,
                            checkId = o.checkId, branchId = o.branchId,
                            bottomInset = bottomInset, onBack = { open = null },
                        )
                    } else {
                        Column(Modifier.fillMaxSize()) {
                            Box(Modifier.weight(1f)) {
                                when (tab) {
                                    "floor" -> FloorScreen(app.api, tabsInset, s.shiftOpen) { id, br ->
                                        open = Open(id, br.ifEmpty { branchId })
                                    }
                                    // ⚠️ The session is re-read when a shift is
                                    // punched, or the room goes on refusing a
                                    // waiter who has just clocked in.
                                    "profile" -> ProfileScreen(
                                        app.api, s.staff, tabsInset, onShiftChanged = vm::refreshShift,
                                    )
                                    else -> SettingsScreen(
                                        api = app.api, staff = s.staff, address = s.address,
                                        bottomInset = tabsInset,
                                        pushState = push.state,
                                        onRetryPush = push.retry,
                                        // ⚠️ The phone is dropped **before** the
                                        // token is cleared, or the request goes
                                        // out unauthenticated and the row stays.
                                        onSignOut = {
                                            scope.launch { push.forget(); vm.signOut(s.address) }
                                        },
                                        onForgetServer = {
                                            scope.launch { push.forget(); vm.forgetServer() }
                                        },
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }

        // ⚠️ **What the phone is still holding, said on every screen.** An app
        // with four dishes queued looks exactly like an app that sent them, and
        // the difference reaches the guest. Above the tab bar rather than inside
        // the check: the waiter walks away from the table while it is still
        // waiting, and that is precisely when they need to know.
        val queued by app.outbox.pending.collectAsState()
        if (ready != null && queued > 0) {
            QueuedBanner(
                queued,
                Modifier.align(Alignment.BottomCenter)
                    .padding(bottom = if (open == null) 84.dp else 8.dp)
                    .padding(bottom = bottomInset.calculateBottomPadding()),
            )
        }

        // The bar belongs to the tabs, not to the check: a screen you came *into*
        // does not offer three ways out of itself.
        if (ready != null && open == null) {
            GlassTabBar(
                items = listOf(
                    TabItem("floor", Icons.Rounded.GridView, t.tabs.floor),
                    TabItem("profile", Icons.Rounded.Person, t.tabs.profile),
                    TabItem("settings", Icons.Rounded.Settings, t.tabs.settings),
                ),
                selected = tab,
                onSelect = { tab = it },
                modifier = Modifier.align(Alignment.BottomCenter),
            )
        }
    }
}

/** The queue, in one line. ⚠️ Not an error colour: nothing was lost, and nothing
 *  needs retyping — which is the opposite of what the old "could not add" told
 *  people, and the reason they retyped it. */
@Composable
private fun QueuedBanner(count: Int, modifier: Modifier = Modifier) {
    val c = KeelTheme.colors
    Row(
        modifier
            .padding(horizontal = 16.dp)
            .glass(c, RoundedCornerShape(999.dp), strong = true)
            .padding(horizontal = 14.dp, vertical = 9.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Icon(
            Icons.Rounded.CloudUpload, null, tint = c.accent,
            modifier = Modifier.size(16.dp),
        )
        Text(
            t.outbox.waiting(count),
            color = c.inkSoft,
            style = MaterialTheme.typography.labelMedium,
        )
    }
}
