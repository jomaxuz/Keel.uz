package uz.keel.owner

import androidx.activity.compose.BackHandler
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
import androidx.compose.material.icons.rounded.Insights
import androidx.compose.material.icons.rounded.NotificationsActive
import androidx.compose.material.icons.rounded.ReceiptLong
import androidx.compose.material.icons.rounded.Settings
import androidx.compose.material.icons.rounded.StarOutline
import androidx.compose.material.icons.rounded.Today
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.runtime.Composable
import uz.keel.app.RoleExit
import uz.keel.app.push.KeelPush
import uz.keel.owner.push.PushRegistration
import uz.keel.owner.push.forRole
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
import uz.keel.design.GlassTabBar
import uz.keel.design.KeelTheme
import uz.keel.design.LocalLangHost
import uz.keel.design.LocalNotice
import uz.keel.design.LocalWords
import uz.keel.design.TabItem
import uz.keel.owner.ui.screens.AlertsScreen
import uz.keel.owner.ui.screens.CheckScreen
import uz.keel.owner.ui.screens.FeedbackScreen
import uz.keel.owner.ui.screens.MoneyScreen
import uz.keel.owner.ui.screens.OfflineScreen
import uz.keel.owner.ui.screens.OrdersScreen
import uz.keel.owner.ui.screens.ReportsScreen
import uz.keel.owner.ui.screens.SettingsScreen
import uz.keel.owner.ui.screens.SupportScreen
import uz.keel.owner.ui.screens.TodayScreen

// Keel Owner.
//
// ⚠️ **This is not the panel on a phone.** The panel is where somebody sits down
// and decides: the menu, a price, a roster, a campaign. Those stay on a keyboard.
// An owner uses a phone to *watch and answer* — how is today going, what is being
// asked for right now, accept this order, why was 400 000 taken off table six.
// Every screen here answers one of those in three seconds.
//
// ⚠️ **One level of navigation, held here, rather than a router.** A tab leads
// to at most one screen over it (help, a check, where the money is), and the
// phone's back button has nothing else to mean. A navigation library at this
// size is a dependency carrying one decision.


/** The owner's half of Keel.
 *
 *  ⚠️ **Everything Keel Owner's `MainActivity` did except the getting in.**
 *  The address and the password are asked once, by Keel, for every role; what
 *  is left is this role's own words and its own screens. The words are
 *  provided here and not by the activity, because the activity serves four
 *  dictionaries and only this composable knows which one it is in.
 *
 *  @param pendingTab a tab from a tapped notification (`data.tab`): an alert opens on the alerts, not on today. */
@Composable
fun OwnerEntry(
    graph: OwnerGraph,
    push: KeelPush,
    exit: RoleExit,
    pendingTab: String?,
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
        Root(graph, push.forRole(), exit, pendingTab, onConsumed)
    }
}

