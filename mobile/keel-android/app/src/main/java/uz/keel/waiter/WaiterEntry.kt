package uz.keel.waiter

import androidx.activity.compose.BackHandler
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.Crossfade
import androidx.compose.animation.core.tween
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
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
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewmodel.compose.viewModel
import uz.keel.app.RoleExit
import uz.keel.app.push.KeelPush
import uz.keel.design.*
import uz.keel.waiter.data.Check
import uz.keel.waiter.data.StaffReport
import uz.keel.waiter.push.PushRegistration
import uz.keel.waiter.push.forRole
import uz.keel.waiter.ui.screens.CheckScreen
import uz.keel.waiter.ui.screens.FloorScreen
import uz.keel.waiter.ui.screens.OfflineScreen
import uz.keel.waiter.ui.screens.ProfileScreen
import uz.keel.waiter.ui.screens.SettingsScreen

// Keel → the floor.
//
// ⚠️ **One level of navigation, held here, rather than a router.** The check is
// the only place a tab leads to, and the phone's back button has nothing else to
// mean. A navigation library at this size is a dependency carrying one decision.

/** The waiter's half of Keel: the room, the check, the shift.
 *
 *  ⚠️ **Everything Keel Waiter's `MainActivity` did except the getting in.**
 *  The address and the password are asked once, by Keel, for every role; what
 *  is left is this role's own words and its own screens. The words are
 *  provided here and not by the activity, because the activity serves four
 *  dictionaries and only this composable knows which one it is in.
 *
 *  @param pendingCheckId a table from a tapped notification — the whole message
 *  is "go to the pass", so it opens the table rather than the room. */
@Composable
fun WaiterEntry(
    graph: WaiterGraph,
    push: KeelPush,
    exit: RoleExit,
    pendingCheckId: String?,
    onConsumed: () -> Unit,
) {
    CompositionLocalProvider(
        LocalPrefs provides graph.prefs,
        LocalLangHost provides graph.prefs.langHost(),
        LocalWords provides DesignWords(
            ok = graph.prefs.dict.notice.ok,
            retry = graph.prefs.dict.common.retry,
            loading = graph.prefs.dict.common.loading,
        ),
    ) {
        Root(graph, push.forRole(), exit, pendingCheckId, onConsumed)
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
private fun Root(
    app: WaiterGraph,
    push: PushRegistration,
    exit: RoleExit,
    pendingCheckId: String?,
    onConsumed: () -> Unit,
) {
    val c = KeelTheme.colors

    // ⚠️ **Keyed on the token.** The activity outlives this screen in Keel —
    // somebody goes to the owner's numbers and comes back — and an unkeyed model
    // would hand a new sign-in the last person's session.
    val vm: SessionViewModel = viewModel(
        key = "waiter:" + app.tokens.staffToken.hashCode(),
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

    // ⚠️ **A refused token leaves the role, it does not draw a login here.**
    // Keel has one sign-in for every role, and which screen comes after it
    // depends on what else is signed in on this phone.
    val refused = session is Session.SignedOut || session is Session.NoServer
    LaunchedEffect(refused) { if (refused) exit.expired() }

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

                is Screen.NoServer -> Box(Modifier.fillMaxSize())

                // ⚠️ **Before the login screen, not instead of an error on it.** A
                // launch with no network used to land on the password field, where
                // the right password fails and the app blames the person for a
                // network they cannot see.
                is Screen.Offline -> OfflineScreen(sc.address) { vm.probe() }

                // Drawn for the frames before `exit.expired` takes over.
                is Screen.SignedOut -> Box(Modifier.fillMaxSize())

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
                                // ⚠️ Keel drops the push row **before** the
                                // token — see `RoleExit.signOut`.
                                onSignOut = exit.signOut,
                                onForgetServer = exit.leaveRestaurant,
                                top = exit.card,
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
