package uz.keel.waiter

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.BackHandler
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.Crossfade
import androidx.compose.animation.core.tween
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
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
import uz.keel.waiter.data.Check
import uz.keel.waiter.data.StaffReport
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

/** A table somebody tapped.
 *
 *  @param preview what the room already knew about this check. ⚠️ Drawn at once
 *  while the check is asked for again: the floor's list carries every line, so
 *  a table that opened on a spinner was waiting for something already on the
 *  phone. */
private data class Open(val checkId: String, val branchId: String, val preview: Check? = null)

/** What the root is showing.
 *
 *  ⚠️ **The animation is keyed on this and draws from it, and that was a real
 *  bug.** The root used to animate on a key but draw from the *current* session
 *  inside both halves of the cross-fade — so opening a table composed two check
 *  screens at once: both loaded the check, the menu and the room (every open
 *  was six requests instead of three), and when the outgoing copy was disposed
 *  it released the hold on the very table the waiter had just opened. */
private sealed interface Screen {
    val key: String

    data object Loading : Screen {
        override val key: String get() = "loading"
    }
    data object NoServer : Screen {
        override val key: String get() = "noServer"
    }
    data class Offline(val address: String) : Screen {
        override val key: String get() = "offline"
    }
    data class SignedOut(val address: String) : Screen {
        override val key: String get() = "signedOut"
    }
    data class Main(val ready: Session.Ready) : Screen {
        override val key: String get() = "main"
    }
    data class Table(val ready: Session.Ready, val open: Open) : Screen {
        override val key: String get() = "check:" + open.checkId
    }
}

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
    // Per person: a new sign-in must not open on the last one's hours.
    val reportCache = remember(ready?.staff?.id) { mutableStateOf<StaffReport?>(null) }

    // ⚠️ Registered once signed in, not at launch: a permission prompt on the
    // first screen is asked before anybody knows what the app is for.
    val push = rememberPushRegistration(app.api, ready != null, app.prefs.lang.value.code)

    // ⚠️ **The room belongs to whoever is signed in.** It is held by the process
    // now (see FloorStore), so signing out has to let go of it — or the next
    // person on this phone sees the last one's tables for a second, and on a
    // different branch that second is somebody else's room.
    val signedIn = ready != null
    LaunchedEffect(signedIn) {
        if (!signedIn) {
            app.floor.clear()
            app.menu.clear()
            open = null
            tab = "floor"
        }
    }

    // A notification tap opens the table rather than the room: somebody reading
    // it on the move has already decided where they are going.
    LaunchedEffect(pendingCheckId, ready) {
        val id = pendingCheckId ?: return@LaunchedEffect
        if (ready == null) return@LaunchedEffect
        open = Open(id, branchId, app.floor.checks.firstOrNull { it.id == id })
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

    val screen: Screen = when (val s = session) {
        is Session.Loading -> Screen.Loading
        is Session.NoServer -> Screen.NoServer
        is Session.Offline -> Screen.Offline(s.address)
        is Session.SignedOut -> Screen.SignedOut(s.address)
        is Session.Ready -> open?.let { Screen.Table(s, it) } ?: Screen.Main(s)
    }

    Box(Modifier.fillMaxSize()) {
        AnimatedContent(
            targetState = screen,
            contentKey = { it.key },
            transitionSpec = {
                // A table slides in from the side it will leave by, so going
                // back reads as going back rather than as a third screen.
                val into = targetState is Screen.Table && initialState !is Screen.Table
                val back = initialState is Screen.Table && targetState !is Screen.Table
                when {
                    into -> (slideInHorizontally(tween(260)) { it / 4 } + fadeIn(tween(220)))
                        .togetherWith(fadeOut(tween(140)))
                    back -> fadeIn(tween(220))
                        .togetherWith(slideOutHorizontally(tween(220)) { it / 4 } + fadeOut(tween(180)))
                    else -> fadeIn(tween(220)).togetherWith(fadeOut(tween(160)))
                }
            },
            label = "root",
        ) { sc ->
            when (sc) {
                is Screen.Loading -> Box(Modifier.fillMaxSize(), Alignment.Center) {
                    CircularProgressIndicator(color = c.accent)
                }

                is Screen.NoServer -> ServerScreen(onChosen = vm::useServer)

                // ⚠️ **Before the login screen, not instead of an error on it.** A
                // launch with no network used to land on the password field, where
                // the right password fails and the app blames the person for a
                // network they cannot see.
                is Screen.Offline -> OfflineScreen(sc.address) { vm.probe() }

                is Screen.SignedOut -> LoginScreen(
                    address = sc.address,
                    onSignIn = { u, p -> vm.signIn(sc.address, u, p) },
                    onForget = vm::forgetServer,
                )

                is Screen.Table -> CheckScreen(
                    api = app.api, outbox = app.outbox, floor = app.floor, menuCache = app.menu,
                    checkId = sc.open.checkId, branchId = sc.open.branchId,
                    preview = sc.open.preview,
                    bottomInset = bottomInset, onBack = { open = null },
                )

                is Screen.Main -> {
                    // ⚠️ The live session when there is one: a shift punched on
                    // the Profile tab re-reads it, and the room has to stop
                    // refusing at once. The snapshot only draws the frames of a
                    // sign-out's fade.
                    val s = ready ?: sc.ready
                    // ⚠️ **A cross-fade between tabs that keeps nothing, over data
                    // that is kept elsewhere.** The room lives in FloorStore, so
                    // coming back to it is instant instead of a spinner.
                    Crossfade(targetState = tab, animationSpec = tween(180), label = "tab") { tb ->
                        when (tb) {
                            "floor" -> FloorScreen(
                                api = app.api, store = app.floor, menu = app.menu,
                                staffId = s.staff.id,
                                bottomInset = tabsInset, shiftOpen = s.shiftOpen,
                            ) { id, br, preview ->
                                open = Open(id, br.ifEmpty { s.staff.branchId ?: "" }, preview)
                            }
                            // ⚠️ The session is re-read when a shift is
                            // punched, or the room goes on refusing a
                            // waiter who has just clocked in.
                            "profile" -> ProfileScreen(
                                app.api, s.staff, tabsInset, onShiftChanged = vm::refreshShift,
                                cached = reportCache,
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

        // ⚠️ **What the phone is still holding, said on every screen.** An app
        // with four dishes queued looks exactly like an app that sent them, and
        // the difference reaches the guest. Above the tab bar: the waiter walks
        // away from the table while it is still waiting, and that is precisely
        // when they need to know. ⚠️ The check screen says it in its own bottom
        // panel instead — drawn here it sat on top of "send to the kitchen".
        val queued by app.outbox.pending.collectAsState()
        if (ready != null && open == null && queued > 0) {
            QueuedBanner(
                queued,
                Modifier.align(Alignment.BottomCenter)
                    .padding(bottom = 84.dp)
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