@Composable
private fun Root(
    app: OwnerGraph,
    push: PushRegistration,
    exit: RoleExit,
    pendingTab: String?,
    onConsumed: () -> Unit,
) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()

    // ⚠️ **Keyed on the token** — the activity outlives this screen in Keel,
    // and an unkeyed model would hand a new sign-in the last one's session.
    val vm: SessionViewModel = viewModel(
        key = "owner:" + app.tokens.read(uz.keel.design.TokenStore.ADMIN_TOKEN).hashCode(),
        factory = object : ViewModelProvider.Factory {
            @Suppress("UNCHECKED_CAST")
            override fun <T : ViewModel> create(modelClass: Class<T>): T =
                SessionViewModel(app.api, app.tokens) as T
        },
    )
    val session by vm.state.collectAsState()
    val ready = session as? Session.Ready

    var tab by remember { mutableStateOf("today") }
    var overlay by remember { mutableStateOf<Overlay?>(null) }



    // ⚠️ **A refused token leaves the role; it does not draw a login here.**
    // Keel has one sign-in for every role, and what follows it depends on what
    // else is signed in on this phone.
    val refused = session is Session.SignedOut || session is Session.NoServer
    LaunchedEffect(refused) { if (refused) exit.expired() }

    // A notification tap opens the screen it is about: somebody reading "a
    // discount was given" wants the list, not the takings.
    LaunchedEffect(pendingTab, ready) {
        val to = pendingTab ?: return@LaunchedEffect
        if (ready == null) return@LaunchedEffect
        tab = to
        overlay = null
        onConsumed()
    }

    // ⚠️ Only while something is open over a tab, so back from a tab still
    // leaves the app — an owner who cannot put their phone away is an owner
    // fighting it.
    BackHandler(enabled = overlay != null) { overlay = null }

    val bottomInset = WindowInsets.navigationBars.asPaddingValues()
    // ⚠️ The bar's own height *plus* the system's strip: the bar applies the
    // inset to itself, so a list padded only by the system's still ends with its
    // last row behind the glass.
    val tabsInset = PaddingValues(bottom = 76.dp + bottomInset.calculateBottomPadding())

    Box(Modifier.fillMaxSize()) {
        AnimatedContent(
            targetState = Triple(session::class, tab, overlay),
            transitionSpec = { fadeIn() togetherWith fadeOut() },
            label = "root",
        ) { _ ->
            when (val s = session) {
                is Session.Loading -> Box(Modifier.fillMaxSize(), Alignment.Center) {
                    CircularProgressIndicator(color = c.accent)
                }

                is Session.NoServer -> Box(Modifier.fillMaxSize())

                // ⚠️ Before the login screen, not instead of an error on it: a
                // launch with no network used to land on the password field,
                // where the right password fails and the app blames the person
                // for a network they cannot see.
                is Session.Offline -> OfflineScreen(s.address) { vm.probe() }

                is Session.SignedOut -> Box(Modifier.fillMaxSize())

                is Session.Ready -> {
                    val over = overlay
                    if (over != null) {
                        when (over) {
                            Overlay.Help -> SupportScreen(app.api, bottomInset) { overlay = null }
                            Overlay.Money -> MoneyScreen(app.api, bottomInset) { overlay = null }
                            is Overlay.Check -> CheckScreen(app.api, over.id, bottomInset) { overlay = null }
                        }
                    } else {
                        Column(Modifier.fillMaxSize()) {
                            Box(Modifier.weight(1f)) {
                                when (tab) {
                                    "today" -> TodayScreen(app.api, s.branches, tabsInset)
                                    "alerts" -> AlertsScreen(app.api, tabsInset) { overlay = Overlay.Check(it) }
                                    "orders" -> OrdersScreen(app.api, tabsInset)
                                    "feedback" -> FeedbackScreen(app.api, tabsInset)
                                    "reports" -> ReportsScreen(app.api, tabsInset) { overlay = Overlay.Money }
                                    else -> SettingsScreen(
                                        api = app.api,
                                        user = s.user,
                                        address = s.address,
                                        branchName = s.branches
                                            .firstOrNull { it.id == app.prefs.branch.value }
                                            ?.name.orEmpty(),
                                        bottomInset = tabsInset,
                                        pushState = push.state,
                                        pushDetail = push.detail,
                                        onRetryPush = push.retry,
                                        onOpenHelp = { overlay = Overlay.Help },
                                        // ⚠️ The phone is dropped **before** the
                                        // token is cleared, or the request goes
                                        // out unauthenticated and the row stays
                                        // — sending tomorrow's alerts to a phone
                                        // that has changed hands.
                                        onSignOut = exit.signOut,
                                        onForgetServer = exit.leaveRestaurant,
                                        top = exit.card,
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }

        // The bar belongs to the tabs, not to a screen you came *into*.
        if (ready != null && overlay == null) {
            GlassTabBar(
                items = listOf(
                    TabItem("today", Icons.Rounded.Today, t.tabs.today),
                    TabItem("alerts", Icons.Rounded.NotificationsActive, t.tabs.alerts),
                    TabItem("orders", Icons.Rounded.ReceiptLong, t.tabs.orders),
                    TabItem("feedback", Icons.Rounded.StarOutline, t.tabs.feedback),
                    TabItem("reports", Icons.Rounded.Insights, t.tabs.reports),
                    TabItem("settings", Icons.Rounded.Settings, t.tabs.settings),
                ),
                selected = tab,
                onSelect = { tab = it },
                modifier = Modifier.align(Alignment.BottomCenter),
            )
        }
    }
}

/** What sits over the tabs, when anything does. */
private sealed interface Overlay {
    data object Help : Overlay
    data object Money : Overlay
    data class Check(val id: String) : Overlay
}
